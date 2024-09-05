package filen

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/client"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
	"github.com/google/uuid"
	"io"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	maxConcurrentDownloads = 16
	maxConcurrentWriters   = 16
	chunkSize              = 1048576
)

// DownloadFileToDisk downloads a file from the cloud drive into a local destination on disk.
func (filen *Filen) DownloadFileToDisk(file *File, destination *os.File) error {
	err := filen.DownloadFile(file, func(chunk int, data []byte) error {
		_, err := destination.WriteAt(data, int64(chunk*chunkSize))
		return err
	})
	return err
}

// DownloadFileInMemory downloads a file from the cloud drive and stores its bytes in memory.
func (filen *Filen) DownloadFileInMemory(file *File) ([]byte, error) {
	fileData := make([]byte, file.Size)
	err := filen.DownloadFile(file, func(chunk int, data []byte) error {
		chunkStart := chunk * chunkSize
		chunkEnd := int(math.Min(float64(chunk+1)*float64(chunkSize), float64(file.Size)))
		copy(fileData[chunkStart:chunkEnd], data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fileData, nil
}

// DownloadFile downloads a file from the cloud drive and calls the chunkHandler for every received chunk.
func (filen *Filen) DownloadFile(file *File, chunkHandler func(chunk int, data []byte) error) error {
	downloadSem := make(chan int, maxConcurrentDownloads)
	writeSem := make(chan int, maxConcurrentWriters)
	cFinished := make(chan int)
	errs := make(chan error)

	// download chunks, decrypt and write to disk concurrently
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

				err = chunkHandler(chunk, chunkData)
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

const maxConcurrentUploads = 16

// UploadFile uploads data to a cloud file (specified by its name and its parent directory's UUID).
func (filen *Filen) UploadFile(fileName string, parentUUID string, data io.Reader) (*File, error) {
	uploaderSem := make(chan int, maxConcurrentUploads)
	uploadFinished := make(chan int)
	errs := make(chan error)

	var region, bucket string

	// uploader
	fileUUID := uuid.New().String()
	key := []byte(crypto.GenerateRandomString(32))
	uploadKey := crypto.GenerateRandomString(32)
	uploader := func(chunkData []byte, chunkIdx int) {
		uploaderSem <- 1
		defer func() { <-uploaderSem }()

		// encrypt data
		chunkData, err := crypto.EncryptData(chunkData, key)
		if err != nil {
			errs <- err
		}

		// upload chunk
		uploadRegion, uploadBucket, err := filen.client.UploadFileChunk(fileUUID, chunkIdx, parentUUID, uploadKey, chunkData)
		if err != nil {
			errs <- err
		}
		region = uploadRegion
		bucket = uploadBucket

		uploadFinished <- 1
	}

	// read chunks
	b := make([]byte, chunkSize)
	chunk := make([]byte, 0)
	chunks := 0
	totalBytes := 0
	for {
		n, err := data.Read(b)
		totalBytes += n
		chunk = append(chunk, b[:n]...)
		if len(chunk) >= chunkSize || (err == io.EOF && len(chunk) > 0) {
			chunkData := chunk
			if len(chunk) > chunkSize {
				chunkData = chunk[:chunkSize]
			}
			chunk = chunk[len(chunkData):]

			go uploader(chunkData, chunks)
			chunks++
		}
		if err == io.EOF {
			if totalBytes == 0 {
				return nil, errors.New("empty uploads are not supported")
			}
			break
		}
		if err != nil {
			return nil, err
		}
	}

	// wait for all to finish, or return error
	uploadsFinished := 0
	if chunks != 0 {
	WaitForAll:
		for {
			select {
			case <-uploadFinished:
				uploadsFinished++
				if uploadsFinished == chunks {
					break WaitForAll
				}
			case err := <-errs:
				return nil, err
			}
		}
	}

	// encrypt info about file
	nameEncrypted, err := crypto.EncryptMetadata(fileName, key)
	if err != nil {
		return nil, err
	}
	nameHashed := hex.EncodeToString(crypto.RunSHA521([]byte(fileName)))
	mimeType, err := crypto.EncryptMetadata("text/plain", key)
	if err != nil {
		return nil, err
	}
	sizeEncrypted, err := crypto.EncryptMetadata(strconv.Itoa(totalBytes), key)
	if err != nil {
		return nil, err
	}

	// encrypt file metadata
	metadata := struct {
		Name         string `json:"name"`
		Size         int    `json:"size"`
		MimeType     string `json:"mime"`
		Key          string `json:"key"`
		LastModified int    `json:"lastModified"`
		Created      int    `json:"created"`
	}{fileName, totalBytes, "text/plain", string(key), int(time.Now().Unix()), int(time.Now().Unix())}
	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	metadataEncrypted, err := crypto.EncryptMetadata(string(metadataStr), filen.CurrentMasterKey())
	if err != nil {
		return nil, err
	}

	// mark upload as done
	response, err := filen.client.UploadDone(client.UploadDoneRequest{
		UUID:       fileUUID,
		Name:       nameEncrypted,
		NameHashed: nameHashed,
		Size:       sizeEncrypted,
		Chunks:     chunks,
		MimeType:   mimeType,
		Rm:         crypto.GenerateRandomString(32),
		Metadata:   metadataEncrypted,
		Version:    2,
		UploadKey:  uploadKey,
	})
	if err != nil {
		return nil, err
	}

	return &File{
		UUID:          fileUUID,
		Name:          fileName,
		Size:          int64(totalBytes),
		MimeType:      "application/octet-stream", //TODO correct mime type
		EncryptionKey: []byte(uploadKey),
		Created:       time.Now(), //TODO really?
		LastModified:  time.Now(),
		ParentUUID:    parentUUID,
		Favorited:     false,
		Region:        region,
		Bucket:        bucket,
		Chunks:        response.Chunks,
	}, nil
}
