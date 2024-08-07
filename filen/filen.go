package filen

import (
	"encoding/json"
	"filen/filen-sdk-go/filen/client"
	"filen/filen-sdk-go/filen/crypto"
	"filen/filen-sdk-go/filen/util"
	"strings"
	"time"
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
		panic(err)
	}

	masterKey, password := crypto.GeneratePasswordAndMasterKey(password, authInfo.Salt)

	// login and get keys
	keys, err := filen.client.Login(email, password)
	if err != nil {
		return err
	}
	filen.client.APIKey = keys.APIKey
	filen.fetchMasterKeys(masterKey)
	//print(filen.MasterKeys[0])
	return nil
}

func (filen *Filen) fetchMasterKeys(masterKey string) {
	// = _updateKeys()
	encryptedMasterKey, err := crypto.EncryptMetadataSymmetrically(masterKey, masterKey)
	if err != nil {
		panic(err)
	}
	masterKeys, err := filen.client.GetUserMasterKeys(encryptedMasterKey)
	if err != nil {
		panic(err)
	}
	masterKeysStr, err := crypto.DecryptMetadataSymmetrically(masterKeys.Keys, masterKey)
	if err != nil {
		panic(err)
	}
	for _, key := range strings.Split(masterKeysStr, "|") {
		filen.MasterKeys = append(filen.MasterKeys, key)
	}
}

type File struct {
	UUID          string
	Name          string
	Size          int64
	MimeType      string
	EncryptionKey string
	Created       time.Time
	LastModified  time.Time
	ParentUUID    string
	Favorited     bool
}

type Directory struct {
	UUID       string
	Name       string
	ParentUUID string
	Color      string
	Created    time.Time
	Favorited  bool
}

func (filen *Filen) GetBaseFolderUUID() (string, error) {
	userBaseFolder, err := filen.client.GetUserBaseFolder()
	if err != nil {
		return "", err
	}
	return userBaseFolder.UUID, nil
}

func (filen *Filen) Readdir(uuid string) ([]*File, []*Directory, error) {
	// fetch directory content
	directoryContent, err := filen.client.GetDirectoryContent(uuid)
	if err != nil {
		return nil, nil, err
	}

	// transform files
	files := make([]*File, 0)
	for _, file := range directoryContent.Uploads {
		metadataStr, err := crypto.DecryptMetadataSymmetrically(file.Metadata, filen.MasterKeys[0]) //TODO master keys
		if err != nil {
			return nil, nil, err
		}
		var metadata struct {
			Name         string `json:"name"`
			Size         int    `json:"size"`
			Mime         string `json:"mime"`
			Key          string `json:"key"`
			LastModified int    `json:"lastModified"`
		}
		err = json.Unmarshal([]byte(metadataStr), &metadata)
		if err != nil {
			return nil, nil, err
		}

		files = append(files, &File{
			UUID:          file.UUID,
			Name:          metadata.Name,
			Size:          int64(metadata.Size),
			MimeType:      metadata.Mime,
			EncryptionKey: metadata.Key,
			Created:       util.TimestampToTime(file.Timestamp),
			LastModified:  util.TimestampToTime(metadata.LastModified),
			ParentUUID:    file.Parent,
			Favorited:     file.Favorited == 1,
		})
	}

	// transform directories
	directories := make([]*Directory, 0)
	for _, directory := range directoryContent.Folders {
		nameStr, err := crypto.DecryptMetadataSymmetrically(directory.Name, filen.MasterKeys[0]) //TODO master keys
		if err != nil {
			return nil, nil, err
		}
		var name struct {
			Name string `json:"name"`
		}
		err = json.Unmarshal([]byte(nameStr), &name)
		if err != nil {
			return nil, nil, err
		}

		directories = append(directories, &Directory{
			UUID:       directory.UUID,
			Name:       name.Name,
			ParentUUID: directory.Parent,
			Color:      "<none>", //TODO tmp
			Created:    util.TimestampToTime(directory.Timestamp),
			Favorited:  directory.Favorited == 1,
		})
	}

	return files, directories, nil
}
