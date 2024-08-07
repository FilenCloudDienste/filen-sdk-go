package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"golang.org/x/crypto/pbkdf2"
	"math/rand"
)

func randString(n int) string {
	// see https://stackoverflow.com/a/22892986/13164753
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// byte array utility

// encodeHex encodes a byte array as a hex string
func encodeHex(b []byte) string {
	return hex.EncodeToString(b)
}

// encodeBase64 encodes a byte array as a base64-encoded string
func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

// decodeBase64 decodes a base64-encoded string
func decodeBase64(s string) ([]byte, error) {
	bytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// cryptographic utility

// runPBKDF2 generates a PBKDF2 (SHA-512) key using
func runPBKDF2(password string, salt string, iterations int, bitLength int) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), iterations, bitLength/8, sha512.New)
}

// runSHA512 generates a SHA-512 hash
func runSHA521(s string) []byte {
	hasher := sha512.New()
	hasher.Write([]byte(s))
	return hasher.Sum(nil)
}

// runAES256GCM generates an AES256-GCM
// TODO
func runAES256GCM(key []byte, nonce []byte, ciphertext []byte, additionalData []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	var result []byte
	result, err = gcm.Open(result, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, err
	}
	return result, nil
}
