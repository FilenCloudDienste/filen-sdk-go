package filen

import (
	"filen/filen-sdk-go/filen/client"
	"filen/filen-sdk-go/filen/crypto"
	"strings"
)

type Filen struct {
	client     *client.Client
	MasterKeys []string
}

func New() *Filen {
	return &Filen{
		client: &client.Client{},
	}
}

func (filen *Filen) Login(email, password string) error {
	// fetch salt
	authInfo, err := filen.client.GetAuthInfo(email)
	if err != nil {
		return err
	}

	masterKey, password := crypto.GeneratePasswordAndMasterKey(password, authInfo.Salt)

	// login and get keys
	keys, err := filen.client.Login(email, password)
	if err != nil {
		return err
	}
	filen.client.APIKey = keys.APIKey

	// fetch master keys
	encryptedMasterKey, err := crypto.EncryptMetadata(masterKey, masterKey)
	if err != nil {
		return err
	}
	masterKeys, err := filen.client.GetUserMasterKeys(encryptedMasterKey)
	if err != nil {
		return err
	}
	masterKeysStr, err := crypto.DecryptMetadata(masterKeys.Keys, masterKey)
	if err != nil {
		return err
	}
	for _, key := range strings.Split(masterKeysStr, "|") {
		filen.MasterKeys = append(filen.MasterKeys, key)
	}

	return nil
}

// masterKey returns the master key to use for encryption
func (filen *Filen) masterKey() string {
	return filen.MasterKeys[len(filen.MasterKeys)-1]
}
