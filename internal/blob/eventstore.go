package blob

import (
	"fmt"
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
