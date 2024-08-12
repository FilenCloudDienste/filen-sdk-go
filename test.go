package main

import (
	"bufio"
	sdk "filen/filen-sdk-go/filen"
	"fmt"
	"os"
	"strings"
)

func main() {
	// get credentials
	//email := Input("Email: ", "filentest1@jupiterpi.de")
	email := "filentest2@jupiterpi.de"
	//password := Input("Password: ", "W74TTbTbJ2bE45M")
	password := "X1QFNYBkf9"
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
	_, _, err = filen.ReadDirectory(baseFolderUUID)
	if err != nil {
		panic(err)
	}

	/*fmt.Println("Files:")
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

	err = filen.UploadFile("downloaded/uploadfile.txt", baseFolderUUID)
	if err != nil {
		panic(err)
	}
}

func WriteSampleFile() {
	data := make([]byte, 0)
	for i := 0; i < 1_000_000; i++ {
		data = append(data, []byte(fmt.Sprintf("%v\n", i))...)
	}
	err := os.WriteFile("downloaded/large_sample.txt", data, 0666)
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
