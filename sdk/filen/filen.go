package filen

import (
	"filen/filen-sdk-go/filen/client"
	"fmt"
)

type Filen struct {
	client *client.Client
}

func New() *Filen {
	return &Filen{
		client: &client.Client{},
	}
}

func (filen *Filen) Login(email, password string) {
	// fetch salt
	authInfo, err := filen.client.GetAuthInfo(email)
	if err != nil {
		panic(err)
	}

	fmt.Println(authInfo)

	/*// compute password as sha512 hash of second half of sha512-PBKDF2 of raw password
	password = hex.EncodeToString(pbkdf2.Key([]byte(password), []byte(authInfo.Salt), 200000, 512/8, sha512.New))
	password = password[len(password)/2:]
	derivedPasswordHash := sha512.New()
	derivedPasswordHash.Write([]byte(password))
	password = fmt.Sprintf("%032x", derivedPasswordHash.Sum(nil))

	// login and get keys
	keys, err := filen.Login(ctx, email, password)
	if err != nil {
		panic(err)
	}
	fmt.Printf("API Key: %s\n", keys.ApiKey)
	ctx = context.WithValue(ctx, "apiKey", keys.ApiKey)

	// fetch base folder uuid
	baseFolderUUID, err := filen.GetUserBaseFolder(ctx)
	fmt.Println("baseFolderUUID:", baseFolderUUID)*/
}
