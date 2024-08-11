package crypto

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

type EncryptedString string

// other

func DeriveKeyFromPassword(password string, salt string, iterations int, bitLength int) []byte {
	return runPBKDF2(password, salt, iterations, bitLength)
}

func GeneratePasswordAndMasterKey(rawPassword string, salt string) (derivedMasterKey string, derivedPassword string) {
	derivedKey := hex.EncodeToString(DeriveKeyFromPassword(rawPassword, salt, 200000, 512))
	derivedMasterKey, derivedPassword = derivedKey[:len(derivedKey)/2], derivedKey[len(derivedKey)/2:]
	derivedPassword = fmt.Sprintf("%032x", RunSHA521([]byte(derivedPassword)))
	return
}

// encryption

func EncryptMetadata(metadata string, key string) (EncryptedString, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)
	nonce := []byte(GenerateRandomString(12))
	result, err := runAES256GCMEncryption(derivedKey, nonce, []byte(metadata))
	if err != nil {
		return "", err
	}
	resultStr := base64.StdEncoding.EncodeToString(result)
	return EncryptedString("002" + string(nonce) + resultStr), nil
}

func EncryptData(data []byte, key string) ([]byte, error) {
	nonce := []byte(GenerateRandomString(12))
	result, err := runAES256GCMEncryption([]byte(key), nonce, data)
	if err != nil {
		return nil, err
	}
	return append(nonce, result...), nil
}

// decryption

type AllKeysFailedError struct {
	Errors []error
}

func (e *AllKeysFailedError) Error() string {
	return fmt.Sprintf("all keys failed: %v", e.Errors)
}

func DecryptMetadataAllKeys(metadata EncryptedString, keys []string) (string, error) {
	errors := make([]error, 0)
	for _, key := range keys {
		decrypted, err := DecryptMetadata(metadata, key)
		if err != nil {
			errors = append(errors, err)
			//log.Fatalf("%s %s %s", key, metadata, err)
		} else {
			return decrypted, nil
		}
	}
	return "", &AllKeysFailedError{errors}
}

func DecryptMetadata(metadata EncryptedString, key string) (string, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)
	nonce := metadata[3:15]
	encrypted, err := base64.StdEncoding.DecodeString(string(metadata[15:]))
	if err != nil {
		return "", err
	}

	result, err := runAES256GCMDecryption(derivedKey, []byte(nonce), encrypted)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func DecryptData(data []byte, key string) ([]byte, error) {
	nonce, ciphertext := data[:12], data[12:]
	result, err := runAES256GCMDecryption([]byte(key), nonce, ciphertext)
	if err != nil {
		return nil, err
	}
	return result, nil
}
