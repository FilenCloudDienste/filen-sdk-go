package api

import (
	"context"
	"fmt"
	"github.com/rclone/rclone/lib/rest"
	"net/http"
)

const (
	apiRoot = "https://gateway.filen.io"
)

type FilenAPI struct {
	restClient *rest.Client
}

func New() *FilenAPI {
	httpClient := &http.Client{}
	restClient := rest.NewClient(httpClient)
	restClient.SetRoot(apiRoot)

	return &FilenAPI{
		restClient,
	}
}

type apiResponse[T fmt.Stringer] struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Data    T      `json:"data"`
}

func (res *apiResponse[T]) String() string {
	return fmt.Sprintf("ApiResponse{status: %t, message: %s, code: %s, data: %s}", res.Status, res.Message, res.Code, res.Data)
}

type AuthInfo struct {
	AuthVersion int    `json:"authVersion"`
	Salt        string `json:"salt"`
}

func (authInfo *AuthInfo) String() string {
	return fmt.Sprintf("AuthInfo{auth version: %d, salt: %s}", authInfo.AuthVersion, authInfo.Salt)
}

func (api *FilenAPI) GetAuthInfo(ctx context.Context, email string) (authInfo *AuthInfo, err error) {
	request := struct {
		Email string `json:"email"`
	}{email}
	var response apiResponse[*AuthInfo]
	_, err = api.restClient.CallJSON(ctx, &rest.Opts{
		Method: "POST",
		Path:   "/v3/auth/info",
	}, request, &response)
	return response.Data, err
}

type LoginKeys struct {
	ApiKey     string `json:"apiKey"`
	MasterKeys string `json:"masterKeys"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func (keys *LoginKeys) String() string {
	return fmt.Sprintf("LoginKeys{\n\tapiKey: %s\n\tmasterKeys: %s\n\tpublicKey: %s\n\tprivateKey: %s\n}",
		keys.ApiKey, keys.MasterKeys, keys.PublicKey, keys.PrivateKey)
}

func (api *FilenAPI) Login(ctx context.Context, email, password string) (loginKeys *LoginKeys, err error) {
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
