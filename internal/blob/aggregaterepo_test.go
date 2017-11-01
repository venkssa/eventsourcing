package blob

import (
	"context"
	"reflect"
	"testing"
)

func TestNewCreateCommand(t *testing.T) {
	store := NewInMemoryEventStore()
	repo := NewAggregateRepository(store)

	blob, err := repo.Process(context.Background(), CreateCommand("1", "application/text", []byte("hello")))
	if err != nil {
		t.Fatal(err)
	}

	newFetchBlob, err := repo.Find(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(blob, newFetchBlob) {
		t.Fatalf("Expeced %#v but was %#v", blob, newFetchBlob)
	}

	t.Logf("%T", blob)
}
