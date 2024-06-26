package client

// POST /v3/auth/info

type AuthInfo struct {
	AuthVersion int    `json:"authVersion"`
	Salt        string `json:"salt"`
}

func (client *Client) GetAuthInfo(email string) (*AuthInfo, error) {
	request := struct {
		Email string `json:"email"`
	}{email}
	authInfo := &AuthInfo{}
	_, err := client.request("POST", "/v3/auth/info", request, authInfo)
	return authInfo, err
}

// POST /v3/login

type LoginKeys struct {
	ApiKey     string `json:"apiKey"`
	MasterKeys string `json:"masterKeys"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func (client *Client) Login(email, password string) (*LoginKeys, error) {
	request := struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		TwoFactorCode string `json:"twoFactorCode"`
		AuthVersion   int    `json:"authVersion"`
	}{email, password, "XXXXXX", 2}
	loginKeys := &LoginKeys{}
	_, err := client.request("POST", "/v3/login", request, loginKeys)
	return loginKeys, err
}

// GET /v3/user/baseFolder

type UserBaseFolder struct {
	UUID string `json:"uuid"`
}

func (client *Client) GetUserBaseFolder() (*UserBaseFolder, error) {
	userBaseFolder := &UserBaseFolder{}
	_, err := client.request("GET", "/v3/user/baseFolder", nil, userBaseFolder)
	return userBaseFolder, err
}
