package crypto

func EncryptMetadataSymmetrically(metadata string, key string) (SymmetricallyEncryptedString, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)
	data := []byte(metadata)
	result, err := runAES256GCMEncrypt(derivedKey, data)
	if err != nil {
		return "", err
	}
	return SymmetricallyEncryptedString(result), nil
}
