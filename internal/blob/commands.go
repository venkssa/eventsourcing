package blob

import (
	"errors"
	"fmt"
)

type Command struct {
	ID
	commandType    string
	eventGenerator func(Blob) []EventWithMetadata
	validator      func(Blob) error
}

func (c Command) GenerateEvents(b Blob) ([]EventWithMetadata, CommandError) {
	if err := c.validator(b); err != nil {
		return nil, err
	}
	return c.eventGenerator(b), nil
}

func (c Command) CommandType() string {
	return c.commandType
}

type CommandError error

func NewCommandError(cmd Command, format string, a ...interface{}) CommandError {
	return commandError{Command: cmd, errStr: fmt.Sprintf(format, a...)}
}

type commandError struct {
	Command
	errStr string
}

func (ce commandError) Error() string {
	return fmt.Sprintf("cannot process %T command: %s", ce.CommandType(), ce.errStr)
}

func CreateCommand(aggregateID ID, blobType BlobType, data []byte) Command {
	return Command{
		ID:          aggregateID,
		commandType: "CREATE",
		validator: func(b Blob) error {
			if b.Deleted {
				return fmt.Errorf("cannot create a deleted blob")
			}
			if b.Sequence != 0 || b.BlobType != "" || b.ID != "" {
				return errors.New("cannot create an existing blob")
			}
			if aggregateID == "" {
				return errors.New("AggregateID should not be empty")
			}
			if blobType == "" {
				return errors.New("BlobType should not be empty")
			}
			return nil
		},
		eventGenerator: func(b Blob) []EventWithMetadata {
			return wrap(aggregateID, 1, CreatedEvent{BlobType: blobType, Data: data})
		},
	}
}

func UpdateCommand(aggregateID ID, updatedData []byte, clearData bool, tagsToAddOrUpdate Tags, tagsToDelete []string) Command {
	return Command{
		ID:          aggregateID,
		commandType: "UPDATE",
		validator: func(b Blob) error {
			if aggregateID == "" {
				return errors.New("ID should not be empty")
			}
			if b.ID != aggregateID {
				return fmt.Errorf("ID %s in blob does not match %s in command", b.ID, aggregateID)
			}
			if len(updatedData) != 0 && clearData {
				return errors.New("cannot updated as well as clear data at the same time")
			}
			for _, tagToDelete := range tagsToDelete {
				if _, ok := tagsToAddOrUpdate[tagToDelete]; ok {
					return fmt.Errorf("cannot delete a tag %v as it is being updated at the same time", tagsToDelete)
				}
			}
			return nil
		},
		eventGenerator: func(b Blob) []EventWithMetadata {
			var events []Event

			if len(updatedData) != 0 && !clearData {
				events = append(events, DataUpdatedEvent{Data: updatedData})
			}
			if clearData {
				events = append(events, DataUpdatedEvent{Data: nil})
			}
			if len(tagsToDelete) != 0 {
				events = append(events, TagsDeletedEvent(tagsToDelete))
			}
			if len(tagsToAddOrUpdate) != 0 {
				tagsToAdd := make(Tags)
				tagsToUpdate := make(Tags)

				for key, value := range tagsToAddOrUpdate {
					if _, ok := b.Tags[key]; ok {
						tagsToUpdate[key] = value
					} else {
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
			if aggregateID == "" {
				return errors.New("AggregateID should not be empty")
			}
			if b.ID != aggregateID {
				return fmt.Errorf("AggregateID %s in blob does not match %s in command", b.ID, aggregateID)
			}
			return nil
		},
		eventGenerator: func(b Blob) []EventWithMetadata {
			if b.Deleted {
				return nil
			}
			return wrap(aggregateID, b.Sequence+1, DeletedEvent{})

		},
	}
}

func RestoreCommand(aggregateID ID) Command {
	return Command{
		ID:          aggregateID,
		commandType: "RESTORE",
		validator: func(b Blob) error {
			if aggregateID == "" {
				return errors.New("AggregateID should not be empty")
			}
			if b.ID != aggregateID {
				return fmt.Errorf("AggregateID %s in blob does not match %s in command", b.ID, aggregateID)
			}
			if !b.Deleted {
				return fmt.Errorf("Blob %v not deleted. Only deleted blob can be restored", b.ID)
			}
			return nil
		},
		eventGenerator: func(b Blob) []EventWithMetadata {
			return wrap(aggregateID, b.Sequence+1, RestoredEvent{})
		},
	}
}
