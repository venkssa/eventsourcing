package blob

import (
	"errors"
	"reflect"
	"testing"
)

func TestUpdateTagsCommand(t *testing.T) {
	tests := map[string]struct {
		ID             ID
		AddOrUpdate    Tags
		Delete         []string
		Blob           Blob
		ExpectedEvents EventWithMetadataSlice
		ExpectedError  error
	}{
		"empty ID should be an error": {
			ID:            "",
			ExpectedError: errEmptyID},
		"ID does not match blob.ID should be an error": {
			ID:            "1",
			Blob:          Blob{ID: "2"},
			ExpectedError: errors.New("id 2 in blob does not match 1 in command")},
		"updating and deleting the same tag is an should be an error": {
			ID:            "1",
			AddOrUpdate:   Tags{"tag1": "value"},
			Delete:        []string{"tag1"},
			Blob:          Blob{ID: "1"},
			ExpectedError: errors.New("cannot delete a tag 'tag1' as it is being updated at the same time"),
		},
		"adding a new tag should create a TagsAddedEvent": {
			ID:             "1",
			AddOrUpdate:    Tags{"add1": "v1"},
			Blob:           Blob{ID: "1", Sequence: 1},
			ExpectedEvents: wrap("1", 2, TagsAddedEvent(Tags{"add1": "v1"})),
		},
		"adding multiple tags should create a TagsAddedEvent": {
			ID:             "1",
			AddOrUpdate:    Tags{"add1": "v1", "add2": "v2"},
			Blob:           Blob{ID: "1", Sequence: 1},
			ExpectedEvents: wrap("1", 2, TagsAddedEvent(Tags{"add1": "v1", "add2": "v2"})),
		},
		"updating an existing tag should create a TagsUpdatedEvent": {
			ID:             "1",
			AddOrUpdate:    Tags{"update1": "v2"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"update1": "v1"}},
			ExpectedEvents: wrap("1", 2, TagsUpdatedEvent(Tags{"update1": "v2"})),
		},
		"updating multiple existing tags should create a TagsUpdatedEvent": {
			ID:             "1",
			AddOrUpdate:    Tags{"update1": "v1", "update2": "v2"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"update1": "v", "update2": "v"}},
			ExpectedEvents: wrap("1", 2, TagsUpdatedEvent(Tags{"update1": "v1", "update2": "v2"})),
		},
		"updating tag with same value should not create any event": {
			ID:             "1",
			AddOrUpdate:    Tags{"update1": "v1"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"update1": "v1"}},
			ExpectedEvents: EventWithMetadataSlice{},
		},
		"deleting an existing tag should create a TagsDeletedEvent": {
			ID:             "1",
			Delete:         []string{"del1"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"del1": "v1"}},
			ExpectedEvents: wrap("1", 2, TagsDeletedEvent([]string{"del1"})),
		},
		"deleting multiple existing tags should be create a TagsDeletedEvent": {
			ID:             "1",
			Delete:         []string{"del1", "del2"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"del1": "v1", "del2": "v2"}},
			ExpectedEvents: wrap("1", 2, TagsDeletedEvent([]string{"del1", "del2"})),
		},
		"deleting non existant tags should not create an event": {
			ID:             "1",
			Delete:         []string{"nonexistant"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"t1": "v1"}},
			ExpectedEvents: EventWithMetadataSlice{},
		},
		"adding a new tag and updating a new tag should create a TagsAddedEvent & TagsUpdatedEvent": {
			ID:             "1",
			AddOrUpdate:    Tags{"add": "v1", "update": "v2"},
			Blob:           Blob{ID: "1", Sequence: 1, Tags: Tags{"update": "v"}},
			ExpectedEvents: wrap("1", 2, TagsUpdatedEvent(Tags{"update": "v2"}), TagsAddedEvent(Tags{"add": "v1"})),
		},
	}

	for testName, data := range tests {
		t.Run(testName, func(t *testing.T) {
			cmd := UpdateTagsCommand(data.ID, data.AddOrUpdate, data.Delete)
			events, err := cmd.GenerateEvents(data.Blob)

			assertError(t, err, data.ExpectedError)
			assertEvents(t, events, data.ExpectedEvents)
		})
	}
}

func assertError(t *testing.T, actual, expected error) {
	if actual == expected {
		return
	}
	if (actual == nil && expected != nil) || (actual != nil && expected == nil) || (actual.Error() != expected.Error()) {
		t.Fatalf("Expected error '%v' but got '%v'", expected, actual)
	}
}

func assertEvents(t *testing.T, actual, expected EventWithMetadataSlice) {
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected events %#v but got %#v", expected, actual)
	}
}
