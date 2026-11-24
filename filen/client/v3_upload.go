package client

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
)

// V3UploadResponse represents the response structure from the upload endpoint.
type V3UploadResponse struct {
	Bucket string `json:"bucket"`
	Region string `json:"region"`
}

// PostV3Upload uploads a file chunk to the Filen storage backend.
// It handles the direct binary upload to the ingest servers and returns storage metadata.
//
// data is not read after PostV3Upload returns, so the caller can reuse it.
func (c *Client) PostV3Upload(ctx context.Context, uuid string, chunkIdx int64, parentUUID string, uploadKey string, data []byte) (*V3UploadResponse, error) {
	// build request
	dataHash := hex.EncodeToString(crypto.RunSHA512(data))
	url := IngestURL(fmt.Sprintf("/v3/upload?uuid=%s&index=%v&parent=%s&uploadKey=%s&hash=%s",
		uuid, chunkIdx, parentUUID, uploadKey, dataHash))
	method := "POST"
	// Can't use the standard Client.RequestData because our request body is raw bytes
	// The transport may read the body after Do returns (see http.RoundTripper)
	// and can't be relied on to close it, so cut every body it was given off
	// from data on return instead.
	bodies := &uploadBodies{}
	req, err := c.buildReaderRequest(ctx, method, url, bodies.newBody(data))
	if err != nil {
		return nil, err
	}
	req.ContentLength = int64(len(data))
	req.GetBody = func() (io.ReadCloser, error) { return bodies.newBody(data), nil }
	defer bodies.revoke()

	response, err := handleRequest(req, &c.httpClient, method, url)
	if err != nil {
		return nil, err
	}

	if !response.Status {
		return nil, errors.New("Cannot upload file chunk: " + response.Message)
	}

	uploadResponse := &V3UploadResponse{}
	err = response.IntoData(uploadResponse)
	if err != nil {
		return nil, err
	}
	return uploadResponse, nil
}

// errBodyRevoked is returned when a request body is read after PostV3Upload returned.
var errBodyRevoked = errors.New("upload request body read after the upload returned")

// uploadBodies hands out request bodies reading the same data, all of which
// can be revoked at once.
type uploadBodies struct {
	mu      sync.Mutex
	revoked bool
}

// newBody returns a request body that reads data until revoke is called.
func (u *uploadBodies) newBody(data []byte) io.ReadCloser {
	return &revocableBody{bodies: u, r: bytes.NewReader(data)}
}

// revoke makes every body fail further reads. It waits for any read in
// progress, so data isn't touched once it returns.
func (u *uploadBodies) revoke() {
	u.mu.Lock()
	u.revoked = true
	u.mu.Unlock()
}

// revocableBody is a request body from uploadBodies.newBody.
//
// It doesn't embed *bytes.Reader, so that io.Copy can't bypass Read with WriteTo.
type revocableBody struct {
	bodies *uploadBodies
	r      *bytes.Reader
}

// Read reads from data, or returns errBodyRevoked once revoked.
func (b *revocableBody) Read(p []byte) (int, error) {
	b.bodies.mu.Lock()
	defer b.bodies.mu.Unlock()
	if b.bodies.revoked {
		return 0, errBodyRevoked
	}
	return b.r.Read(p)
}

// Close does nothing; reads stop when the bodies are revoked.
func (b *revocableBody) Close() error {
	return nil
}
