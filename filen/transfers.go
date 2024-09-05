package filen

import (
	"encoding/hex"
	"encoding/json"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/client"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
	"github.com/google/uuid"
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

const (
	maxConcurrentReaders = 16
	maxConcurrentUploads = 16
)

// UploadFile uploads a local file (specified by path) to a cloud directory (specified by UUID).
func (filen *Filen) UploadFile(source *os.File, parentUUID string) error {
	// initialize random keys
	fileUUID := uuid.New().String()
	key := []byte(crypto.GenerateRandomString(32))
	uploadKey := crypto.GenerateRandomString(32)

	readSem := make(chan int, maxConcurrentReaders)
	uploadSem := make(chan int, maxConcurrentUploads)
	cFinished := make(chan int)
	errs := make(chan error)

	// read chunks, encrypt and upload concurrently
	stat, err := source.Stat()
	if err != nil {
		return err
	}
	fileName := stat.Name()
	sourceSize := stat.Size()
	chunks := int(math.Ceil(float64(sourceSize) / float64(chunkSize)))
	for chunk := 0; chunk < chunks; chunk++ {
		go func() {
			readSem <- 1
			defer func() { <-readSem }()

			// read chunk
			chunkStart := chunk * chunkSize
			chunkEnd := int(math.Min(float64(chunk+1)*float64(chunkSize), float64(sourceSize)))
			plaintextChunkData := make([]byte, chunkEnd-chunkStart)
			_, err := source.ReadAt(plaintextChunkData, int64(chunkStart))
			if err != nil {
				errs <- err
			}

			// encrypt data
			chunkData, err := crypto.EncryptData(plaintextChunkData, key)
			if err != nil {
				errs <- err
			}

			// upload chunk
			go func() {
				uploadSem <- 1
				defer func() { <-uploadSem }()

				err = filen.client.UploadFileChunk(fileUUID, chunk, parentUUID, uploadKey, chunkData)
				if err != nil {
					errs <- err
				}

				cFinished <- 1
			}()
		}()
	}

	// wait for all to finish, or return error
	chunkUploadsFinished := 0
WaitForAll:
	for {
		select {
		case <-cFinished:
			chunkUploadsFinished++
			if chunkUploadsFinished == chunks {
				break WaitForAll
			}
		case err := <-errs:
			return err
		}
	}

	// encrypt info about file
	nameEncrypted, err := crypto.EncryptMetadata(fileName, key)
	if err != nil {
		return err
	}
	nameHashed := hex.EncodeToString(crypto.RunSHA521([]byte(fileName)))
	mime, err := crypto.EncryptMetadata("text/plain", key)
	if err != nil {
		return err
	}
	sizeEncrypted, err := crypto.EncryptMetadata(strconv.Itoa(int(sourceSize)), key)
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
	}{fileName, int(sourceSize), "text/plain", string(key), int(time.Now().Unix()), int(time.Now().Unix())}
	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	metadataEncrypted, err := crypto.EncryptMetadata(string(metadataStr), filen.CurrentMasterKey())
	if err != nil {
		return err
	}

	// mark upload as done
	_, err = filen.client.UploadDone(client.UploadDoneRequest{
		UUID:       fileUUID,
		Name:       nameEncrypted,
		NameHashed: nameHashed,
		Size:       sizeEncrypted,
		Chunks:     chunks,
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
