package rpc

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type CallbackRequest struct {
	Err        error
	Status     int                    `json:"status"`
	Output     string                 `json:"output"`
	Error      string                 `json:"error"`
	Identifier string                 `json:"identified"`
	Code       string                 `json:"code"`
	Op_type    string                 `json:"op_type"`
	Metadata   map[string]interface{} `json:"metadata"`
}

func (req *CallbackRequest) Prepare() {
	if req.Err == nil {
		req.Status = 200
		req.Error = ""
	} else {
		req.Status = 400
		req.Error = req.Err.Error()
	}
}

func DoCallbackRequest(url string, request CallbackRequest, clientAuth ClientAuthorization) (string, error) {
	request.Prepare()
	body, enc_err := json.Marshal(request)
	if enc_err != nil {
		return "", enc_err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	clientAuth.Authorize(req)

	client := &http.Client{}
	log.Printf("DoCallbackRequest to %s, %v", url, request)
	response, err := client.Do(req)
	defer response.Body.Close()
	response_body, _ := ioutil.ReadAll(response.Body)
	log.Printf("DoCallbackRequest request to %s, %v, headers: %v, response: %v, err: %v", url, request, response, string(response_body), err)
	return string(response_body), err
}

func DoErrorCallbackRequest(url string, ident string, err error, op_type string, code string, clientAuth ClientAuthorization) {
	req := CallbackRequest{Err: err, Identifier: ident, Code: code, Op_type: op_type}
	req.Prepare()
	DoCallbackRequest(url, req, clientAuth)
}
