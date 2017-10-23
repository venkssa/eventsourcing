package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/venkssa/eventsourcing/internal/blob"
	"github.com/venkssa/eventsourcing/internal/platform/log"
)

type BlobHandler struct {
	HandlerRegisterFunc
	aggregateRepo blob.AggregateRepository
}

func NewBlobHandler(logger log.Logger, aggregateRepo blob.AggregateRepository) HandlerRegisterer {
	hdlr := &BlobHandler{aggregateRepo: aggregateRepo}
	hdlr.HandlerRegisterFunc = HandlerRegisterFunc(func(muxRouter *mux.Router) {
		s := muxRouter.PathPrefix("/blob").Subrouter()

		s.HandleFunc("/{id}", withErrorHandler(logger, hdlr.Find)).Methods(http.MethodGet)
		s.HandleFunc("/{id}", withErrorHandler(logger, hdlr.Create)).Methods(http.MethodPost)
		s.HandleFunc("/{id}", withErrorHandler(logger, hdlr.Update)).Methods(http.MethodPut)
		s.HandleFunc("/{id}", withErrorHandler(logger, hdlr.Delete)).Methods(http.MethodDelete)
		s.HandleFunc("/{id}/data", withErrorHandler(logger, hdlr.Data)).Methods(http.MethodGet)
		s.HandleFunc("/{id}/tags", withErrorHandler(logger, hdlr.UpdateTags)).Methods(http.MethodPut)
	})
	return hdlr
}

func (bh *BlobHandler) Find(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	blb, err := bh.aggregateRepo.Find(blob.ID(vars["id"]))
	if err != nil {
		return notFoundError(err)
	}

	b := struct {
		blob.ID       `json:"id"`
		blob.BlobType `json:"blobType"`
		Data          []byte `json:"data"`
		Deleted       bool   `json:"deleted"`
		Sequence      uint64 `json:"sequence"`
		blob.Tags     `json:"tags"`
	}(blb)

	return OkJSON(rw, b)
}

func (bh *BlobHandler) Create(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	blobType := req.Header.Get("Content-Type")
	if blobType == "" {
		return notFoundError(errors.New("Content-Type not set"))
	}
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return notFoundError(err)
	}

	cmd := blob.CreateCommand(blob.ID(vars["id"]), blob.BlobType(blobType), data)
	return bh.process(cmd, rw)
}

func (bh *BlobHandler) Update(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	var updateReq struct {
		UpdatedData []byte `json:"updatedData"`
		ClearData   bool   `json:"clearData"`
		RestoreBlob bool   `json:"restoreBlob"`
	}
	if err := json.NewDecoder(req.Body).Decode(&updateReq); err != nil {
		return errorWithStatusCode{Status: http.StatusBadRequest, error: fmt.Errorf("failed to decode response body: %v", err)}
	}

	var cmd blob.Command
	if updateReq.RestoreBlob {
		cmd = blob.RestoreCommand(blob.ID(vars["id"]))
	} else {
		cmd = blob.UpdateCommand(blob.ID(vars["id"]), updateReq.UpdatedData, updateReq.ClearData)
	}
	return bh.process(cmd, rw)
}

func (bh *BlobHandler) Delete(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	return bh.process(blob.DeleteCommand(blob.ID(vars["id"])), rw)
}

func (bh *BlobHandler) Data(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	blb, err := bh.aggregateRepo.Find(blob.ID(vars["id"]))
	if err != nil {
		return notFoundError(err)
	}

	if blb.Deleted {
		return notFoundError(fmt.Errorf("blob %v is deleted", blb.ID))
	}
	return Ok(rw, blb.BlobType.String(), bytes.NewBuffer(blb.Data))
}

func (bh *BlobHandler) UpdateTags(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	var tagReq struct {
		AddOrUpdate blob.Tags `json:"addOrUpdate"`
		Delete      []string  `json:"delete"`
	}
	if err := json.NewDecoder(req.Body).Decode(&tagReq); err != nil {
		return errorWithStatusCode{Status: http.StatusBadRequest, error: fmt.Errorf("failed to decode response body: %v", err)}
	}

	cmd := blob.UpdateTagsCommand(blob.ID(vars["id"]), tagReq.AddOrUpdate, tagReq.Delete)
	return bh.process(cmd, rw)
}

func (bh *BlobHandler) process(cmd blob.Command, rw http.ResponseWriter) error {
	if _, err := bh.aggregateRepo.Process(cmd); err != nil {
		return notFoundError(fmt.Errorf("cannot process %v with aggregate id %v: %v", cmd.CommandType(), cmd.ID, err))
	}
	rw.WriteHeader(http.StatusNoContent)
	return nil
}
