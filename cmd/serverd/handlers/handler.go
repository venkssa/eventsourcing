package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	perrors "github.com/pkg/errors"
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

func withErrorHandler(logger log.Logger, fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		err := fn(rw, req)
		if err == nil {
			return
		}
		switch err := err.(type) {
		case handlerError:
			err.Write(logger, rw)
		case error:
			notFoundError(err).Write(logger, rw)
		}
	}
}

func Ok(rw http.ResponseWriter, contentType string, data io.Reader) error {
	rw.Header().Set("Content-Type", contentType)
	rw.WriteHeader(http.StatusOK)
	if _, err := io.Copy(rw, data); err != nil {
		return internalServerError(perrors.Wrap(err, "failed to write response body"))
	}
	return nil
}

func OkJSON(rw http.ResponseWriter, responseToEncodeAsJSON interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(responseToEncodeAsJSON); err != nil {
		return internalServerError(err)
	}
	return Ok(rw, "application/json", buf)
}

type handlerError struct {
	Status int
	error
}

func badRequestError(err error) handlerError {
	return handlerError{Status: http.StatusBadRequest, error: err}
}

func internalServerError(err error) handlerError {
	return handlerError{Status: http.StatusInternalServerError, error: err}
}

func notFoundError(err error) handlerError {
	return handlerError{Status: http.StatusNotFound, error: err}
}

func (e handlerError) Write(logger log.Logger, rw http.ResponseWriter) {
	if e.Status >= 500 {
		logger.Info(e.error)
	} else {
		logger.Debug(e.error)
	}
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
