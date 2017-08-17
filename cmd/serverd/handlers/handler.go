package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/venkssa/eventsourcing/internal/platform/log"
)

type HandlerRegisterer interface {
	Register(mux *mux.Router)
}

type HandlerRegisterFunc func(mux *mux.Router)

func (h HandlerRegisterFunc) Register(mux *mux.Router) {
	h(mux)
}

func NotFoundHandler(logger log.Logger) http.HandlerFunc {
	return func(rw http.ResponseWriter, _ *http.Request) {
		notFoundError(errors.New("not found")).Write(logger, rw)
	}
}

func OkJSON(rw http.ResponseWriter, v interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return internalServerError(err)
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if _, err := io.Copy(rw, buf); err != nil {
		return internalServerError(fmt.Errorf("failed to write response body: %v", err))
	}
	return nil
}

func withErrorHandler(logger log.Logger, fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		err := fn(rw, req)
		if err == nil {
			return
		}
		switch err := err.(type) {
		case errorWithStatusCode:
			err.Write(logger, rw)
		case error:
			notFoundError(err).Write(logger, rw)
		}
	}
}

type errorWithStatusCode struct {
	Status int
	error
}

func badRequestError(err error) errorWithStatusCode {
	return errorWithStatusCode{Status: http.StatusBadRequest, error: err}
}

func internalServerError(err error) errorWithStatusCode {
	return errorWithStatusCode{Status: http.StatusInternalServerError, error: err}
}

func notFoundError(err error) errorWithStatusCode {
	return errorWithStatusCode{Status: http.StatusNotFound, error: err}
}

func (e errorWithStatusCode) Write(logger log.Logger, rw http.ResponseWriter) {
	logger.Info(e)
	ej := struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
	}{
		StatusCode: e.Status,
		Message:    fmt.Sprintf("Sorry we cannot find what you are looking for: %v", e.Error()),
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(e.Status)
	if encodeError := json.NewEncoder(rw).Encode(ej); encodeError != nil {
		logger.Info("Failed to write error reseponse", encodeError)
	}
}
