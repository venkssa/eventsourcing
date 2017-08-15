package blob

type EventStore interface {
	Find(id string) ([]Event, error)
	Persist([]Event) error
}

type InMemoryEventStore map[string][]Event

func (i InMemoryEventStore) Find(ID string) ([]Event, error) {
	return i[ID], nil
}

func (i InMemoryEventStore) Persist(events []Event) error {
	id := events[0].AggregateID()
	i[id] = append(i[id], events...)
	return nil
}
