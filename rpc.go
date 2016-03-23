package rpc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

type Error interface {
	error
	GetErrorResponse(int) map[string]interface{}
}

type StatusError struct {
	Err         error
	MetadataMap map[string]interface{}
}

// Allows StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Allows StatusError to satisfy the error interface.
func (se StatusError) GetErrorResponse(status int) map[string]interface{} {
	md := se.MetadataMap
	if md == nil {
		md = map[string]interface{}{}
	}
	return map[string]interface{}{"type": "error", "error_code": status, "error": se.Error(), "metadata": md}
}

type Response interface {
	GetResponse() map[string]interface{}
}

type SyncResponse struct {
	Status     string
	StatusCode int
	Metadata   map[string]interface{}
}

func (s SyncResponse) GetResponse() map[string]interface{} {
	return map[string]interface{}{"type": "sync", "status": s.Status, "status_code": s.StatusCode, "metadata": s.Metadata}
}

type AsyncResponse struct {
	Status     string
	StatusCode int
	Metadata   map[string]interface{}
}

func (s AsyncResponse) GetResponse() map[string]interface{} {
	return map[string]interface{}{"type": "async", "status": s.Status, "status_code": s.StatusCode, "metadata": s.Metadata}
}

type Authorization interface {
	Authorize(*http.Request) bool
}

type BasicAuthorization struct {
	Username string
	Password string
}

func (auth BasicAuthorization) Authorize(r *http.Request) bool {
	header := r.Header.Get("Authorization")
	if header == "" {
		return false
	}

	split := strings.Fields(header)
	if len(split) != 2 || split[0] != "Basic" {
		return false
	}

	credentials, err := base64.StdEncoding.DecodeString(split[1])

	if err != nil {
		return false
	}

	user_pass := strings.Split(string(credentials), ":")

	if len(user_pass) != 2 {
		return false
	}

	if auth.Username != user_pass[0] || auth.Password != user_pass[1] {
		return false
	}

	return true
}

// A (simple) example of our application-wide configuration.
type Env struct {
	Auth           Authorization
	ClientUser     string
	ClientPassword string
}

// The Handler struct that takes a configured Env and a function matching
// our useful signature.
type Handler struct {
	*Env
	H func(e *Env, w http.ResponseWriter, r *http.Request) (Response, int, error)
}

// ServeHTTP allows our Handler type to satisfy http.Handler.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got request %s %s", r.Method, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)

	if h.Env.Auth != nil {
		authorized := h.Env.Auth.Authorize(r)
		if !authorized {
			status := http.StatusUnauthorized
			w.WriteHeader(status)
			encoder.Encode(StatusError{Err: errors.New("Unauthorized")}.GetErrorResponse(status))
			return
		}
	}

	response, status, err := h.H(h.Env, w, r)

	w.WriteHeader(status)

	if err != nil {
		switch e := err.(type) {
		case Error:
			error_response := e.GetErrorResponse(status)
			log.Printf("HTTP %d - %s, %v", status, e, error_response)
			encoder.Encode(error_response)
		default:
			// Any error types we don't specifically look out for default
			// to serving a HTTP 500
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
	} else {
		response_content := response.GetResponse()
		log.Println("HTTP %d - %s", status, response_content)
		encoder.Encode(response_content)
	}
}