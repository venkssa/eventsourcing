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
		muxRouter.Path("/blob/{id}/data").Methods(http.MethodGet).HandlerFunc(withErrorHandler(logger, hdlr.Data))

		sr := muxRouter.Path("/blob/{id}").Subrouter()
		sr.Methods(http.MethodGet).HandlerFunc(withErrorHandler(logger, hdlr.Find))
		sr.Methods(http.MethodPost).HandlerFunc(withErrorHandler(logger, hdlr.Create))
		sr.Methods(http.MethodPut).HandlerFunc(withErrorHandler(logger, hdlr.Update))
		sr.Methods(http.MethodDelete).HandlerFunc(withErrorHandler(logger, hdlr.Delete))
	})
	return hdlr
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
		UpdatedData     []byte    `json:"updatedData"`
		ClearData       bool      `json:"clearData"`
		AddOrUpdateTags blob.Tags `json:"addOrUpdateTags"`
		DeleteTags      []string  `json:"deleteTags"`
		RestoreBlob     bool      `json:"restoreBlob"`
	}
	if err := json.NewDecoder(req.Body).Decode(&updateReq); err != nil {
		return errorWithStatusCode{Status: http.StatusBadRequest, error: fmt.Errorf("failed to decode response body: %v", err)}
	}
	cmd := blob.UpdateCommand(
		blob.ID(vars["id"]),
		updateReq.UpdatedData,
		updateReq.ClearData,
		updateReq.AddOrUpdateTags,
		updateReq.DeleteTags)

	if updateReq.RestoreBlob {
		cmd = blob.RestoreCommand(cmd.ID)
	}
	return bh.process(cmd, rw)
}

func (bh *BlobHandler) Delete(rw http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)
	return bh.process(blob.DeleteCommand(blob.ID(vars["id"])), rw)
}

func (bh *BlobHandler) process(cmd blob.Command, rw http.ResponseWriter) error {
	if _, err := bh.aggregateRepo.Process(cmd); err != nil {
		return notFoundError(fmt.Errorf("cannot process %v with aggregate id %v: %v", cmd.CommandType(), cmd.ID, err))
	}
	rw.WriteHeader(http.StatusNoContent)
	return nil
}
