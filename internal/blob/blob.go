package blob

type ID string

type Tags map[string]string

type BlobType string

func (bt BlobType) String() string {
	return string(bt)
}

type Blob struct {
	ID
	BlobType
	Data     []byte
	Deleted  bool
	Sequence uint64
	Tags
}
