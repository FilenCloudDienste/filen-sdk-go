package crypto

import (
	"encoding/base64"
	"encoding/json"
	"filen/filen-sdk-go/filen/client"
	"fmt"
	"strings"
)

type MetadataEncryptedString string

type FolderMetadata struct {
	Name string `json:"name"`
}

func DeriveKeyFromPassword(password string, salt string, iterations int, bitLength int) []byte {
	// see filen-sdk-ts /crypto/util.ts deriveKeyFromPassword()
	return runPBKDF2(password, salt, iterations, bitLength)
}

func GeneratePasswordAndMasterKey(rawPassword string, salt string) (derivedMasterKey string, derivedPassword string) {
	// see filen-sdk-ts /crypto/util.ts generatePasswordAndMasterKeyBasedOnAuthVersion()
	// compute password (for login) as sha512 hash of second half of sha512-PBKDF2 of raw password

	derivedKey := encodeHex(DeriveKeyFromPassword(rawPassword, salt, 200000, 512))
	derivedMasterKey = derivedKey[:len(derivedKey)/2]
	derivedPassword = derivedKey[len(derivedKey)/2:]

	derivedPassword = fmt.Sprintf("%032x", runSHA521(derivedPassword))

	return
}

func UpdateKeys(client *client.Client, apiKey string, masterKeys []string) (newMasterKeys []string, publicKey string, privateKey string, err error) {
	// see filen-sdk-ts /index.ts FilenSDK._updateKeys()

	masterKey := masterKeys[len(masterKeys)-1]

	encryptedMasterKeys := EncryptMetadata(strings.Join(masterKeys, "|"), masterKey, true)

	userMasterKeys, err := client.GetUserMasterKeys(encryptedMasterKeys)
	if err != nil {
		panic(err)
	}

	decryptedMasterKeys, err := DecryptMetadata(userMasterKeys.Keys, masterKey)

	newMasterKeys = strings.Split(decryptedMasterKeys, "|")
	publicKey, privateKey, err = UpdateKeyPair(client, apiKey, newMasterKeys)
	if err != nil {
		return nil, "", "", err
	}
	return newMasterKeys, publicKey, privateKey, nil
}

func UpdateKeyPair(client *client.Client, apiKey string, masterKeys []string) (publicKey string, privateKey string, err error) {
	// see filen-sdk-ts /index.ts FilenSDK._updateKeyPair

	keyPairInfo, err := client.GetUserKeyPairInfo()
	if err != nil {
		return "", "", err
	}
	publicKey = keyPairInfo.PublicKey

	for _, masterKey := range masterKeys {
		privateKey, _ := DecryptMetadata(keyPairInfo.PrivateKey, masterKey)
		if privateKey != "" {
			break
		}
	}

	encryptedPrivateKey := EncryptMetadata(privateKey, masterKeys[len(masterKeys)-1], true)
	err = client.UpdateUserKeyPair(publicKey, encryptedPrivateKey)
	if err != nil {
		return "", "", err
	}

	return
}

func DecryptFolderMetadata(masterKeys []string, encryptedName string) (*FolderMetadata, error) {
	// see filen-sdk-ts /crypto/decrypt.ts Decrypt.metadata()

	fmt.Printf("%#v\n", masterKeys)
	fmt.Printf("%#v\n", encryptedName)

	key := DeriveKeyFromPassword(masterKeys[0], masterKeys[0], 1, 256)
	iv := encryptedName[3:15]
	encrypted, err := decodeBase64(encryptedName[15:])
	if err != nil {
		return nil, err
	}
	authTag := encrypted[:len(encrypted)-16]
	cipherText := encrypted[len(encrypted)-16:]
	fmt.Printf("key: %#v\n", base64.StdEncoding.EncodeToString(key))
	fmt.Printf("cipherText: %#v\n", base64.StdEncoding.EncodeToString(cipherText))
	fmt.Printf("iv: %#v\n", iv)
	fmt.Printf("authTag: %#v\n", base64.StdEncoding.EncodeToString(authTag))
	result, err := runAES256GCM(key, []byte(iv), cipherText, authTag)
	if err != nil {
		return nil, err
	}
	folderMetadata := &FolderMetadata{}
	err = json.Unmarshal(result, folderMetadata)
	if err != nil {
		return nil, err
	}
	return folderMetadata, nil
}

func EncryptMetadata(metadata string, key string, derive bool) string {
	// see filen-sdk-ts /crypto/encrypt.ts Encrypt.metadata
	iv := randString(12)
	derivedKey := []byte(key)
	if derive {
		derivedKey = DeriveKeyFromPassword(key, key, 1, 256)
	}

	result, err := runAES256GCM(derivedKey, []byte(iv), []byte(metadata), []byte{})
	if err != nil {
		panic(err)
	}

	return "002" + encodeBase64(result) //TODO append auth tag
}

func DecryptMetadata(metadata string, key string) (string, error) {
	derivedKey := DeriveKeyFromPassword(key, key, 1, 256)
	iv := metadata[3:15]
	encrypted, err := decodeBase64(metadata[15:])
	authTag := encrypted[:len(encrypted)-16]
	cipherText := encrypted[len(encrypted)-16:]
	result, err := runAES256GCM(derivedKey, []byte(iv), cipherText, authTag)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
