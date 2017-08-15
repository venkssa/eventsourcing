package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/venkssa/eventsourcing/internal/blob"
	"github.com/venkssa/eventsourcing/internal/platform/log"
)

type BlobHandler struct {
	HandlerRegisterFunc
	logger        log.Logger
	aggregateRepo blob.AggregateRepository
}

func NewBlobHandler(logger log.Logger, aggregateRepo blob.AggregateRepository) HandlerRegisterer {
	hdlr := &BlobHandler{logger: logger, aggregateRepo: aggregateRepo}
	hdlr.HandlerRegisterFunc = HandlerRegisterFunc(func(muxRouter *mux.Router) {
		sr := muxRouter.Path("/blob/{id}").Subrouter()
		sr.Methods(http.MethodGet).HandlerFunc(hdlr.Find)
		sr.Methods(http.MethodPost).HandlerFunc(hdlr.Create)
		sr.Methods(http.MethodDelete).HandlerFunc(hdlr.Delete)
	})
	return hdlr
}

func (bh *BlobHandler) Find(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	blb, err := bh.aggregateRepo.Find(vars["id"])

	b := struct {
		blob.ID       `json:"id"`
		blob.BlobType `json:"blobType"`
		Data          []byte `json:"data"`
		Deleted       bool   `json:"deleted"`
		Sequence      uint64 `json:"sequence"`
	}(blb)

	WriteJSON(bh.logger, rw, b, err)
}

func (bh *BlobHandler) Create(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var createReq struct {
		Data     []byte        `json:"data"`
		BlobType blob.BlobType `json:"blobType"`
	}
	err := json.NewDecoder(req.Body).Decode(&createReq)
	if err != nil {
		ErrorJSON(bh.logger, rw, errors.New("failed to decode response body"), http.StatusBadRequest)
		return
	}

	cmd := blob.CreateCommand(blob.ID(vars["id"]), createReq.BlobType, createReq.Data)
	bh.process(cmd, rw)
}

func (bh *BlobHandler) Update(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	var dataJSON struct {
		Data []byte `json:"data"`
	}
	err := json.NewDecoder(req.Body).Decode(&dataJSON)
	if err != nil {
		ErrorJSON(bh.logger, rw, errors.New("failed to decode response body"), http.StatusBadRequest)
		return
	}
	bh.process(blob.UpdateCommand(blob.ID(vars["id"]), dataJSON.Data), rw)
}

func (bh *BlobHandler) Delete(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	bh.process(blob.DeleteCommand(blob.ID(vars["id"])), rw)
}

func (bh *BlobHandler) process(cmd blob.Command, rw http.ResponseWriter) {
	_, err := bh.aggregateRepo.Process(cmd)
	if err != nil {
		ErrorJSON(bh.logger, rw, fmt.Errorf("cannot process %v with aggregate id %v: %v", cmd.CommandType(), cmd.AggregateID(), err), http.StatusNotFound)
		return
	}
	rw.WriteHeader(http.StatusNoContent)
}
