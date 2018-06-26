package blob

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/boltdb/bolt"

	"github.com/pkg/errors"
)

// EventStore can store and retrieve events for aggregate ID.
type EventStore interface {
	//Find events for the aggregate ID.
	// If events cannot be found for the aggregate ID we return an error with IsMissingAggregate() true.
	Find(context.Context, ID) (EventWithMetadataSlice, error)

	// Persist events for an aggregate ID.
	Persist(context.Context, ID, EventWithMetadataSlice) error
}

type eventStoreError struct {
	isMissingAggregate bool
	error
}

func (e eventStoreError) IsMissingAggregate() bool {
	return e.isMissingAggregate
}

type InMemoryEventStore struct {
	mux        *sync.Mutex
	eventStore map[ID]EventWithMetadataSlice
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{mux: new(sync.Mutex), eventStore: make(map[ID]EventWithMetadataSlice)}
}

func (i *InMemoryEventStore) Find(ctx context.Context, id ID) (EventWithMetadataSlice, error) {
	i.mux.Lock()
	defer i.mux.Unlock()
	return i.eventStore[id], nil
}

func (i *InMemoryEventStore) Persist(ctx context.Context, id ID, events EventWithMetadataSlice) error {
	i.mux.Lock()
	defer i.mux.Unlock()

	seenSequences := make(map[uint64]bool)
	for _, existingEvents := range i.eventStore[id] {
		seenSequences[existingEvents.Sequence] = true
	}

	for _, event := range events {
		if event.ID != id {
			return fmt.Errorf("cannot persist event %v as it does not have a matching aggregateID %v", event, id)
		}
		if seenSequences[event.Sequence] {
			return fmt.Errorf("cannot persist event %v as an event with the same sequence already exists", event)
		}
	}

	i.eventStore[id] = append(i.eventStore[id], events...)
	return nil
}

type LocalFileSystemEventStore struct {
	mux           *sync.Mutex
	baseDirectory string
}

func NewLocalFileSystemEventStore(baseDirectory string) *LocalFileSystemEventStore {
	return &LocalFileSystemEventStore{mux: new(sync.Mutex), baseDirectory: baseDirectory}
}

func (l *LocalFileSystemEventStore) Find(ctx context.Context, id ID) (EventWithMetadataSlice, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	var events EventWithMetadataSlice
	dirPath := path.Join(l.baseDirectory, id.String())
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, eventStoreError{
			isMissingAggregate: true,
			error:              fmt.Errorf("cannot find events directory for id %v in eventstore", id)}
	}
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		event, err := unmarshal(data)
		if err != nil {
			return err
		}
		events = append(events, event)
		return nil
	})

	sort.Slice(events, func(i, j int) bool {
		return events[i].Sequence < events[j].Sequence
	})

	return events, err
}

func (l *LocalFileSystemEventStore) Persist(ctx context.Context, id ID, events EventWithMetadataSlice) error {
	l.mux.Lock()
	defer l.mux.Unlock()

	for _, event := range events {
		if event.ID != id {
			return fmt.Errorf("cannot persist event %v as it does not have a matching aggregateID %v", event, id)
		}
	}

	dirPath := path.Join(l.baseDirectory, id.String())
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return errors.Wrap(err, "cannot create directory for persisting events")
	}

	for _, event := range events {
		data, err := marshal(event)
		if err != nil {
			return errors.Wrapf(err, "cannot marshal event %v for persisting", event)
		}
		err = ioutil.WriteFile(path.Join(dirPath, strconv.FormatUint(event.Sequence, 10)), data, 0755)
		if err != nil {
			return errors.Wrapf(err, "cannot persist event %v: %v", event)
		}
	}
	return nil
}

func marshal(event EventWithMetadata) ([]byte, error) {
	var eventType string
	switch event.Event.(type) {
	case CreatedEvent:
		eventType = "CE"
	case DataUpdatedEvent:
		eventType = "DUE"
	case TagsAddedEvent:
		eventType = "TAE"
	case TagsUpdatedEvent:
		eventType = "TUE"
	case TagsDeletedEvent:
		eventType = "TDE"
	case DeletedEvent:
		eventType = "DE"
	case RestoredEvent:
		eventType = "RE"
	}
	marshaledEvent, err := json.Marshal(event.Event)
	if err != nil {
		return nil, err
	}
	return json.Marshal(persistableEvent{event.ID, event.Sequence, eventType, marshaledEvent})
}

func unmarshal(data []byte) (EventWithMetadata, error) {
	var pe persistableEvent
	if err := json.Unmarshal(data, &pe); err != nil {
		return EventWithMetadata{}, err
	}
	var event Event
	switch pe.EventType {
	case "CE":
		event = &CreatedEvent{}
	case "DUE":
		event = &DataUpdatedEvent{}
	case "TAE":
		event = &TagsAddedEvent{}
	case "TUE":
		event = &TagsUpdatedEvent{}
	case "TDE":
		event = &TagsDeletedEvent{}
	case "DE":
		event = &DeletedEvent{}
	case "RE":
		event = &RestoredEvent{}
	}

	err := json.Unmarshal(pe.MarshaledEvent, event)
	return EventWithMetadata{ID: pe.ID, Sequence: pe.Sequence, Event: event}, err
}

type persistableEvent struct {
	ID             `json:"id"`
	Sequence       uint64          `json:"sequence"`
	EventType      string          `json:"eventType"`
	MarshaledEvent json.RawMessage `json:"marshaledEvent"`
}

type BoltDBEventStore struct {
	db *bolt.DB
}

var _ EventStore = &BoltDBEventStore{}

func NewBoltDBEventStore(storePath string) *BoltDBEventStore {
	db, err := bolt.Open(storePath, 0755, nil)
	if err != nil {
		panic(err)
	}
	return &BoltDBEventStore{db}
}

func (b *BoltDBEventStore) Find(ctx context.Context, id ID) (events EventWithMetadataSlice, err error) {
	err = b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(id))
		if bkt == nil {
			return eventStoreError{isMissingAggregate: true, error: fmt.Errorf("cannot find events bucket for id %v in eventstore", id)}
		}
		return bkt.ForEach(func(_ []byte, value []byte) error {
			event, err := unmarshal(value)
			if err != nil {
				return err
			}
			events = append(events, event)
			return nil
		})
	})
	return events, err
}

func (b *BoltDBEventStore) Persist(ctx context.Context, id ID, events EventWithMetadataSlice) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte(id))
		if err != nil {
			return errors.Wrapf(err, "cannot create bucket for %s", id)
		}
		for _, event := range events {
			eventBytes, err := marshal(event)
			if err != nil {
				return errors.Wrapf(err, "cannot marshal event %v for persisting", event)
			}
			if err := bkt.Put([]byte(strconv.FormatUint(event.Sequence, 10)), eventBytes); err != nil {
				return errors.Wrapf(err, "failed to persist event %v", event)
			}
		}
		return nil
	})
}
