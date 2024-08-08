package filen

import (
	"encoding/json"
	"filen/filen-sdk-go/filen/crypto"
	"filen/filen-sdk-go/filen/util"
	"time"
)

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
	Region        string
	Bucket        string
	Chunks        int
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

func (filen *Filen) ReadDirectory(uuid string) ([]*File, []*Directory, error) {
	// fetch directory content
	directoryContent, err := filen.client.GetDirectoryContent(uuid)
	if err != nil {
		return nil, nil, err
	}

	// transform files
	files := make([]*File, 0)
	for _, file := range directoryContent.Uploads {
		metadataStr, err := crypto.DecryptMetadataAllKeys(file.Metadata, filen.MasterKeys)
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
			Region:        file.Region,
			Bucket:        file.Bucket,
			Chunks:        file.Chunks,
		})
	}

	// transform directories
	directories := make([]*Directory, 0)
	for _, directory := range directoryContent.Folders {
		nameStr, err := crypto.DecryptMetadataAllKeys(directory.Name, filen.MasterKeys)
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

func (filen *Filen) ReadFile(file *File) ([]byte, error) {
	data := make([]byte, 0)
	for chunk := 0; chunk < file.Chunks; chunk++ {
		encryptedChunkData, err := filen.client.DownloadFileChunk(file.UUID, file.Region, file.Bucket, chunk)
		if err != nil {
			return nil, err
		}
		chunkData, err := crypto.DecryptData(encryptedChunkData, file.EncryptionKey)
		if err != nil {
			return nil, err
		}
		data = append(data, chunkData...)
	}
	return data, nil
}
