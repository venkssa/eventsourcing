package blob

import "fmt"

type AggregateRepository struct {
	store EventStore
}

func NewAggregateRepository(store EventStore) AggregateRepository {
	return AggregateRepository{store}
}

// Find finds an aggregate for the given ID or returns a error if the aggregate cannot be found.
func (ar AggregateRepository) Find(id ID) (Blob, error) {
	events, err := ar.store.Find(id)
	if err != nil {
		return Blob{}, fmt.Errorf("cannot find aggregate for ID %s: %v", id, err)
	}
	return events.Apply(Blob{}), nil
}

// Process applies the command to the aggregate to generate events, persist the newly generated events,
// apply the new events to the aggrgate and return the updated aggregate or error. error is a CommandError.
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

	return newEvents.Apply(blob), nil
}
