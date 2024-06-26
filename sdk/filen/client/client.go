package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	apiRoot = "https://gateway.filen.io"
)

type Client struct {
	APIKey string
}

func (client *Client) request(method string, path string, body any, data any) (*apiResponse, error) {
	marshalled, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, apiRoot+path, bytes.NewReader(marshalled))
	if err != nil {
		log.Fatalf("Cannot build request: %s", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if client.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+client.APIKey)
	}

	httpClient := http.Client{Timeout: 10 * time.Second}
	res, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("Cannot send request: %s", err)
		return nil, err
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Cannot read response body: %s", err)
		return nil, err
	}

	response := apiResponse{}
	err = json.Unmarshal(resBody, &response)
	if err != nil {
		var nakedResponse nakedApiResponse
		err = json.Unmarshal(resBody, &nakedResponse)
		if err != nil {
			log.Fatalf("Cannot unmarshal naked response body: %s, %s", resBody, err)
		}
		log.Fatalf("Response contains no data: %v", nakedResponse)
	}

	err = json.Unmarshal(response.Data, data)
	if err != nil {
		log.Fatalf("Cannot unmarshal response data: %s, %s", response.Data, err)
		return nil, err
	}

	return &response, nil
}

type nakedApiResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type apiResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Code    string          `json:"code"`
	Data    json.RawMessage `json:"data"`
}

func (res *apiResponse) String() string {
	return fmt.Sprintf("ApiResponse{status: %t, message: %s, code: %s, data: %s}", res.Status, res.Message, res.Code, res.Data)
}

type AuthInfo struct {
	AuthVersion int    `json:"authVersion"`
	Salt        string `json:"salt"`
}

func (authInfo AuthInfo) String() string {
	return fmt.Sprintf("AuthInfo{auth version: %d, salt: %s}", authInfo.AuthVersion, authInfo.Salt)
}

func (client *Client) GetAuthInfo(email string) (*AuthInfo, error) {
	request := struct {
		Email string `json:"email"`
	}{email}
	authInfo := AuthInfo{}
	_, err := client.request("POST", "/v3/auth/info", request, &authInfo)
	return &authInfo, err
}

/*type LoginKeys struct {
	ApiKey     string `json:"apiKey"`
	MasterKeys string `json:"masterKeys"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func (keys *LoginKeys) String() string {
	return fmt.Sprintf("LoginKeys{\n\tapiKey: %s\n\tmasterKeys: %s\n\tpublicKey: %s\n\tprivateKey: %s\n}",
		keys.ApiKey, keys.MasterKeys, keys.PublicKey, keys.PrivateKey)
}

func (api *filen.Filen) Login(ctx context.Context, email, password string) (loginKeys *LoginKeys, err error) {
	request := struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		TwoFactorCode string `json:"twoFactorCode"`
		AuthVersion   int    `json:"authVersion"`
	}{email, password, "XXXXXX", 2}
	var response apiResponse[*LoginKeys]
	_, err = api.restClient.CallJSON(ctx, &rest.Opts{
		Method: "POST",
		Path:   "/v3/login",
	}, request, &response)
	return response.Data, err
}

type UserBaseFolderResponse struct {
	UUID string `json:"uuid"`
}

func (res *UserBaseFolderResponse) String() string {
	return fmt.Sprintf("UserBaseFolderResponse{uuid: %s}", res.UUID)
}

func (api *filen.Filen) GetUserBaseFolder(ctx context.Context) (baseFolderUUID string, err error) {
	var response apiResponse[*UserBaseFolderResponse]
	_, err = api.restClient.CallJSON(ctx, &rest.Opts{
		Method: "GET",
		Path:   "/v3/user/baseFolder",
		ExtraHeaders: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", ctx.Value("apiKey")),
		},
	}, nil, &response)
	fmt.Println(&response)
	return response.Data.UUID, err
}
*/
