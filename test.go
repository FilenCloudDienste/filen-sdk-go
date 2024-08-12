package main

import (
	"bufio"
	"fmt"
	sdk "github.com/JupiterPi/filen-sdk-go/filen"
	"os"
	"strings"
)

func main() {
	// get credentials
	email := "filentest1@jupiterpi.de"
	password := "W74TTbTbJ2bE45M"
	fmt.Printf("Credentials: %s, %s\n", email, password)

	WriteSampleFile()

	filen, err := sdk.New(email, password)
	if err != nil {
		panic(err)
	}
	baseFolderUUID, err := filen.GetBaseFolderUUID()
	if err != nil {
		panic(err)
	}

	/*files, directories, err := filen.ReadDirectory(baseFolderUUID)
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
	}*/

	/*idx := slices.IndexFunc(files, func(file *sdk.File) bool { return file.Name == "lsample.txt" })
	if idx == -1 {
		panic(errors.New("file not found"))
	}
	file := files[idx]
	_, err = os.Create("downloaded/" + file.Name)
	if err != nil {
		panic(err)
	}*/

	/*start := time.Now()
	err = filen.DownloadFile(file, destination)
	if err != nil {
		panic(err)
	}
	duration := time.Since(start)
	fmt.Printf("Took %vs\n", duration.Seconds())*/
	//fmt.Printf("File: \n\n%s\n\n", content)

	/*err = os.WriteFile("downloaded/"+file.Name, content, 0666)
	if err != nil {
		panic(err)
	}*/

	uploadFile, err := os.Open("downloaded/large_sample-1mb.txt")
	if err != nil {
		panic(err)
	}
	err = filen.UploadFile(uploadFile, baseFolderUUID)
	if err != nil {
		panic(err)
	}
}

func WriteSampleFile() {
	data := make([]byte, 0)
	for i := 0; i < 200_000; i++ {
		data = append(data, []byte(fmt.Sprintf("%v\n", i))...)
	}
	err := os.WriteFile("downloaded/large_sample-1mb.txt", data, 0666)
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
