package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"golang.org/x/crypto/pbkdf2"
	"math/big"
)

func runPBKDF2(password string, salt string, iterations int, bitLength int) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), iterations, bitLength/8, sha512.New)
}

func RunSHA521(b []byte) []byte {
	hasher := sha512.New()
	hasher.Write(b)
	return hasher.Sum(nil)
}

func runAES256GCMDecryption(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	result, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GenerateRandomString(length int) string {
	runes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	str := ""
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(runes))))
		if err != nil {
			panic(err)
		}
		str += string(runes[idx.Int64()])
	}
	return str
}

func runAES256GCMEncryption(key []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	var result []byte
	result = gcm.Seal(nil, nonce, plaintext, nil)
	return result, nil
}
