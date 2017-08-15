package blob

type ID string

func (id ID) AggregateID() string {
	return string(id)
}

type BlobType string

type Blob struct {
	ID
	BlobType
	Data     []byte
	Deleted  bool
	Sequence uint64
}
