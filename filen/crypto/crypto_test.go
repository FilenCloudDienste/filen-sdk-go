package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDataInPlace(t *testing.T) {
	key, err := NewEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("PLAINTEXT-CHUNK-PLAINTEXT-CHUNK-")
	buf := make([]byte, NonceSize+len(plaintext), NonceSize+len(plaintext)+key.Cipher.Overhead())

	copy(buf[NonceSize:], plaintext)
	encrypted := key.EncryptDataInPlace(buf)
	if &encrypted[0] != &buf[0] {
		t.Fatal("EncryptDataInPlace did not reuse buf")
	}
	decrypted, err := key.DecryptData(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted %q, want %q", decrypted, plaintext)
	}
}
