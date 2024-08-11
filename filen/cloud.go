package filen

import (
	"encoding/hex"
	"encoding/json"
	"filen/filen-sdk-go/filen/client"
	"filen/filen-sdk-go/filen/crypto"
	"filen/filen-sdk-go/filen/util"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
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

const (
	maxConcurrentDownloads = 16
	maxConcurrentWriters   = 16
	chunkSize              = 1048576
)

func (filen *Filen) DownloadFile(file *File, destination *os.File) error {
	downloadSem := make(chan int, maxConcurrentDownloads)
	writeSem := make(chan int, maxConcurrentWriters)
	c := make(chan int)
	errs := make(chan error)

	for chunk := 0; chunk < file.Chunks; chunk++ {
		go func() {
			downloadSem <- 1
			defer func() { <-downloadSem }()

			encryptedChunkData, err := filen.client.DownloadFileChunk(file.UUID, file.Region, file.Bucket, chunk)
			if err != nil {
				errs <- err
				return
			}
			chunkData, err := crypto.DecryptData(encryptedChunkData, file.EncryptionKey)
			if err != nil {
				errs <- err
				return
			}

			go func() {
				writeSem <- 1
				defer func() { <-writeSem }()

				_, err = destination.WriteAt(chunkData, int64(chunk*chunkSize))
				if err != nil {
					errs <- err
					return
				}

				c <- 1
			}()
		}()
	}
	finished := 0
	for {
		select {
		case <-c:
			finished++
			if finished == file.Chunks {
				return nil
			}
		case err := <-errs:
			return err
		}
	}
}

func (filen *Filen) UploadFile(sourcePath string, parentUUID string) error {
	plaintextData, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	fileUUID := uuid.New().String()
	key := crypto.GenerateRandomString(32)
	uploadKey := crypto.GenerateRandomString(32)

	data, err := crypto.EncryptData(plaintextData, key)

	err = filen.client.UploadFileChunk(fileUUID, 0, parentUUID, uploadKey, data)
	if err != nil {
		return err
	}

	name := filepath.Base(sourcePath)
	nameEncrypted, err := crypto.EncryptMetadata(name, key)
	if err != nil {
		return err
	}
	nameHashed := hex.EncodeToString(crypto.RunSHA521([]byte(name)))
	mime, err := crypto.EncryptMetadata("text/plain", key)
	if err != nil {
		return err
	}
	sizeEncrypted, err := crypto.EncryptMetadata(strconv.Itoa(len(plaintextData)), key)
	if err != nil {
		return err
	}

	metadata := struct {
		Name         string `json:"name"`
		Size         int    `json:"size"`
		Mime         string `json:"mime"`
		Key          string `json:"key"`
		LastModified int    `json:"lastModified"`
		Created      int    `json:"created"`
	}{name, len(plaintextData), "text/plain", key, int(time.Now().Unix()), int(time.Now().Unix())}
	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	fmt.Println(string(metadataStr))
	metadataEncrypted, err := crypto.EncryptMetadata(string(metadataStr), filen.masterKey())
	if err != nil {
		return err
	}

	err = filen.client.UploadDone(client.UploadDonePayload{
		UUID:       fileUUID,
		Name:       nameEncrypted,
		NameHashed: nameHashed,
		Size:       sizeEncrypted,
		Chunks:     1,
		Mime:       mime,
		Rm:         crypto.GenerateRandomString(32),
		Metadata:   metadataEncrypted,
		Version:    2,
		UploadKey:  uploadKey,
	})
	if err != nil {
		return err
	}

	return nil
}
