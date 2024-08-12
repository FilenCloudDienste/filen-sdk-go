// Package filen provides an SDK interface to interact with the cloud drive.
package filen

import (
	"github.com/JupiterPi/filen-sdk-go/filen/client"
	"github.com/JupiterPi/filen-sdk-go/filen/crypto"
	"strings"
)

// Filen provides the SDK interface. Needs to be initialized via [New].
type Filen struct {
	client *client.Client

	// MasterKeys contains the crypto master keys for the current user. When the user changes
	// their password, a new master key is appended. For decryption, all master keys are tried
	// until one works; for decryption, always use the latest master key.
	MasterKeys [][]byte
}

// New creates a new Filen and initializes it with the given email and password
// by logging in with the API and preparing the API key and master keys.
func New(email, password string) (*Filen, error) {
	filen := &Filen{
		client: &client.Client{},
	}

	// fetch salt
	authInfo, err := filen.client.GetAuthInfo(email)
	if err != nil {
		return nil, err
	}

	masterKey, password := crypto.GeneratePasswordAndMasterKey(password, authInfo.Salt)

	// login and get keys
	keys, err := filen.client.Login(email, password)
	if err != nil {
		return nil, err
	}
	filen.client.APIKey = keys.APIKey

	// fetch, encrypt and apply master keys
	encryptedMasterKey, err := crypto.EncryptMetadata(string(masterKey), masterKey)
	if err != nil {
		return nil, err
	}
	masterKeys, err := filen.client.GetUserMasterKeys(encryptedMasterKey)
	if err != nil {
		return nil, err
	}
	masterKeysStr, err := crypto.DecryptMetadata(masterKeys.Keys, masterKey)
	if err != nil {
		return nil, err
	}
	for _, key := range strings.Split(masterKeysStr, "|") {
		filen.MasterKeys = append(filen.MasterKeys, []byte(key))
	}

	return filen, nil
}

// CurrentMasterKey returns the current master key to use for encryption.
// Multiple possible master keys exist for decryption, but only the latest one should be used for encryption.
func (filen *Filen) CurrentMasterKey() []byte {
	return filen.MasterKeys[len(filen.MasterKeys)-1]
}
