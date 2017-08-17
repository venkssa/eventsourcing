package blob

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestT(t *testing.T) {
	s := struct {
		V []byte
	}{
		V: []byte("abc hello world"),
	}
	buf := new(bytes.Buffer)
	t.Log(json.NewEncoder(buf).Encode(s))
	t.Log(buf.String())
}
