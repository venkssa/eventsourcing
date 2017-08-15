package handlers

import (
	"bytes"
	"encoding/json"
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
		ErrorJSON(logger, rw, nil, http.StatusNotFound)
	}
}

func WriteJSON(logger log.Logger, rw http.ResponseWriter, v interface{}, err error) {
	if err != nil {
		ErrorJSON(logger, rw, err, http.StatusNotFound)
		return
	}
	OkJSON(logger, rw, v)
}

func ErrorJSON(logger log.Logger, rw http.ResponseWriter, err error, status int) {
	if err != nil {
		logger.Info(err)
	}
	ej := struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
	}{
		StatusCode: status,
		Message:    fmt.Sprintf("Sorry we cannot find what you are looking for: %v", err),
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	if encodeError := json.NewEncoder(rw).Encode(ej); encodeError != nil {
		logger.Info("Failed to write error reseponse", encodeError)
	}
}

func OkJSON(logger log.Logger, rw http.ResponseWriter, v interface{}) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		ErrorJSON(logger, rw, err, http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	if _, err := io.Copy(rw, buf); err != nil {
		logger.Info("Failed to write response: ", err)
	}
}
