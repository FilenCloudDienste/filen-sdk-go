package filen

import (
	"filen/filen-sdk-go/filen/client"
	"filen/filen-sdk-go/filen/crypto"
	"fmt"
)

type Filen struct {
	client     *client.Client
	MasterKeys []string
	PublicKey  string
	PrivateKey string
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

	masterKey, password := crypto.GeneratePasswordAndMasterKey(password, authInfo.Salt)

	// login and get keys
	keys, err := filen.client.Login(email, password)
	if err != nil {
		return err
	}
	filen.client.APIKey = keys.APIKey
	filen.MasterKeys, filen.PublicKey, filen.PrivateKey, err = crypto.UpdateKeys(filen.client, filen.client.APIKey, []string{masterKey})
	if err != nil {
		return err
	}
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

	sampleName := directoryContent.Folders[0].Name
	folderMetadata, err := crypto.DecryptFolderMetadata(filen.MasterKeys, sampleName)
	if err != nil {
		return nil, err
	}
	fmt.Println(folderMetadata.Name)

	return directoryContent, nil
}
