package rpc

import (
	"encoding/json"
	"net/http"
)

func WriteJson(w http.ResponseWriter, body interface{}) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(body)
}
