package rpc

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteJson(w http.ResponseWriter, body interface{}) error {
	log.Printf("Encoding response: %v", body)
	encoder := json.NewEncoder(w)
	return encoder.Encode(body)
}
