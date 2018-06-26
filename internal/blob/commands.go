package blob

import (
	"fmt"
)

type Command struct {
	ID
	commandType    string
	eventGenerator func(Blob) EventWithMetadataSlice
	validator      func(Blob) error
}

func (c Command) GenerateEvents(b Blob) (EventWithMetadataSlice, error) {
	if err := c.validator(b); err != nil {
		return nil, err
	}
	return c.eventGenerator(b), nil
}

func (c Command) CommandType() string {
	return c.commandType
}

var (
	errEmptyID = commandError("ID should not be empty")
)

type commandError string

func (commandError) CommandError() bool {
	return true
}

func (ce commandError) Error() string {
	return string(ce)
}

func CreateCommand(aggregateID ID, blobType BlobType, data []byte) Command {
	return Command{
		ID:          aggregateID,
		commandType: "CREATE",
		validator: func(b Blob) error {
			if b.Deleted {
				return commandError("cannot create a deleted blob")
			}
			if b.Sequence != 0 || b.BlobType != "" || b.ID != "" {
				return commandError("cannot create an existing blob")
			}
			if aggregateID == "" {
				return errEmptyID
			}
			if blobType == "" {
				return commandError("BlobType should not be empty")
			}
			return nil
		},
		eventGenerator: func(b Blob) EventWithMetadataSlice {
			return wrap(aggregateID, 1, CreatedEvent{BlobType: blobType, Data: data})
		},
	}
}

func UpdateCommand(aggregateID ID, updatedData []byte, clearData bool) Command {
	return Command{
		ID:          aggregateID,
		commandType: "UPDATE",
		validator: func(b Blob) error {
			if err := validateID(b.ID, aggregateID); err != nil {
				return err
			}
			if len(updatedData) != 0 && clearData {
				return commandError("cannot updated as well as clear data at the same time")
			}
			return nil
		},
		eventGenerator: func(b Blob) EventWithMetadataSlice {
			var event Event

			if clearData {
				event = DataUpdatedEvent{Data: nil}
			} else {
				event = DataUpdatedEvent{Data: updatedData}
			}

			return wrap(aggregateID, b.Sequence+1, event)
		},
	}
}

func UpdateTagsCommand(aggregateID ID, tagsToAddOrUpdate Tags, tagsToDelete []string) Command {
	return Command{
		ID:          aggregateID,
		commandType: "UPDATE_TAGS",
		validator: func(b Blob) error {
			if err := validateID(b.ID, aggregateID); err != nil {
				return err
			}
			for _, tagToDelete := range tagsToDelete {
				if _, ok := tagsToAddOrUpdate[tagToDelete]; ok {
					msg := fmt.Sprintf("cannot delete a tag '%v' as it is being updated at the same time", tagToDelete)
					return commandError(msg)
				}
			}
			return nil
		},
		eventGenerator: func(b Blob) EventWithMetadataSlice {
			var events []Event

			if len(tagsToDelete) != 0 {
				var tagsDeleteEvent TagsDeletedEvent
				for _, tagToDelete := range tagsToDelete {
					if b.HasTag(tagToDelete) {
						tagsDeleteEvent = append(tagsDeleteEvent, tagToDelete)
					}
				}
				if len(tagsDeleteEvent) != 0 {
					events = append(events, tagsDeleteEvent)
				}
			}

			if len(tagsToAddOrUpdate) != 0 {
				tagsToAdd := make(Tags)
				tagsToUpdate := make(Tags)

				for key, value := range tagsToAddOrUpdate {
					if tagValue, ok := b.Tags[key]; ok && tagValue != value {
						tagsToUpdate[key] = value
					} else if !ok {
						tagsToAdd[key] = value
					}
				}

				if len(tagsToUpdate) != 0 {
					events = append(events, TagsUpdatedEvent(tagsToUpdate))
				}
				if len(tagsToAdd) != 0 {
					events = append(events, TagsAddedEvent(tagsToAdd))
				}
			}
			return wrap(aggregateID, b.Sequence+1, events...)
		},
	}
}

func DeleteCommand(aggregateID ID) Command {
	return Command{
		ID:          aggregateID,
		commandType: "DELETE",
		validator: func(b Blob) error {
			if err := validateID(b.ID, aggregateID); err != nil {
				return err
			}
			if b.Deleted {
				return commandError("cannot delete an already deleted blob")
			}
			return nil
		},
		eventGenerator: func(b Blob) EventWithMetadataSlice {
			return wrap(aggregateID, b.Sequence+1, DeletedEvent{})
		},
	}
}

func RestoreCommand(aggregateID ID) Command {
	return Command{
		ID:          aggregateID,
		commandType: "RESTORE",
		validator: func(b Blob) error {
			if err := validateID(b.ID, aggregateID); err != nil {
				return err
			}
			if !b.Deleted {
				return commandError(fmt.Sprintf("blob %v not deleted; only deleted blob can be restored", b.ID))
			}
			return nil
		},
		eventGenerator: func(b Blob) EventWithMetadataSlice {
			return wrap(aggregateID, b.Sequence+1, RestoredEvent{})
		},
	}
}

func validateID(blobID ID, aggregateID ID) error {
	if aggregateID == "" {
		return errEmptyID
	}
	if blobID == "" {
		return commandError(fmt.Sprintf("blob with id %s does not exist", aggregateID))
	}
	if blobID != aggregateID {
		return commandError(fmt.Sprintf("id %s in blob does not match %s in command", blobID, aggregateID))
	}
	return nil
}
