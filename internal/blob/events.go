package blob

type EventWithMetadata struct {
	ID
	Sequence uint64
	Event
}

func (e EventWithMetadata) Apply(b Blob) Blob {
	appliedBlob := e.Event.Apply(b)
	appliedBlob.ID = e.ID
	appliedBlob.Sequence = e.Sequence
	return appliedBlob
}

type EventWithMetadataSlice []EventWithMetadata

func (e EventWithMetadataSlice) Apply(b Blob) Blob {
	for _, event := range e {
		b = event.Apply(b)
	}
	return b
}

func wrap(aggregateID ID, sequence uint64, events ...Event) EventWithMetadataSlice {
	wrappedEvents := make(EventWithMetadataSlice, len(events))
	for i, event := range events {
		wrappedEvents[i] = EventWithMetadata{ID: aggregateID, Sequence: sequence, Event: event}
		sequence++
	}
	return wrappedEvents
}

type Event interface {
	Apply(Blob) Blob
}

type CreatedEvent struct {
	BlobType
	Data []byte
}

func (c CreatedEvent) Apply(Blob) Blob {
	return Blob{BlobType: c.BlobType, Data: c.Data}
}

type DataUpdatedEvent struct {
	Data []byte
}

func (u DataUpdatedEvent) Apply(b Blob) Blob {
	b.Data = u.Data
	return b
}

type TagsAddedEvent Tags

func (t TagsAddedEvent) Apply(b Blob) Blob {
	tags := make(Tags)
	for k, v := range b.Tags {
		tags[k] = v
	}
	for addedKey, addedValue := range t {
		tags[addedKey] = addedValue
	}
	b.Tags = tags
	return b
}

type TagsDeletedEvent []string

func (t TagsDeletedEvent) Apply(b Blob) Blob {
	tags := make(Tags)
	for k, v := range b.Tags {
		tags[k] = v
	}
	for _, deletedKey := range t {
		delete(tags, deletedKey)
	}
	b.Tags = tags
	return b
}

type TagsUpdatedEvent Tags

func (t TagsUpdatedEvent) Apply(b Blob) Blob {
	tags := make(Tags)
	for k, v := range b.Tags {
		tags[k] = v
	}
	for updatedKey, updatedValue := range t {
		tags[updatedKey] = updatedValue
	}
	b.Tags = tags
	return b
}

type DeletedEvent struct{}

func (d DeletedEvent) Apply(b Blob) Blob {
	b.Deleted = true
	return b
}

type RestoredEvent struct{}

func (r RestoredEvent) Apply(b Blob) Blob {
	b.Deleted = false
	return b
}
