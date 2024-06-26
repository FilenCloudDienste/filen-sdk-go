package filen

import (
	"crypto/sha512"
	"encoding/hex"
	"filen/filen-sdk-go/filen/client"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
)

type Filen struct {
	client *client.Client
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
		panic(err)
	}

	// compute password as sha512 hash of second half of sha512-PBKDF2 of raw password
	password = hex.EncodeToString(pbkdf2.Key([]byte(password), []byte(authInfo.Salt), 200000, 512/8, sha512.New))
	password = password[len(password)/2:]
	derivedPasswordHash := sha512.New()
	derivedPasswordHash.Write([]byte(password))
	password = fmt.Sprintf("%032x", derivedPasswordHash.Sum(nil))

	// login and get keys
	keys, err := filen.client.Login(email, password)
	if err != nil {
		return err
	}
	filen.client.APIKey = keys.ApiKey
	return nil
}

func (filen *Filen) Readdir() (*client.DirectoryContent, error) {
	// fetch base folder uuid
	userBaseFolder, err := filen.client.GetUserBaseFolder()
	if err != nil {
		return nil, err
	}

	// fetch directory content
	directoryContent, err := filen.client.GetDirectoryContent(userBaseFolder.UUID)
	if err != nil {
		return nil, err
	}

	return directoryContent, nil
}
