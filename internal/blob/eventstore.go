package blob

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

type EventStore interface {
	Find(ID) ([]EventWithMetadata, error)
	Persist(ID, []EventWithMetadata) error
}

type InMemoryEventStore struct {
	mux        *sync.Mutex
	eventStore map[ID][]EventWithMetadata
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{mux: new(sync.Mutex), eventStore: make(map[ID][]EventWithMetadata)}
}

func (i *InMemoryEventStore) Find(id ID) ([]EventWithMetadata, error) {
	i.mux.Lock()
	defer i.mux.Unlock()
	return i.eventStore[id], nil
}

func (i *InMemoryEventStore) Persist(id ID, events []EventWithMetadata) error {
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

func (l *LocalFileSystemEventStore) Find(id ID) ([]EventWithMetadata, error) {
	l.mux.Lock()
	defer l.mux.Unlock()

	var events []EventWithMetadata
	dirPath := path.Join(l.baseDirectory, id.String())
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return nil, nil
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

func (l *LocalFileSystemEventStore) Persist(id ID, events []EventWithMetadata) error {
	l.mux.Lock()
	defer l.mux.Unlock()

	for _, event := range events {
		if event.ID != id {
			return fmt.Errorf("cannot persist event %v as it does not have a matching aggregateID %v", event, id)
		}
	}

	dirPath := path.Join(l.baseDirectory, id.String())
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("cannot create directory for persisting events: %v", err)
	}

	for _, event := range events {
		data, err := marshal(event)
		if err != nil {
			return fmt.Errorf("cannot marshal event to persist %v: %v", event, err)
		}
		err = ioutil.WriteFile(path.Join(dirPath, strconv.FormatUint(event.Sequence, 10)), data, 0755)
		if err != nil {
			return fmt.Errorf("cannot persist event %v: %v", event, err)
		}
	}
	return nil
}

func marshal(event EventWithMetadata) ([]byte, error) {
	persistedEvent := localFileSystemPersistedEvent{ID: event.ID, Sequence: event.Sequence}

	switch ev := event.Event.(type) {
	case CreatedEvent:
		persistedEvent.EventType = "CE"
		persistedEvent.BlobType = ev.BlobType
		persistedEvent.Data = ev.Data
	case DataUpdatedEvent:
		persistedEvent.EventType = "DUE"
		persistedEvent.DataUpdated = ev.Data
	case TagsAddedEvent:
		persistedEvent.EventType = "TAE"
		persistedEvent.TagsAdded = Tags(ev)
	case TagsUpdatedEvent:
		persistedEvent.EventType = "TUE"
		persistedEvent.TagsUpdated = Tags(ev)
	case TagsDeletedEvent:
		persistedEvent.EventType = "TDE"
		persistedEvent.TagsDeleted = []string(ev)
	case DeletedEvent:
		persistedEvent.EventType = "DE"
	case RestoredEvent:
		persistedEvent.EventType = "RE"
	}
	return json.Marshal(persistedEvent)
}

func unmarshal(data []byte) (EventWithMetadata, error) {
	var persistedEvent localFileSystemPersistedEvent
	if err := json.Unmarshal(data, &persistedEvent); err != nil {
		return EventWithMetadata{}, err
	}

	e := EventWithMetadata{ID: persistedEvent.ID, Sequence: persistedEvent.Sequence}
	switch persistedEvent.EventType {
	case "CE":
		e.Event = CreatedEvent{
			BlobType: persistedEvent.BlobType,
			Data:     persistedEvent.Data,
		}
	case "DUE":
		e.Event = DataUpdatedEvent{
			Data: persistedEvent.DataUpdated,
		}
	case "TAE":
		e.Event = TagsAddedEvent(persistedEvent.TagsAdded)
	case "TUE":
		e.Event = TagsUpdatedEvent(persistedEvent.TagsUpdated)
	case "TDE":
		e.Event = TagsDeletedEvent(persistedEvent.TagsDeleted)
	case "DE":
		e.Event = DeletedEvent{}
	case "RE":
		e.Event = RestoredEvent{}
	}

	return e, nil
}

type localFileSystemPersistedEvent struct {
	ID          `json:"id"`
	Sequence    uint64 `json:"sequence"`
	EventType   string `json:"eventType"`
	BlobType    `json:"blobType"`
	Data        []byte   `json:"data"`
	DataUpdated []byte   `json:"dataUpdated"`
	TagsAdded   Tags     `json:"tagsAdded"`
	TagsUpdated Tags     `json:"tagsUpdated"`
	TagsDeleted []string `json:"tagsDeleted"`
}
