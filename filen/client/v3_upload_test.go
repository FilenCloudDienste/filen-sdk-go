package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

// fakeIngest answers every request at once with a successful upload response,
// after passing the request to the function.
type fakeIngest func(req *http.Request)

func (f fakeIngest) RoundTrip(req *http.Request) (*http.Response, error) {
	f(req)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":true,"data":{"bucket":"b","region":"r"}}`))),
		Request:    req,
	}, nil
}

func newTestClient(rt http.RoundTripper) *Client {
	return &Client{UnauthorizedClient: UnauthorizedClient{httpClient: http.Client{Transport: rt}}}
}

func TestPostV3UploadBodyUnreadableAfterReturn(t *testing.T) {
	data := []byte("ciphertext")
	returned := make(chan struct{})
	lateRead := make(chan error, 1)
	c := newTestClient(fakeIngest(func(req *http.Request) {
		if req.ContentLength != int64(len(data)) {
			t.Errorf("ContentLength = %d, want %d", req.ContentLength, len(data))
		}
		if req.GetBody == nil {
			t.Error("GetBody is nil, so the transport can't replay the body")
		}
		// The transport may still read the body after Do returns, see http.RoundTripper.
		go func() {
			<-returned
			n, err := req.Body.Read(make([]byte, len(data)))
			if n != 0 {
				err = errors.New("read data after PostV3Upload returned")
			}
			lateRead <- err
		}()
	}))

	_, err := c.PostV3Upload(context.Background(), "uuid", 0, "parent", "key", data)
	if err != nil {
		t.Fatal(err)
	}
	close(returned)
	if err := <-lateRead; !errors.Is(err, errBodyRevoked) {
		t.Errorf("late read: got %v, want errBodyRevoked", err)
	}
}

func TestPostV3UploadBodySwappedWithoutClose(t *testing.T) {
	data := []byte("ciphertext")
	// Like rclone's --dump curl, which copies the body and never closes the original.
	c := newTestClient(fakeIngest(func(req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(body, data) {
			t.Error("request body differs from data")
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}))

	done := make(chan error, 1)
	go func() {
		_, err := c.PostV3Upload(context.Background(), "uuid", 0, "parent", "key", data)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PostV3Upload hung waiting for a body that is never closed")
	}
}

func TestPostV3UploadBodyReadDuringReturn(t *testing.T) {
	// Too big to read one byte at a time before PostV3Upload returns.
	data := make([]byte, 16<<20)
	readerDone := make(chan struct{})
	c := newTestClient(fakeIngest(func(req *http.Request) {
		// Keep reading while PostV3Upload returns, so any read that touches data
		// without holding the lock races with the caller's writes below (-race).
		reading := make(chan struct{})
		go func() {
			defer close(readerDone)
			p := make([]byte, 1)
			for i := 0; ; i++ {
				if _, err := req.Body.Read(p); err != nil {
					return
				}
				if i == 0 {
					close(reading)
				}
			}
		}()
		<-reading
	}))

	_, err := c.PostV3Upload(context.Background(), "uuid", 0, "parent", "key", data)
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		data[i] = 0
	}
	<-readerDone
}
