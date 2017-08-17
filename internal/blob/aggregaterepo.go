package blob

import "fmt"

type AggregateRepository struct {
	store EventStore
}

func NewAggregateRepository(store EventStore) AggregateRepository {
	return AggregateRepository{store}
}

func (ar AggregateRepository) Find(id ID) (Blob, error) {
	events, err := ar.store.Find(id)
	if err != nil {
		return Blob{}, fmt.Errorf("cannot find Blob events from store for ID %s: %v", id, err)
	}
	var blob Blob
	for _, event := range events {
		blob = event.Apply(blob)
	}
	return blob, nil
}

func (ar AggregateRepository) Process(cmd Command) (Blob, error) {
	blob, err := ar.Find(cmd.ID)
	if err != nil {
		return Blob{}, NewCommandError(cmd, err.Error())
	}

	newEvents, err := cmd.GenerateEvents(blob)
	if err != nil {
		return Blob{}, err
	}

	if err := ar.store.Persist(cmd.ID, newEvents); err != nil {
		return Blob{}, NewCommandError(cmd, "failed to persist new events: %v", err)
	}

	for _, newEvent := range newEvents {
		blob = newEvent.Apply(blob)
	}

	return blob, nil
}
