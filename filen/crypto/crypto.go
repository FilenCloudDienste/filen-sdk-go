// Package crypto provides the cryptographic functions required within the SDK.
//
// There are two kinds of decrypted data:
//   - Metadata means any small string data, typically file metadata, but also e.g. directory names.
//   - Data means file content.
package crypto

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// EncryptedString denotes that a string is encrypted and can't be used meaningfully before being decrypted.
type EncryptedString string

// other

// DeriveKeyFromPassword derives a valid key from the raw password.
func DeriveKeyFromPassword(password string, salt string, iterations int, bitLength int) []byte {
	return runPBKDF2(password, salt, iterations, bitLength)
}

// GeneratePasswordAndMasterKey derives a password and a master key from the raw password (used for login).
func GeneratePasswordAndMasterKey(rawPassword string, salt string) (derivedMasterKey string, derivedPassword string) {
	derivedKey := hex.EncodeToString(DeriveKeyFromPassword(rawPassword, salt, 200000, 512))
	derivedMasterKey, derivedPassword = derivedKey[:len(derivedKey)/2], derivedKey[len(derivedKey)/2:]
	derivedPassword = fmt.Sprintf("%032x", RunSHA521([]byte(derivedPassword)))
	return
}

// encryption

// EncryptMetadata encrypts metadata.
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

// EncryptData encrypts file data.
func EncryptData(data []byte, key string) ([]byte, error) {
	nonce := []byte(GenerateRandomString(12))
	result, err := runAES256GCMEncryption([]byte(key), nonce, data)
	if err != nil {
		return nil, err
	}
	return append(nonce, result...), nil
}

// decryption

// AllKeysFailedError denotes that no key passed to [DecryptMetadataAllKeys] worked.
type AllKeysFailedError struct {
	Errors []error // errors thrown in the process
}

func (e *AllKeysFailedError) Error() string {
	return fmt.Sprintf("all keys failed: %v", e.Errors)
}

// DecryptMetadataAllKeys calls [DecryptMetadata] using all provided keys.
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

// DecryptMetadata decrypts metadata.
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

// DecryptData decrypts file data.
func DecryptData(data []byte, key string) ([]byte, error) {
	nonce, ciphertext := data[:12], data[12:]
	result, err := runAES256GCMDecryption([]byte(key), nonce, ciphertext)
	if err != nil {
		return nil, err
	}
	return result, nil
}
