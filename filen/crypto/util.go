package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
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

// runPBKDF2 generates a PBKDF2 (SHA-512) key //TODO
func runPBKDF2(password string, salt string, iterations int, bitLength int) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), iterations, bitLength/8, sha512.New)
}

// runSHA512 generates a SHA-512 hash
func runSHA521(s string) []byte {
	hasher := sha512.New()
	hasher.Write([]byte(s))
	return hasher.Sum(nil)
}

// runAES256GCMDecrypt generates an AES256-GCM
func runAES256GCMDecrypt(key []byte, nonce []byte, ciphertext []byte, authTag []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	var result []byte
	result, err = gcm.Open(result, nonce, ciphertext, authTag)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// runAES256GCMEncrypt generates an AES256-GCM
func runAES256GCMEncrypt(key []byte, plaintext []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonce := []byte(randString(gcm.NonceSize()))
	var result []byte
	result = gcm.Seal(nil, nonce, plaintext, nil)
	return result, nil
}
