package rpc

import (
	"encoding/base64"
	"net/http"
	"strings"
)

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

type ClientAuthorization interface {
	Authorize(*http.Request)
}

type ClientBasicAuthorization struct {
	User     string
	Password string
}

func (clientAuth ClientBasicAuthorization) Authorize(request *http.Request) {
	request.SetBasicAuth(clientAuth.User, clientAuth.Password)
}
