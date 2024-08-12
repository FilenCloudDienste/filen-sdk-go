package filen

import (
	"encoding/hex"
	"encoding/json"
	"filen/filen-sdk-go/filen/client"
	"filen/filen-sdk-go/filen/crypto"
	"filen/filen-sdk-go/filen/util"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// File represents a file on the cloud drive.
type File struct {
	UUID          string    // the UUID of the cloud item
	Name          string    // the file name
	Size          int64     // the file size in bytes
	MimeType      string    // the MIME type of the file
	EncryptionKey []byte    // the key used to encrypt the file data
	Created       time.Time // when the file was created
	LastModified  time.Time // when the file was last modified
	ParentUUID    string    // the [Directory.UUID] of the file's parent directory
	Favorited     bool      // whether the file is marked a favorite
	Region        string    // the file's storage region
	Bucket        string    // the file's storage bucket
	Chunks        int       // how many 1 MiB chunks the file is partitioned into
}

// Directory represents a directory on the cloud drive.
type Directory struct {
	UUID       string    // the UUID of the cloud item
	Name       string    // the directory name
	ParentUUID string    // the [Directory.UUID] of the directory's parent directory (or zero value for the root directory)
	Color      string    // the color assigned to the directory (zero value means default color)
	Created    time.Time // when the directory was created
	Favorited  bool      // whether the directory is marked a favorite
}

// GetBaseFolderUUID fetches the UUID of the cloud drive's root directory.
func (filen *Filen) GetBaseFolderUUID() (string, error) {
	userBaseFolder, err := filen.client.GetUserBaseFolder()
	if err != nil {
		return "", err
	}
	return userBaseFolder.UUID, nil
}

// ReadDirectory fetches the files and directories that are children of a directory (specified by UUID).
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
			EncryptionKey: []byte(metadata.Key),
			Created:       util.TimestampToTime(int64(file.Timestamp)),
			LastModified:  util.TimestampToTime(int64(metadata.LastModified)),
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
			Created:    util.TimestampToTime(int64(directory.Timestamp)),
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

// DownloadFile downloads a file from the cloud drive into a local destination.
func (filen *Filen) DownloadFile(file *File, destination *os.File) error {
	downloadSem := make(chan int, maxConcurrentDownloads)
	writeSem := make(chan int, maxConcurrentWriters)
	cFinished := make(chan int)
	errs := make(chan error)

	// download chunks and write to disk concurrently
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

				cFinished <- 1
			}()
		}()
	}

	// wait for all to finish, or return error
	finished := 0
	for {
		select {
		case <-cFinished:
			finished++
			if finished == file.Chunks {
				return nil
			}
		case err := <-errs:
			return err
		}
	}
}

// UploadFile uploads a local file (specified by path) to a cloud directory (specified by UUID).
func (filen *Filen) UploadFile(sourcePath string, parentUUID string) error {
	// read file
	plaintextData, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// initialize random keys
	fileUUID := uuid.New().String()
	key := []byte(crypto.GenerateRandomString(32))
	uploadKey := crypto.GenerateRandomString(32)

	// encrypt data
	data, err := crypto.EncryptData(plaintextData, key)

	// upload chunks
	err = filen.client.UploadFileChunk(fileUUID, 0, parentUUID, uploadKey, data)
	if err != nil {
		return err
	}
	//TODO handle multiple file chunks

	// encrypt info about file
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

	// encrypt file metadata
	metadata := struct {
		Name         string `json:"name"`
		Size         int    `json:"size"`
		Mime         string `json:"mime"`
		Key          string `json:"key"`
		LastModified int    `json:"lastModified"`
		Created      int    `json:"created"`
	}{name, len(plaintextData), "text/plain", string(key), int(time.Now().Unix()), int(time.Now().Unix())}
	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	metadataEncrypted, err := crypto.EncryptMetadata(string(metadataStr), filen.CurrentMasterKey())
	if err != nil {
		return err
	}

	// mark upload as done
	_, err = filen.client.UploadDone(client.UploadDonePayload{
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
