package client

import "filen/filen-sdk-go/filen/crypto"

// POST /v3/auth/info

type AuthInfo struct {
	AuthVersion int    `json:"authVersion"`
	Salt        string `json:"salt"`
}

// GetAuthInfo calls /v3/auth/info.
func (client *Client) GetAuthInfo(email string) (*AuthInfo, error) {
	request := struct {
		Email string `json:"email"`
	}{email}
	authInfo := &AuthInfo{}
	_, err := client.Request("POST", "/v3/auth/info", request, authInfo)
	return authInfo, err
}

// POST /v3/login

type LoginKeys struct {
	APIKey     string                 `json:"apiKey"`
	MasterKeys crypto.EncryptedString `json:"masterKeys"`
	PublicKey  string                 `json:"publicKey"`
	PrivateKey string                 `json:"privateKey"`
}

// Login calls /v3/login.
func (client *Client) Login(email, password string) (*LoginKeys, error) {
	request := struct {
		Email         string `json:"email"`
		Password      string `json:"password"`
		TwoFactorCode string `json:"twoFactorCode"`
		AuthVersion   int    `json:"authVersion"`
	}{email, password, "XXXXXX", 2}
	loginKeys := &LoginKeys{}
	_, err := client.Request("POST", "/v3/login", request, loginKeys)
	return loginKeys, err
}

// GET /v3/user/baseFolder

type UserBaseFolder struct {
	UUID string `json:"uuid"`
}

// GetUserBaseFolder calls /v3/user/baseFolder.
func (client *Client) GetUserBaseFolder() (*UserBaseFolder, error) {
	userBaseFolder := &UserBaseFolder{}
	_, err := client.Request("GET", "/v3/user/baseFolder", nil, userBaseFolder)
	return userBaseFolder, err
}

// POST /v3/dir/content

type DirectoryContent struct {
	Uploads []struct {
		UUID      string                 `json:"uuid"`
		Metadata  crypto.EncryptedString `json:"metadata"`
		Rm        string                 `json:"rm"`
		Timestamp int                    `json:"timestamp"`
		Chunks    int                    `json:"chunks"`
		Size      int                    `json:"size"`
		Bucket    string                 `json:"bucket"`
		Region    string                 `json:"region"`
		Parent    string                 `json:"parent"`
		Version   int                    `json:"version"`
		Favorited int                    `json:"favorited"`
	} `json:"uploads"`
	Folders []struct {
		UUID      string                 `json:"uuid"`
		Name      crypto.EncryptedString `json:"name"`
		Parent    string                 `json:"parent"`
		Color     interface{}            `json:"color"`
		Timestamp int                    `json:"timestamp"`
		Favorited int                    `json:"favorited"`
		IsSync    int                    `json:"is_sync"`
		IsDefault int                    `json:"is_default"`
	} `json:"folders"`
}

// GetDirectoryContent calls /v3/dir/content.
func (client *Client) GetDirectoryContent(uuid string) (*DirectoryContent, error) {
	request := struct {
		UUID string `json:"uuid"`
	}{uuid}
	directoryContent := &DirectoryContent{}
	_, err := client.Request("POST", "/v3/dir/content", request, directoryContent)
	return directoryContent, err
}

// POST /v3/user/masterKeys

type UserMasterKeys struct {
	Keys crypto.EncryptedString `json:"keys"`
}

// GetUserMasterKeys calls /v3/user/masterKeys.
func (client *Client) GetUserMasterKeys(encryptedMasterKey crypto.EncryptedString) (*UserMasterKeys, error) {
	request := struct {
		MasterKey crypto.EncryptedString `json:"masterKeys"`
	}{encryptedMasterKey}
	userMasterKeys := &UserMasterKeys{}
	_, err := client.Request("POST", "/v3/user/masterKeys", request, userMasterKeys)
	return userMasterKeys, err
}

// POST /v3/upload/done

type UploadDonePayload struct {
	UUID       string                 `json:"uuid"`
	Name       crypto.EncryptedString `json:"name"`
	NameHashed string                 `json:"nameHashed"`
	Size       crypto.EncryptedString `json:"size"`
	Chunks     int                    `json:"chunks"`
	Mime       crypto.EncryptedString `json:"mime"`
	Rm         string                 `json:"rm"`
	Metadata   crypto.EncryptedString `json:"metadata"`
	Version    int                    `json:"version"`
	UploadKey  string                 `json:"uploadKey"`
}

type UploadDoneResponse struct {
	Chunks int `json:"chunks"`
	Size   int `json:"size"`
}

// UploadDone calls /v3/upload/done.
func (client *Client) UploadDone(payload UploadDonePayload) (*UploadDoneResponse, error) {
	response := &UploadDoneResponse{}
	_, err := client.Request("POST", "/v3/upload/done", payload, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
