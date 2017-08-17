package blob

import (
	"reflect"
	"testing"
)

func TestNewCreateCommand(t *testing.T) {
	store := NewInMemoryEventStore()
	repo := NewAggregateRepository(store)

	blob, err := repo.Process(CreateCommand("1", "text", []byte("hello")))
	if err != nil {
		t.Fatal(err)
	}

	newFetchBlob, err := repo.Find("1")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(blob, newFetchBlob) {
		t.Fatalf("Expeced %#v but was %#v", blob, newFetchBlob)
	}

	t.Logf("%T", blob)
}
