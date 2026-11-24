package filen

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"strconv"

	"github.com/zeebo/blake3"

	"github.com/FilenCloudDienste/filen-sdk-go/filen/client"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/types"
)

// FileUpload encapsulates the state of an ongoing file upload.
// It tracks the file being uploaded and maintains the upload context,
// encryption key, and hash calculation.
type FileUpload struct {
	types.IncompleteFile           // The file being uploaded
	UploadKey            string    // Random key for this upload session
	Hasher               hash.Hash // For calculating file hash during upload
}

// NewFileUpload creates a new FileUpload structure from an IncompleteFile.
// It initializes a new random upload key and hash calculator for the upload process.
func (api *Filen) NewFileUpload(file *types.IncompleteFile) *FileUpload {
	return &FileUpload{
		IncompleteFile: *file,
		UploadKey:      crypto.GenerateRandomString(32),
		Hasher:         blake3.New(),
	}
}

// ErrChunkConsumed is returned by UploadChunk when the ChunkBuffer holds no
// plaintext: it was never filled, its last Fill failed, or it was already
// encrypted by an earlier UploadChunk.
var ErrChunkConsumed = errors.New("chunk buffer holds no plaintext: Fill it before uploading")

// ErrChunkEmpty is returned by UploadChunk when the last Fill read no data,
// because the reader was already at io.EOF. There is no chunk to upload.
var ErrChunkEmpty = errors.New("chunk buffer is empty: there is no chunk to upload")

// ChunkBuffer holds one chunk of file data for UploadChunk, which encrypts it
// in place so uploads don't allocate.
//
// Call Fill, then UploadChunk. UploadChunk overwrites the plaintext with
// ciphertext even when the upload fails, so to retry, call Fill again with the
// same data. The buffer can be reused for the next chunk.
//
// A retry loop therefore has to re-read the chunk on every attempt, for
// example by seeking back to where the chunk starts:
//
//	for attempt := 0; ; attempt++ {
//		if _, err := r.Seek(chunkStart, io.SeekStart); err != nil {
//			return err
//		}
//		if _, err := b.Fill(r); err != nil {
//			return err
//		}
//		if attempt == 0 {
//			fu.Hasher.Write(b.Plaintext()) // hash each chunk only once
//		}
//		resp, err = api.UploadChunk(ctx, fu, chunkIndex, b)
//		if err == nil || attempt == maxAttempts {
//			break
//		}
//	}
//
// Create a ChunkBuffer with FileUpload.NewChunkBuffer; the zero value can't
// be used. A ChunkBuffer is not safe for concurrent use.
type ChunkBuffer struct {
	buf   []byte // nonce, then the chunk, with capacity for the auth tag
	n     int    // length of the plaintext ready to encrypt
	ready bool   // buf holds plaintext that hasn't been encrypted
}

// NewChunkBuffer returns an empty ChunkBuffer for uploading fu's chunks.
func (fu *FileUpload) NewChunkBuffer() *ChunkBuffer {
	return &ChunkBuffer{
		buf: make([]byte, crypto.NonceSize+ChunkSize, crypto.NonceSize+ChunkSize+fu.EncryptionKey.Cipher.Overhead()),
	}
}

// Fill reads the next chunk from r, up to ChunkSize bytes or until io.EOF,
// and returns its length. Reaching io.EOF is not an error.
//
// If r returns any other error, even with the last bytes of the chunk, Fill
// returns it and the buffer holds no plaintext.
func (b *ChunkBuffer) Fill(r io.Reader) (int, error) {
	b.ready = false
	// Not io.ReadFull, which can't tell a short read at io.EOF from a reader's
	// own io.ErrUnexpectedEOF, and drops errors returned with the last bytes.
	data := b.buf[crypto.NonceSize:]
	n := 0
	for n < len(data) {
		read, err := r.Read(data[n:])
		n += read
		if err == io.EOF {
			break
		}
		if err != nil {
			return n, err
		}
	}
	b.n = n
	b.ready = true
	return n, nil
}

// Plaintext returns the data read by the last Fill, or nil if the buffer holds
// no plaintext because that Fill failed or UploadChunk has encrypted it.
//
// The slice shares the buffer's memory, so UploadChunk turns its contents into
// ciphertext and the next Fill overwrites them. Copy it to keep the data.
func (b *ChunkBuffer) Plaintext() []byte {
	if !b.ready {
		return nil
	}
	return b.buf[crypto.NonceSize : crypto.NonceSize+b.n]
}

