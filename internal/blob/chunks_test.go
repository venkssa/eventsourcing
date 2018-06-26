package blob

import "testing"

import "crypto/sha256"
import "bytes"
import "io/ioutil"

func TestChunk_Hash(t *testing.T) {
	hwB := []byte("helloworld")

	cs, err := NewChunks(ioutil.NopCloser(bytes.NewBuffer(hwB)), 5, sha256.New)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if len(cs.Chunks) != 2 {
		t.Errorf("Expected 2 chunks but got only %v", len(cs.Chunks))
	}

	h := sha256.New()
	h.Write(hwB[0:5])
	h.Write(hwB[5:])
	expectedHash := string(h.Sum(nil))
	actualHash := string(cs.Hash)

	if expectedHash != actualHash {
		t.Errorf("Expected chunks has %s but got %s", expectedHash, actualHash)
	}
}
