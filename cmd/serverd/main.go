package main

import (
	"log"
	"net/http"
	"os"

	"github.com/venkssa/eventsourcing/cmd/serverd/handlers"
	"github.com/venkssa/eventsourcing/internal/blob"

	"github.com/gorilla/mux"

	plog "github.com/venkssa/eventsourcing/internal/platform/log"
)

func main() {
	logger := &plog.StdLibLogger{Level: plog.Debug, Logger: log.New(os.Stderr, "", log.LstdFlags)}

	hdlrRegs := []handlers.HandlerRegisterer{
		handlers.NewBlobHandler(logger, blob.NewAggregateRepository(blob.NewInMemoryEventStore())),
	}

	muxRouter := mux.NewRouter()
	for _, hdlrReg := range hdlrRegs {
		hdlrReg.Register(muxRouter)
	}

	if err := http.ListenAndServe(":8080", muxRouter); err != nil {
		logger.Info(err)
		os.Exit(1)
	}
}
