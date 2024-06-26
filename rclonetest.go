package main

import (
	"bufio"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"filen/rclone-test/api"
	"fmt"
	"golang.org/x/crypto/pbkdf2"
	"os"
	"strings"
)

var (
	ctx   = context.Background()
	filen = api.New()
)

func main() {
	// get credentials
	email := Input("Email: ", "filentest1@jupiterpi.de")
	password := Input("Password: ", "W74TTbTbJ2bE45M")
	fmt.Printf("Credentials: %s, %s\n", email, password)

	// fetch salt
	authInfo, err := filen.GetAuthInfo(ctx, email)
	if err != nil {
		panic(err)
	}

	// compute password as sha512 hash of second half of sha512-PBKDF2 of raw password
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

	fmt.Println(keys)
}

func Input(prompt, def string) string {
	fmt.Print(prompt)
	input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	input = input[:strings.Index(input, "\n")]
	if len(input) == 0 {
		input = def
	}
	return input
}
