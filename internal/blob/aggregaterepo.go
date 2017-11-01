package blob

import (
	"context"

	"github.com/pkg/errors"
	"github.com/venkssa/eventsourcing/internal/platform"
)

type AggregateRepository struct {
	store EventStore
}

func NewAggregateRepository(store EventStore) AggregateRepository {
	return AggregateRepository{store}
}

// Find finds an aggregate for the given ID or returns a error if the aggregate cannot be found.
func (ar AggregateRepository) Find(ctx context.Context, id ID) (Blob, error) {
	events, err := ar.store.Find(ctx, id)
	if err != nil {
		return Blob{}, errors.Wrapf(err, "cannot find aggregate for ID %s", id)
	}
	return events.Apply(Blob{}), nil
}

// Process applies the command to the aggregate to generate events, persist the newly generated events,
// apply the new events to the aggrgate and return the updated aggregate or error. error is a CommandError.
func (ar AggregateRepository) Process(ctx context.Context, cmd Command) (Blob, error) {
	blob, err := ar.Find(ctx, cmd.ID)
	if err != nil && !platform.IsMissingAggregate(err) {
		return Blob{}, errors.Wrapf(err, "cannot process %v command with %v", cmd.CommandType(), cmd.ID)
	}

	newEvents, err := cmd.GenerateEvents(blob)
	if err != nil {
		return Blob{}, errors.Wrapf(err, "cannot generate events for %v command with %v", cmd.CommandType(), cmd.ID)
	}

	if err := ar.store.Persist(ctx, cmd.ID, newEvents); err != nil {
		return Blob{}, errors.Wrapf(err, "failed to persist new events for %v command with %v", cmd.CommandType(), cmd.ID)
	}

	return newEvents.Apply(blob), nil
}