// seal encrypts the plaintext in place and returns the nonce followed by the
// ciphertext, ErrChunkConsumed if there is no plaintext ready, or ErrChunkEmpty
// if the plaintext is empty.
func (b *ChunkBuffer) seal(key *crypto.EncryptionKey) ([]byte, error) {
	if !b.ready {
		return nil, ErrChunkConsumed
	}
	if b.n == 0 {
		return nil, ErrChunkEmpty
	}
	b.ready = false
	return key.EncryptDataInPlace(b.buf[:crypto.NonceSize+b.n]), nil
}

// UploadChunk encrypts and uploads a single chunk of a file.
// This function handles the encryption of the chunk before sending it to the server.
// It returns storage details (bucket and region) for the uploaded chunk.
//
// The chunk is encrypted in place, consuming b's plaintext, see ChunkBuffer.
// It returns ErrChunkConsumed if b holds no plaintext, and ErrChunkEmpty if the
// last Fill read no data.
func (api *Filen) UploadChunk(ctx context.Context, fu *FileUpload, chunkIndex int64, b *ChunkBuffer) (*client.V3UploadResponse, error) {
	data, err := b.seal(&fu.EncryptionKey)
	if err != nil {
		return nil, err
	}
	response, err := api.Client.PostV3Upload(ctx, fu.UUID, chunkIndex, fu.ParentUUID, fu.UploadKey, data)
	if err != nil {
		return nil, fmt.Errorf("upload chunk %d: %w", chunkIndex, err)
	}
	return response, nil
}

// makeEmptyRequestFromUploaderNoMeta creates a basic upload request without metadata.
// This is used as a foundation for both regular and empty file uploads.
func (api *Filen) makeEmptyRequestFromUploaderWithMetaAndSize(fu *FileUpload, meta string, size string) (*client.V3UploadEmptyRequest, error) {
	metaKey, err := fu.EncryptionKey.ToMasterKey()
	if err != nil {
		return nil, err
	}

	return &client.V3UploadEmptyRequest{
		UUID:       fu.UUID,
		Name:       metaKey.EncryptMeta(fu.Name),
		NameHashed: api.HashFileName(fu.Name),
		Size:       metaKey.EncryptMeta(size),
		Parent:     fu.ParentUUID,
		MimeType:   metaKey.EncryptMeta(fu.MimeType),
		Metadata:   api.EncryptMeta(meta),
		Version:    api.FileEncryptionVersion,
	}, nil
}

// makeEmptyRequestFromUploader creates a complete upload request for an empty file.
// It includes encrypted metadata and file hash information.
func (api *Filen) makeEmptyRequestFromUploader(fu *FileUpload, fileHash string) (*client.V3UploadEmptyRequest, error) {
	metadata := fu.GetRawMeta(api.FileEncryptionVersion)
	metadata.Size = 0
	metadata.Hash = fileHash

	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal file metadata: %w", err)
	}
	return api.makeEmptyRequestFromUploaderWithMetaAndSize(fu, string(metadataStr), "0")
}

// makeRequestFromUploader creates a complete upload request for a non-empty file.
// It includes encrypted metadata, file size, chunk count, and hash information.
func (api *Filen) makeRequestFromUploader(fu *FileUpload, size int64, fileHash string) (*client.V3UploadDoneRequest, error) {
	metadata := fu.GetRawMeta(api.FileEncryptionVersion)
	metadata.Size = size
	metadata.Hash = fileHash

	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal file metadata: %w", err)
	}
	emptyRequest, err := api.makeEmptyRequestFromUploaderWithMetaAndSize(fu, string(metadataStr), strconv.FormatInt(size, 10))

	if err != nil {
		return nil, fmt.Errorf("make empty request from uploader: %w", err)
	}

	return &client.V3UploadDoneRequest{
		V3UploadEmptyRequest: *emptyRequest,
		Chunks:               (size + ChunkSize - 1) / ChunkSize,
		UploadKey:            fu.UploadKey,
		Rm:                   crypto.GenerateRandomString(32),
	}, nil
}

