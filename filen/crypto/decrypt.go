package crypto

import "encoding/base64"

type SymmetricallyEncryptedString string

func DecryptMetadataSymmetrically(metadata SymmetricallyEncryptedString, key string) (string, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)

	iv := metadata[3:15]
	encrypted, err := base64.StdEncoding.DecodeString(string(metadata[15:]))
	if err != nil {
		return "", err
	}

	result, err := runAES256GCMDecrypt(derivedKey, []byte(iv), encrypted, nil)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func DecryptMetadataAsymmetrically(metadata string, key string) (string, error) {
	//TODO
	return "", nil
}

func DecryptData(data []byte, key string) ([]byte, error) {
	//TODO
	return nil, nil
}
