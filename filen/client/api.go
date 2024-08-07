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
	APIKey     string `json:"apiKey"`
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

// POST /v3/dir/content

type DirectoryContent struct {
	Uploads []struct {
		UUID      string `json:"uuid"`
		Metadata  string `json:"metadata"`
		Rm        string `json:"rm"`
		Timestamp int    `json:"timestamp"`
		Chunks    int    `json:"chunks"`
		Size      int    `json:"size"`
		Bucket    string `json:"bucket"`
		Region    string `json:"region"`
		Parent    string `json:"parent"`
		Version   int    `json:"version"`
		Favorited int    `json:"favorited"`
	} `json:"uploads"`
	Folders []struct {
		UUID      string      `json:"uuid"`
		Name      string      `json:"name"`
		Parent    string      `json:"parent"`
		Color     interface{} `json:"color"`
		Timestamp int         `json:"timestamp"`
		Favorited int         `json:"favorited"`
		IsSync    int         `json:"is_sync"`
		IsDefault int         `json:"is_default"`
	} `json:"folders"`
}

func (client *Client) GetDirectoryContent(uuid string) (*DirectoryContent, error) {
	request := struct {
		UUID string `json:"uuid"`
	}{uuid}
	directoryContent := &DirectoryContent{}
	_, err := client.request("POST", "/v3/dir/content", request, directoryContent)
	return directoryContent, err
}

// POST /v3/user/masterKeys

type UserMasterKeys struct {
	Keys string `json:"keys"`
}

func (client *Client) GetUserMasterKeys(encryptedMasterKeys string) (*UserMasterKeys, error) {
	request := struct {
		MasterKeys string `json:"masterKeys"`
	}{encryptedMasterKeys}
	userMasterKeys := &UserMasterKeys{}
	_, err := client.request("POST", "/v3/user/masterKeys", request, userMasterKeys)
	return userMasterKeys, err
}

// GET /v3/user/keyPair/info

type UserKeyPairInfo struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func (client *Client) GetUserKeyPairInfo() (*UserKeyPairInfo, error) {
	userKeyPairInfo := &UserKeyPairInfo{}
	_, err := client.request("GET", "/v3/user/keyPairInfo", nil, userKeyPairInfo)
	return userKeyPairInfo, err
}

// POST /v3/user/keyPair/update

func (client *Client) UpdateUserKeyPair(publicKey string, encryptedPrivateKey string) error {
	request := struct {
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
	}{publicKey, encryptedPrivateKey}
	_, err := client.request("POST", "/v3/user/keyPair/update", request, nil)
	return err
}
