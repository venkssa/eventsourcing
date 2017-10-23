package main

import (
	_ "expvar"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"

	"github.com/venkssa/eventsourcing/cmd/serverd/handlers"
	"github.com/venkssa/eventsourcing/internal/blob"

	"github.com/gorilla/mux"

	plog "github.com/venkssa/eventsourcing/internal/platform/log"
)

var eventStoreFilePath = flag.String("eventStoreFilePath", "/tmp/eventstore", "path for event store using file system.")

func main() {
	flag.Parse()
	logger := &plog.StdLibLogger{Level: plog.Debug, Logger: log.New(os.Stderr, "", log.LstdFlags)}

	hdlrRegs := []handlers.HandlerRegisterer{
		handlers.NewBlobHandler(logger, blob.NewAggregateRepository(blob.NewLocalFileSystemEventStore(*eventStoreFilePath))),
	}

	muxRouter := mux.NewRouter()
	for _, hdlrReg := range hdlrRegs {
		hdlrReg.Register(muxRouter)
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := http.ListenAndServe(":8080", muxRouter); err != nil {
			logger.Info(err)
			os.Exit(1)
		}
	}()

	go func() {
		defer wg.Done()
		if err := http.ListenAndServe(":8081", nil); err != nil {
			logger.Info(err)
			os.Exit(1)
		}
	}()

	wg.Wait()
}
