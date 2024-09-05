package filen

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/util"
	"strings"
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

// PathToUUID identifies a cloud item by its path and returns its UUID.
// Set the requireDirectory to differentiate between files and directories with the same path
// (otherwise, the file will be found).
func (filen *Filen) PathToUUID(path string, requireDirectory bool) (string, error) {
	baseFolderUUID, err := filen.GetBaseFolderUUID()
	if err != nil {
		return "", err
	}

	segments := strings.Split(path, "/")

	currentPath := ""
	currentUUID := baseFolderUUID
SegmentsLoop:
	for _, segment := range segments {
		if segment == "" {
			continue
		}

		files, directories, err := filen.ReadDirectory(currentUUID)
		if err != nil {
			return "", err
		}
		if !requireDirectory {
			for _, file := range files {
				if file.Name == segment {
					currentUUID = file.UUID
					currentPath = currentPath + "/" + segment
					continue SegmentsLoop
				}
			}
		}
		for _, directory := range directories {
			if directory.Name == segment {
				currentUUID = directory.UUID
				currentPath = currentPath + "/" + segment
				continue SegmentsLoop
			}
		}
		return "", errors.New(fmt.Sprintf("item %s not found in directory %s", segment, currentPath))
	}
	return currentUUID, nil
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
