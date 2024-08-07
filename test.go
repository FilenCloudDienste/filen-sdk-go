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

	err := filen.Login(email, password)
	if err != nil {
		panic(err)
	}
	baseFolderUUID, err := filen.GetBaseFolderUUID()
	if err != nil {
		panic(err)
	}
	files, directories, err := filen.Readdir(baseFolderUUID)
	if err != nil {
		panic(err)
	}

	fmt.Println("Files:")
	for _, file := range files {
		fmt.Printf("%#v\n", file)
	}
	fmt.Println("Directories:")
	for _, directory := range directories {
		fmt.Printf("%#v\n", directory)
	}
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
