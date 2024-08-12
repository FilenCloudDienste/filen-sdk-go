// Package client handles HTTP requests to the API and storage backends.
//
// API definitions are at https://gateway.filen.io/v3/docs.
package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"filen/filen-sdk-go/filen/crypto"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

var (
	gatewayURLs = []string{
		"https://gateway.filen.io",
		"https://gateway.filen.net",
		"https://gateway.filen-1.net",
		"https://gateway.filen-2.net",
		"https://gateway.filen-3.net",
		"https://gateway.filen-4.net",
		"https://gateway.filen-5.net",
		"https://gateway.filen-6.net",
	}
	egestURLs = []string{
		"https://egest.filen.io",
		"https://egest.filen.net",
		"https://egest.filen-1.net",
		"https://egest.filen-2.net",
		"https://egest.filen-3.net",
		"https://egest.filen-4.net",
		"https://egest.filen-5.net",
		"https://egest.filen-6.net",
	}
	ingestURLs = []string{
		"https://ingest.filen.io",
		"https://ingest.filen.net",
		"https://ingest.filen-1.net",
		"https://ingest.filen-2.net",
		"https://ingest.filen-3.net",
		"https://ingest.filen-4.net",
		"https://ingest.filen-5.net",
		"https://ingest.filen-6.net",
	}
)

// Client carries configuration.
type Client struct {
	APIKey string // the Filen API key
}

// A RequestError carries information on a failed HTTP request.
type RequestError struct {
	Message         string // description of where the error occurred
	Method          string // HTTP method of the request
	Path            string // URL path of the request
	UnderlyingError error  // the underlying error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s %s: %s (%s)", e.Method, e.Path, e.Message, e.UnderlyingError)
}

// api

// Request makes an HTTP request with an optional body and optionally returning a response body.
//
// The API sends responses in the format (written as TS type):
//
//	{status: number, message: string, code: string, data?: any}
//
// The APIResponse is returned, and the unmarshalled `data` is written to the data parameter, if applicable.
func (client *Client) Request(method string, path string, request any, data any) (*APIResponse, error) {
	// marshal request body
	var marshalled []byte
	if request != nil {
		var err error
		marshalled, err = json.Marshal(request)
		if err != nil {
			return nil, &RequestError{fmt.Sprintf("Cannot unmarshal request body %#v", request), method, path, err}
		}
	}

	// build request
	gatewayURL := gatewayURLs[rand.Intn(len(gatewayURLs))]
	req, err := http.NewRequest(method, gatewayURL+path, bytes.NewReader(marshalled))
	if err != nil {
		return nil, &RequestError{"Cannot build request", method, path, err}
	}

	// set headers (authorization)
	req.Header.Set("Content-Type", "application/json")
	if client.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+client.APIKey)
	}

	// send request
	httpClient := http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, &RequestError{"Cannot send request", method, path, err}
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, &RequestError{"Cannot read response body", method, path, err}
	}

	// try unmarshal response
	response := APIResponse{}
	err = json.Unmarshal(resBody, &response)
	if err != nil {
		var nakedResponse nakedApiResponse
		err = json.Unmarshal(resBody, &nakedResponse)
		if err != nil {
			return nil, &RequestError{"Cannot unmarshal naked response body", method, path, err}
		}
		return nil, &RequestError{"Response contains no data", method, path, err}
	}
	err = json.Unmarshal(response.Data, data)
	if err != nil {
		return nil, &RequestError{fmt.Sprintf("Cannot unmarshal response data for response %s", string(resBody)), method, path, err}
	}

	return &response, nil
}

type nakedApiResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// APIResponse represents a response from the API.
type APIResponse struct {
	Status  bool            `json:"status"`  // whether the request was successful
	Message string          `json:"message"` // additional information
	Code    string          `json:"code"`    // a status code
	Data    json.RawMessage `json:"data"`    // (optional) response body
}

func (res *APIResponse) String() string {
	return fmt.Sprintf("ApiResponse{status: %t, message: %s, code: %s, data: %s}", res.Status, res.Message, res.Code, res.Data)
}

// file chunks

// DownloadFileChunk downloads a file chunk from the storage backend.
func (client *Client) DownloadFileChunk(uuid string, region string, bucket string, chunkIdx int) ([]byte, error) {
	egestURL := egestURLs[rand.Intn(len(egestURLs))]
	url := fmt.Sprintf("%s/%s/%s/%s/%v", egestURL, region, bucket, uuid, chunkIdx)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+client.APIKey)
	httpClient := http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// UploadFileChunk uploads a file chunk to the storage backend.
func (client *Client) UploadFileChunk(uuid string, chunkIdx int, parentUUID string, uploadKey string, data []byte) error {
	// build request
	ingestURL := ingestURLs[rand.Intn(len(ingestURLs))]
	dataHash := hex.EncodeToString(crypto.RunSHA521(data))
	url := fmt.Sprintf("%s/v3/upload?uuid=%s&index=%v&parent=%s&uploadKey=%s&hash=%s",
		ingestURL, uuid, chunkIdx, parentUUID, uploadKey, dataHash)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+client.APIKey)

	// send request
	httpClient := http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	// check response
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	response := APIResponse{}
	err = json.Unmarshal(resBody, &response)
	if err != nil {
		return err
	}
	if response.Status == false {
		return errors.New("Cannot upload file chunk: " + response.Message)
	}

	return nil
}
