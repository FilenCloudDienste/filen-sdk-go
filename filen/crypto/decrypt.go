package crypto

import (
	"encoding/base64"
	"fmt"
)

type EncryptedString string

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
		} else {
			return decrypted, nil
		}
	}
	return "", &AllKeysFailedError{errors}
}

func DecryptMetadata(metadata EncryptedString, key string) (string, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)

	iv := metadata[3:15]
	encrypted, err := base64.StdEncoding.DecodeString(string(metadata[15:]))
	if err != nil {
		return "", err
	}

	result, err := runAES256GCMDecryption(derivedKey, []byte(iv), encrypted, nil)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func DecryptDataAllKeys(data []byte, keys []string) ([]byte, error) {
	errors := make([]error, 0)
	for _, key := range keys {
		decrypted, err := DecryptData(data, key)
		if err != nil {
			errors = append(errors, err)
		} else {
			return decrypted, nil
		}
	}
	return nil, &AllKeysFailedError{errors}
}

func DecryptData(data []byte, key string) ([]byte, error) {
	nonce, ciphertext := data[:12], data[12:]
	result, err := runAES256GCMDecryption([]byte(key), nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}
