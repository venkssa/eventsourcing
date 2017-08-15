package log

import "log"

type Logger interface {
	Info(v ...interface{})
	Debug(v ...interface{})
}

type Level byte

const (
	Info Level = iota
	Debug
)

type StdLibLogger struct {
	Level
	*log.Logger
}

func (l *StdLibLogger) Info(v ...interface{}) {
	l.Logger.Printf("INFO: %v", v...)
}

func (l *StdLibLogger) Debug(v ...interface{}) {
	if l.Level == Debug {
		l.Logger.Printf("DEBUG: %v", v...)
	}
}
