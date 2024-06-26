package main

import (
	"bufio"
	sdk "filen/filen-sdk-go/filen"
	"fmt"
	"os"
	"strings"
)

var (
	filen = sdk.New()
)

func main() {
	// get credentials
	//email := Input("Email: ", "filentest1@jupiterpi.de")
	email := "filentest1@jupiterpi.de"
	//password := Input("Password: ", "W74TTbTbJ2bE45M")
	password := "W74TTbTbJ2bE45M"
	fmt.Printf("Credentials: %s, %s\n", email, password)

	filen.Login(email, password)
	filen.Readdir()
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
