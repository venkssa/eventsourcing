package blob

type Event interface {
	AggregateID() string
	Apply(Blob) Blob
}

type CreatedEvent struct {
	ID
	BlobType
	Data     []byte
	Sequence uint64
}

func (c CreatedEvent) Apply(Blob) Blob {
	return Blob{ID: c.ID, BlobType: c.BlobType, Data: c.Data, Sequence: c.Sequence}
}

type UpdatedEvent struct {
	ID
	Data     []byte
	Sequence uint64
}

func (u UpdatedEvent) Apply(b Blob) Blob {
	b.Data = u.Data
	b.Sequence = u.Sequence
	return b
}

type DeletedEvent struct {
	ID
	Sequence uint64
}

func (d DeletedEvent) Apply(b Blob) Blob {
	b.Deleted = true
	b.Sequence = d.Sequence
	return b
}
