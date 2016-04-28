package rpc

import (
	"log"
	"net/http"
)

type Response interface {
	Render(w http.ResponseWriter) error
}

type errorResponse struct {
	code int
	msg  string
}

func (err *errorResponse) Render(w http.ResponseWriter) error {
	w.WriteHeader(err.code)
	body := map[string]interface{}{"type": "error", "code": err.code, "error": err.msg}
	return WriteJson(w, body)
}

var NotFound = &errorResponse{http.StatusNotFound, "not found"}
var Unauthorized = &errorResponse{http.StatusUnauthorized, "unauthorized"}

func BadRequest(err error) Response {
	return &errorResponse{http.StatusBadRequest, err.Error()}
}

func InternalError(err error) Response {
	return &errorResponse{http.StatusInternalServerError, err.Error()}
}

type syncResponse struct {
	Metadata interface{}
}

var EmptySyncResponse = &syncResponse{}

func (s *syncResponse) Render(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	body := map[string]interface{}{"type": "sync", "code": http.StatusOK, "metadata": s.Metadata}
	return WriteJson(w, body)
}

func SyncResponse(metadata interface{}) Response {
	return &syncResponse{metadata}
}

type asyncResponse struct {
	Metadata  interface{}
	Operation string
}

func (s *asyncResponse) Render(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	body := map[string]interface{}{"type": "async", "code": http.StatusOK, "metadata": s.Metadata, "operation": s.Operation}
	return WriteJson(w, body)
}

func AsyncResponse(metadata interface{}, operation string) Response {
	return &asyncResponse{metadata, operation}
}

// A (simple) example of our application-wide configuration.
type Env struct {
	Auth       Authorization
	ClientAuth ClientAuthorization
	Config     map[string]string
	Debug      bool
}

// The Handler struct that takes a configured Env and a function matching
// our useful signature.
type Handler struct {
	*Env
	H func(e *Env, w http.ResponseWriter, r *http.Request) Response
}

// ServeHTTP allows our Handler type to satisfy http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Env.Debug {
		log.Printf("Got request %s %s", r.Method, r.URL.Path)
	}
	w.Header().Set("Content-Type", "application/json")

	if h.Env.Auth != nil {
		authorized := h.Env.Auth.Authorize(r)
		if !authorized {
			Unauthorized.Render(w)
			return
		}
	}

	response := h.H(h.Env, w, r)

	switch e := response.(type) {
	case Response:
		if h.Env.Debug {
			log.Printf("Response: %v", e)
		}
		response.Render(w)
	default:
		if h.Env.Debug {
			log.Printf("Internal error: %v", e)
		}

		// Any error types we don't specifically look out for default
		// to serving a HTTP 500
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}
