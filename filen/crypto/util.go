package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"golang.org/x/crypto/pbkdf2"
	"io"
)

func runPBKDF2(password string, salt string, iterations int, bitLength int) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), iterations, bitLength/8, sha512.New)
}

func runSHA521(s string) []byte {
	hasher := sha512.New()
	hasher.Write([]byte(s))
	return hasher.Sum(nil)
}

func runAES256GCMDecryption(key []byte, nonce []byte, ciphertext []byte, authTag []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	result, err := gcm.Open(nil, nonce, ciphertext, authTag)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func runAES256GCMEncryption(key []byte, plaintext []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	var result []byte
	result = gcm.Seal(nil, nonce, plaintext, nil)
	return result, nil
}
