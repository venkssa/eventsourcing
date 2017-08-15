package blob

import (
	"errors"
	"fmt"
)

type Command struct {
	aggregateID    ID
	commandType    string
	eventGenerator func(Blob) []Event
	validator      func(Blob) error
}

func (c Command) GenerateEvents(b Blob) ([]Event, CommandError) {
	if err := c.validator(b); err != nil {
		return nil, err
	}
	return c.eventGenerator(b), nil
}

func (c Command) AggregateID() string {
	return c.aggregateID.AggregateID()
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
		aggregateID: aggregateID,
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
		eventGenerator: func(b Blob) []Event {
			return []Event{CreatedEvent{ID: aggregateID, BlobType: blobType, Data: data, Sequence: 1}}
		},
	}
}

func UpdateCommand(aggregateID ID, data []byte) Command {
	return Command{
		aggregateID: aggregateID,
		commandType: "UPDATE",
		validator: func(b Blob) error {

			return nil
		},
		eventGenerator: func(b Blob) []Event {
			return []Event{UpdatedEvent{ID: aggregateID, Data: data, Sequence: b.Sequence + 1}}
		},
	}
}

func DeleteCommand(aggregateID ID) Command {
	return Command{
		aggregateID: aggregateID,
		commandType: "DELETE",
		validator: func(b Blob) error {
			if b.ID != aggregateID {
				return fmt.Errorf("BID %s does not match the BID in DeleteCommand %s", b.ID, aggregateID)
			}
			return nil
		},
		eventGenerator: func(b Blob) []Event {
			return []Event{DeletedEvent{ID: aggregateID, Sequence: b.Sequence + 1}}
		},
	}
}