// completeUpload finalizes the upload of a non-empty file.
// It sends the final metadata to the server and constructs the completed File object.
func (api *Filen) completeUpload(ctx context.Context, fu *FileUpload, bucket string, region string, size int64) (*types.File, error) {
	fileHash := hex.EncodeToString(fu.Hasher.Sum(nil))
	uploadRequest, err := api.makeRequestFromUploader(fu, size, fileHash)
	if err != nil {
		return nil, fmt.Errorf("make request from uploader: %w", err)
	}
	_, err = api.Client.PostV3UploadDone(ctx, *uploadRequest)
	if err != nil {
		return nil, fmt.Errorf("complete upload: %w", err)
	}

	return &types.File{
		IncompleteFile: fu.IncompleteFile,
		Size:           size,
		Region:         region,
		Bucket:         bucket,
		Chunks:         uploadRequest.Chunks,
		Hash:           fileHash,
		Version:        api.FileEncryptionVersion,
	}, nil
}

// completeUploadEmpty finalizes the upload of an empty (zero-byte) file.
// It sends the appropriate metadata to the server and constructs the completed File object.
func (api *Filen) completeUploadEmpty(ctx context.Context, fu *FileUpload) (*types.File, error) {
	fileHash := hex.EncodeToString(fu.Hasher.Sum(nil))
	uploadRequest, err := api.makeEmptyRequestFromUploader(fu, fileHash)
	if err != nil {
		return nil, fmt.Errorf("make request from uploader: %w", err)
	}
	_, err = api.Client.PostV3UploadEmpty(ctx, *uploadRequest)
	if err != nil {
		return nil, fmt.Errorf("complete upload: %w", err)
	}

	return &types.File{
		IncompleteFile: fu.IncompleteFile,
		Size:           0,
		Region:         "",
		Bucket:         "",
		Chunks:         0,
		Hash:           fileHash,
	}, nil
}

// UploadFile uploads a file to Filen cloud storage.
// It uses the provided IncompleteFile for metadata and reads file content from the io.Reader.
// The upload process happens in parallel chunks for efficiency.
// Once upload completes, it returns a File object representing the uploaded file.
//
// This method handles:
// - Chunking the file into manageable pieces
// - Parallel upload of chunks
// - Calculating file hash for integrity verification
// - Finalizing the upload with appropriate metadata
// - Updating shared parent metadata
func (api *Filen) UploadFile(ctx context.Context, file *types.IncompleteFile, r io.Reader) (*types.File, error) {
	fu := api.NewFileUpload(file)
	bucketAndRegion := make(chan client.V3UploadResponse, 1)

	size, err := api.uploadFileChunks(ctx, fu, bucketAndRegion, r)
	if err != nil {
		return nil, err
	}

	return api.CompleteFileUpload(ctx, fu, bucketAndRegion, size)
}

func (api *Filen) uploadFileChunks(ctx context.Context, fu *FileUpload, bucketAndRegion chan client.V3UploadResponse, r io.Reader) (int64, error) {
	size := int64(0)
	b := fu.NewChunkBuffer()
	for i := int64(0); ; i++ {
		n, err := b.Fill(r)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			break
		}
		size += int64(n)
		fu.Hasher.Write(b.Plaintext())
		resp, err := api.UploadChunk(ctx, fu, i, b)
		if err != nil {
			return 0, err
		}
		select { // only care about getting this once
		case bucketAndRegion <- *resp:
		default:
		}
		if n < ChunkSize {
			break
		}
	}

	return size, nil
}

// CompleteFileUpload finalizes an upload after its chunks, if any, were sent with UploadChunk.
// It also adds the new file to the parent directory's shares and public links,
// so a file uploaded chunk by chunk ends up in the same state as one uploaded
// with UploadFile.
func (api *Filen) CompleteFileUpload(ctx context.Context, fu *FileUpload, bucketAndRegion chan client.V3UploadResponse, size int64) (*types.File, error) {
	var (
		completeFile *types.File
		err          error
	)
	if size == 0 {
		completeFile, err = api.completeUploadEmpty(ctx, fu)
		if err != nil {
			return nil, err
		}
	} else {
		select {
		case resp, ok := <-bucketAndRegion:
			if !ok {
				return nil, fmt.Errorf("no chunks successfully uploaded")
			}
			completeFile, err = api.completeUpload(ctx, fu, resp.Bucket, resp.Region, size)
			if err != nil {
				return nil, fmt.Errorf("complete upload: %w", err)
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("context done %w", context.Cause(ctx))
		}
	}

	if err := api.updateItemWithMaybeSharedParent(ctx, completeFile); err != nil {
		return nil, fmt.Errorf("update parent shares and links: %w", err)
	}
	return completeFile, nil
}
