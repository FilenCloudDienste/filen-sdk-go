package crypto

func EncryptMetadata(metadata string, key string) (EncryptedString, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)
	data := []byte(metadata)
	result, err := runAES256GCMEncryption(derivedKey, data)
	if err != nil {
		return "", err
	}
	return EncryptedString(result), nil
}
