package filen

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/zeebo/blake3"

	"github.com/FilenCloudDienste/filen-sdk-go/filen/client"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/types"
)

func newTestFileUpload(t *testing.T) *FileUpload {
	t.Helper()
	file, err := types.NewIncompleteFile(crypto.FileEncryptionVersion(3), "test.bin", "", time.Now(), time.Now(), types.NewRootDirectory("root-uuid"))
	if err != nil {
		t.Fatal(err)
	}
	return (&Filen{}).NewFileUpload(file)
}

func TestChunkBufferRefusesConsumedPlaintext(t *testing.T) {
	fu := newTestFileUpload(t)
	key := &fu.EncryptionKey
	plaintext := []byte("PLAINTEXT-CHUNK-PLAINTEXT-CHUNK-")

	b := fu.NewChunkBuffer()
	if _, err := b.seal(key); !errors.Is(err, ErrChunkConsumed) {
		t.Fatalf("seal before Fill: got %v, want ErrChunkConsumed", err)
	}
	n, err := b.Fill(bytes.NewReader(plaintext))
	if err != nil || n != len(plaintext) {
		t.Fatalf("Fill = %d, %v, want %d, nil", n, err, len(plaintext))
	}
	if !bytes.Equal(b.Plaintext(), plaintext) {
		t.Fatalf("Plaintext() = %q, want %q", b.Plaintext(), plaintext)
	}
	if _, err := b.seal(key); err != nil {
		t.Fatal(err)
	}
	// A retry that doesn't refill must fail rather than encrypt the ciphertext again.
	if _, err := b.seal(key); !errors.Is(err, ErrChunkConsumed) {
		t.Fatalf("second seal without Fill: got %v, want ErrChunkConsumed", err)
	}
	if b.Plaintext() != nil {
		t.Fatal("Plaintext() returned ciphertext after seal")
	}

	// Nor may a refill that fails part way after a good one, which leaves a mix
	// of both reads in the buffer.
	if _, err := b.Fill(bytes.NewReader(plaintext)); err != nil {
		t.Fatal(err)
	}
	errRead := errors.New("read failed")
	_, err = b.Fill(io.MultiReader(bytes.NewReader(plaintext[:5]), iotest.ErrReader(errRead)))
	if !errors.Is(err, errRead) {
		t.Fatalf("Fill from failing reader: got %v, want %v", err, errRead)
	}
	if _, err := b.seal(key); !errors.Is(err, ErrChunkConsumed) {
		t.Fatalf("seal after failed Fill: got %v, want ErrChunkConsumed", err)
	}
	if b.Plaintext() != nil {
		t.Fatal("Plaintext() returned data after failed Fill")
	}

	if _, err := b.Fill(bytes.NewReader(plaintext)); err != nil {
		t.Fatal(err)
	}
	encrypted, err := b.seal(key)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := key.DecryptData(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted %q, want %q", decrypted, plaintext)
	}
}

func TestChunkBufferRefusesEmptyChunk(t *testing.T) {
	fu := newTestFileUpload(t)
	b := fu.NewChunkBuffer()
	// Filling from a reader that's already at io.EOF reads nothing.
	if n, err := b.Fill(bytes.NewReader(nil)); err != nil || n != 0 {
		t.Fatalf("Fill = %d, %v, want 0, nil", n, err)
	}
	if _, err := b.seal(&fu.EncryptionKey); !errors.Is(err, ErrChunkEmpty) {
		t.Fatalf("seal after empty Fill: got %v, want ErrChunkEmpty", err)
	}
}

func TestChunkBufferFillReturnsReadErrors(t *testing.T) {
	b := newTestFileUpload(t).NewChunkBuffer()
	for _, tc := range []struct {
		name string
		r    io.Reader
	}{
		// A reader's own io.ErrUnexpectedEOF means the data was cut off, not that it ended.
		{"UnexpectedEOFAfterData", io.MultiReader(bytes.NewReader(make([]byte, 7)), iotest.ErrReader(io.ErrUnexpectedEOF))},
		{"UnexpectedEOFAtStart", iotest.ErrReader(io.ErrUnexpectedEOF)},
		// An error returned with the bytes that fill the chunk must not be lost.
		{"ErrorWithLastBytes", &errWithLastRead{r: bytes.NewReader(make([]byte, ChunkSize)), err: io.ErrUnexpectedEOF}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := b.Fill(tc.r); !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("Fill: got %v, want io.ErrUnexpectedEOF", err)
			}
			if b.Plaintext() != nil {
				t.Fatal("buffer holds plaintext after failed Fill")
			}
		})
	}
}

// errWithLastRead returns err together with the last bytes of r.
type errWithLastRead struct {
	r   *bytes.Reader
	err error
}

func (e *errWithLastRead) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if err == nil && e.r.Len() == 0 {
		err = e.err
	}
	return n, err
}

// recordingIngest stores the body of every upload by chunk index.
type recordingIngest struct {
	mu     sync.Mutex
	chunks map[int][]byte
}

func (ri *recordingIngest) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	index, err := strconv.Atoi(req.URL.Query().Get("index"))
	if err != nil {
		return nil, err
	}
	ri.mu.Lock()
	ri.chunks[index] = body
	ri.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":true,"data":{"bucket":"b","region":"r"}}`))),
		Request:    req,
	}, nil
}

func TestUploadFileChunks(t *testing.T) {
	ingest := &recordingIngest{}
	// The zero UnauthorizedClient sends through http.DefaultTransport.
	oldTransport := http.DefaultTransport
	http.DefaultTransport = ingest
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	api := &Filen{Client: (&client.UnauthorizedClient{}).Authorize("test-api-key")}

	for _, size := range []int{1, ChunkSize, 2*ChunkSize + 1000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			ingest.chunks = map[int][]byte{}
			fu := newTestFileUpload(t)
			plaintext := make([]byte, size)
			_, _ = rand.New(rand.NewSource(int64(size))).Read(plaintext)

			uploaded, err := api.uploadFileChunks(context.Background(), fu, make(chan client.V3UploadResponse, 1), bytes.NewReader(plaintext))
			if err != nil {
				t.Fatal(err)
			}
			if uploaded != int64(size) {
				t.Errorf("uploaded %d bytes, want %d", uploaded, size)
			}

			wantChunks := (size + ChunkSize - 1) / ChunkSize
			if len(ingest.chunks) != wantChunks {
				t.Fatalf("uploaded %d chunks, want %d", len(ingest.chunks), wantChunks)
			}
			for i := range wantChunks {
				decrypted, err := fu.EncryptionKey.DecryptData(ingest.chunks[i])
				if err != nil {
					t.Fatalf("chunk %d: %v", i, err)
				}
				if !bytes.Equal(decrypted, plaintext[i*ChunkSize:min((i+1)*ChunkSize, size)]) {
					t.Errorf("chunk %d doesn't decrypt to its plaintext", i)
				}
			}

			want := blake3.New()
			_, _ = want.Write(plaintext)
			if got := fu.Hasher.Sum(nil); !bytes.Equal(got, want.Sum(nil)) {
				t.Errorf("file hash %x, want %x", got, want.Sum(nil))
			}
		})
	}
}
