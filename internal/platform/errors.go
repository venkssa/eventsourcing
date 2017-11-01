package platform

import "github.com/pkg/errors"

func IsMissingAggregate(err error) bool {
	type ismissingaggregate interface {
		IsMissingAggregate() bool
	}
	ime, ok := errors.Cause(err).(ismissingaggregate)
	return ok && ime.IsMissingAggregate()
}

func CommandError(err error) bool {
	type commandError interface {
		CommandError() bool
	}
	br, ok := errors.Cause(err).(commandError)
	return ok && br.CommandError()
}
