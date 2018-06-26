package blob

import (
	"hash"
	"io"
)

type Chunk struct {
	Hash []byte
	Data []byte
}

func NewChunk(data []byte, newHashFn func() hash.Hash) Chunk {
	h := newHashFn()
	h.Write(data)
	return Chunk{h.Sum(nil), data}
}

type Chunks struct {
	Chunks []Chunk
	Hash   []byte
}

func NewChunks(rc io.ReadCloser, chunkSize uint32, newHashFn func() hash.Hash) (chunks Chunks, err error) {
	defer rc.Close()
	h := newHashFn()
	for err == nil {
		buf := make([]byte, chunkSize)
		var n uint32
		for n < chunkSize && err == nil {
			var nn int
			nn, err = rc.Read(buf[n:])
			n += uint32(nn)
		}
		if n > 0 && (err == nil || err == io.EOF) {
			h.Write(buf)
			chunks.Chunks = append(chunks.Chunks, NewChunk(buf, newHashFn))
		}
	}

	if err != nil && err != io.EOF {
		return Chunks{}, err
	}

	chunks.Hash = h.Sum(nil)
	return chunks, nil
}
