package main

import (
	"bufio"
	"errors"
	sdk "filen/filen-sdk-go/filen"
	"fmt"
	"os"
	"slices"
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
	files, directories, err := filen.ReadDirectory(baseFolderUUID)
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

	idx := slices.IndexFunc(files, func(file *sdk.File) bool { return file.Name == "large_lipsum.txt" })
	if idx == -1 {
		panic(errors.New("file not found"))
	}
	file := files[idx]
	content, err := filen.ReadFile(file)
	if err != nil {
		panic(err)
	}
	fmt.Printf("File: \n\n%s\n\n", content)

	err = os.WriteFile("downloaded/"+file.Name, content, 0666)
	if err != nil {
		panic(err)
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
