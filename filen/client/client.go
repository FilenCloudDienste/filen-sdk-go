package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiRoot = "https://gateway.filen.io"
)

type Client struct {
	APIKey string
}

type RequestError struct {
	Message         string
	Method          string
	Path            string
	UnderlyingError error
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("%s %s: %s (%s)", e.Method, e.Path, e.Message, e.UnderlyingError)
}

func (client *Client) request(method string, path string, request any, data any) (*APIResponse, error) {
	var marshalled []byte
	if request != nil {
		var err error
		marshalled, err = json.Marshal(request)
		if err != nil {
			return nil, &RequestError{fmt.Sprintf("Cannot unmarshal request body %#v", request), method, path, err}
		}
	}

	req, err := http.NewRequest(method, apiRoot+path, bytes.NewReader(marshalled))
	if err != nil {
		return nil, &RequestError{"Cannot build request", method, path, err}
	}

	req.Header.Set("Content-Type", "application/json")
	if client.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+client.APIKey)
	}

	httpClient := http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, &RequestError{"Cannot send request", method, path, err}
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, &RequestError{"Cannot read response body", method, path, err}
	}

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
		return nil, &RequestError{fmt.Sprintf("Cannot unmarshal response data for response %#v", response), method, path, err}
	}

	return &response, nil
}

type nakedApiResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type APIResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Code    string          `json:"code"`
	Data    json.RawMessage `json:"data"`
}

func (res *APIResponse) String() string {
	return fmt.Sprintf("ApiResponse{status: %t, message: %s, code: %s, data: %s}", res.Status, res.Message, res.Code, res.Data)
}
