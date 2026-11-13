package filen_sdk_go

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	sdk "github.com/FilenCloudDienste/filen-sdk-go/filen"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/client"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/crypto"
	"github.com/FilenCloudDienste/filen-sdk-go/filen/types"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

var filen *sdk.Filen
var baseTestDir *types.Directory

func setupEnv() error {

	err := godotenv.Load()
	if err != nil {
		// we don't panic in case the environment variables are set somewhere else
		println("Warning: Error loading .env file: ", err.Error())
	}

	email := os.Getenv("TEST_EMAIL")
	password := os.Getenv("TEST_PASSWORD")
	apiKey := os.Getenv("TEST_API_KEY")
	if email == "" || password == "" {
		return fmt.Errorf("TEST_EMAIL and TEST_PASSWORD environment variables must be set")
	}
	if apiKey == "" {
		filen, err = sdk.New(context.Background(), email, password, "XXXXXX")
	} else {
		filen, err = sdk.NewWithAPIKey(context.Background(), email, password, apiKey)
	}
	if err != nil {
		return err
	}
	baseTestDir, err = filen.CreateDirectory(context.Background(), filen.BaseFolder, "go")
	if err != nil {
		return err
	}
	testPath := filepath.Join(".", "test_files")
	err = os.MkdirAll(testPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating test_files directory: %w", err)
	}
	if err = writeTestFiles(); err != nil {
		return err
	}
	downloadedPath := filepath.Join(".", "downloaded")
	err = os.MkdirAll(downloadedPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating downloaded directory: %w", err)
	}
	return nil
}

func cleanupEnv() error {
	err := filen.TrashDirectory(context.Background(), baseTestDir)
	if err != nil {
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	// prep client
	err := setupEnv()
	if err != nil {
		panic(err)
	}

	// run tests
	code := m.Run()
	err = cleanupEnv()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func getCompatTestFile(parent types.DirectoryInterface) (*types.IncompleteFile, io.Reader, error) {
	creationTime := time.Date(2025, time.January, 11, 12, 13, 14, 15*1000*1000, time.UTC).Local()
	modificationTime := time.Date(2025, time.January, 11, 12, 13, 14, 16*1000*1000, time.UTC).Local()
	incompleteFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "large_sample-20mb.txt", "", creationTime, modificationTime, parent)
	if err != nil {
		return nil, nil, err
	}
	setKey, err := crypto.MakeEncryptionKeyFromUnknownStr("0123456789abcdefghijklmnopqrstuv")
	if err != nil {
		return nil, nil, err
	}
	incompleteFile.EncryptionKey = *setKey
	specificFile, err := os.Open("test_files/large_sample-20mb.txt")
	if err != nil {
		return nil, nil, err
	}
	return incompleteFile, specificFile, nil
}

// TestUploadsToGoDir uploads test files and directories to the "go" directory
// this is so the TS sdk can validate these files on its side and check if they are compatible
// there should ideally be a TestDownloadsFromTSDir that validates files from the TS sdk
func TestUploadsToGoDir(t *testing.T) {
	goDir, err := filen.FindDirectory(context.Background(), "compat-go")
	if err != nil {
		t.Fatal(err)
	}
	if goDir != nil {
		err = filen.TrashDirectory(context.Background(), goDir)
		if err != nil {
			t.Fatal(err)
		}
	}

	goDir, err = filen.CreateDirectory(context.Background(), filen.BaseFolder, "compat-go")
	if err != nil {
		t.Fatal(err)
	}

	testDir, err := filen.FindDirectory(context.Background(), "compat-go/dir")
	if err != nil {
		t.Fatal(err)
	}
	if testDir != nil {
		err = filen.TrashDirectory(context.Background(), testDir)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = filen.CreateDirectory(context.Background(), goDir, "dir")
	if err != nil {
		t.Fatal(err)
	}

	testEmptyFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "empty.txt", "", time.Now(), time.Now(), goDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = filen.UploadFile(context.Background(), testEmptyFile, bytes.NewReader(make([]byte, 0)))
	if err != nil {
		t.Fatal(err)
	}

	shareEmail := os.Getenv("TEST_SHARE_EMAIL")

	if shareEmail != "" {
		if obj, ok := goDir.(types.NonRootFileSystemObject); ok {
			err = filen.ShareItemToUser(context.Background(), obj, shareEmail)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			t.Fatal("object is not a NonRootFileSystemObject")
		}
	}

	testSmallFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "small.txt", "", time.Now(), time.Now(), goDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = filen.UploadFile(context.Background(), testSmallFile, bytes.NewReader([]byte("Hello World from Go!")))
	if err != nil {
		t.Fatal(err)
	}

	bigRandomBytes := make([]byte, 1024*1024*4)
	_, _ = rand.Read(bigRandomBytes)
	testBigFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "big.txt", "", time.Now(), time.Now(), goDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = filen.UploadFile(context.Background(), testBigFile, bytes.NewReader([]byte(hex.EncodeToString(bigRandomBytes))))
	if err != nil {
		t.Fatal(err)
	}

	testCompatFile, compatFileReader, err := getCompatTestFile(goDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = filen.UploadFile(context.Background(), testCompatFile, compatFileReader)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDownloadsFromTSDir(t *testing.T) {
	tsDir, err := filen.FindDirectory(context.Background(), "compat-ts")
	if err != nil {
		t.Fatal(err)
	}
	if tsDir == nil {
		fmt.Printf("WARNING: could not find compat-ts directory, skipping compatibility checks\n")
		return
	}

	dir, err := filen.FindDirectory(context.Background(), "compat-ts/dir")
	if err != nil {
		t.Fatal(err)
	}
	if dir == nil {
		t.Fatal("expected directory 'compat-ts/dir' to exist")
	}

	empty, err := filen.FindFile(context.Background(), "compat-ts/empty.txt")
	if err != nil {
		t.Fatal(err)
	}
	emptyBytes, err := io.ReadAll(filen.GetDownloadReader(context.Background(), empty))
	if err != nil {
		t.Fatal(err)
	}
	if len(emptyBytes) != 0 {
		t.Fatal("expected empty file to be empty")
	}

	small, err := filen.FindFile(context.Background(), "compat-ts/small.txt")
	if err != nil {
		t.Fatal(err)
	}
	smallBytes, err := io.ReadAll(filen.GetDownloadReader(context.Background(), small))
	if err != nil {
		t.Fatal(err)
	}
	if string(smallBytes) != "Hello World from TypeScript!" {
		t.Fatalf("small file did not match expected contents: %s", string(smallBytes))
	}

	if filen.AuthVersion != 1 {
		big, err := filen.FindFile(context.Background(), "compat-ts/big.txt")
		if err != nil {
			t.Fatal(err)
		}
		bigBytes, err := io.ReadAll(filen.GetDownloadReader(context.Background(), big))
		if err != nil {
			t.Fatal(err)
		}
		if len(bigBytes) != 1024*1024*4*2 {
			t.Fatalf("expected big file to be 8MB, was instead %d bytes", len(bigBytes))
		}

		goSideCompatFile, goSideReader, err := getCompatTestFile(tsDir)
		if err != nil {
			t.Fatal(err)
		}
		tsSideCompatFile, err := filen.FindFile(context.Background(), path.Join("compat-ts", goSideCompatFile.Name))
		if err != nil {
			t.Fatal(err)
		}
		if tsSideCompatFile == nil {
			fmt.Printf("WARNING: could not find compatibility file %s, skipping compatibility check\n", goSideCompatFile.Name)
			return
		}
		tsSideCompatFileBytes, err := io.ReadAll(filen.GetDownloadReader(context.Background(), tsSideCompatFile))
		if err != nil {
			t.Fatal(err)
		}
		goSideCompatFileBytes, err := io.ReadAll(goSideReader)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(tsSideCompatFileBytes, goSideCompatFileBytes) {
			t.Fatal("compatibility file contents did not match")
		}
		goSideCompatFile.UUID = tsSideCompatFile.UUID
		goSideCompatFile.EncryptionKey.Cipher = tsSideCompatFile.EncryptionKey.Cipher
		if !reflect.DeepEqual(*goSideCompatFile, tsSideCompatFile.IncompleteFile) {
			t.Fatalf("compatibility file objects did not match; go side:\n%#v\nTS side:\n%#v", goSideCompatFile, tsSideCompatFile.IncompleteFile)
		}
	}
}

func TestReadDirectories(t *testing.T) {
	expectedDirs := map[string]*types.Directory{}
	expectedFiles := map[string]*types.File{}
	expectedExtraDirs := map[string]*types.Directory{}

	t.Run("setup", func(t *testing.T) {
		var err error
		def, err := filen.CreateDirectory(context.Background(), baseTestDir, "def")
		if err != nil {
			t.Fatal(err)
		}
		expectedDirs["def"] = def
		extra, err := filen.CreateDirectory(context.Background(), def, "extra")
		if err != nil {
			t.Fatal(err)
		}
		expectedExtraDirs["extra"] = extra
		uploads, err := filen.CreateDirectory(context.Background(), baseTestDir, "uploads")
		if err != nil {
			t.Fatal(err)
		}
		expectedDirs["uploads"] = uploads
		incompleteFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "large_sample-1mb.txt", "", time.Now(), time.Now(), baseTestDir)
		if err != nil {
			t.Fatal(err)
		}
		largeSample, err := filen.UploadFile(context.Background(), incompleteFile, bytes.NewReader([]byte("Sample!")))
		if err != nil {
			t.Fatal(err)
		}
		expectedFiles["large_sample-1mb.txt"] = largeSample
		incompleteFile, err = types.NewIncompleteFile(filen.FileEncryptionVersion, "abc.txt", "", time.Now(), time.Now(), baseTestDir)
		if err != nil {
			t.Fatal(err)
		}
		abc, err := filen.UploadFile(context.Background(), incompleteFile, bytes.NewReader([]byte("ABC!")))
		if err != nil {
			t.Fatal(err)
		}
		expectedFiles["abc.txt"] = abc
	})

	t.Run("Check", func(t *testing.T) {
		requiredDirs := map[string]*types.Directory{}
		requiredFiles := map[string]*types.File{}

		for k, v := range expectedDirs {
			requiredDirs[k] = v
		}
		for k, v := range expectedFiles {
			requiredFiles[k] = v
		}
		files, dirs, err := filen.ReadDirectory(context.Background(), baseTestDir)
		if err != nil {
			t.Fatal(err)
		}

		for _, dir := range dirs {
			if requiredDir, ok := requiredDirs[dir.Name]; ok {
				if !reflect.DeepEqual(dir, requiredDir) {
					t.Fatalf("Directory %s does not match found:\n%#v\nExpected:\n%#v\n", dir.Name, dir, requiredDir)
				}
				delete(requiredDirs, dir.Name)
			}
		}

		if len(requiredDirs) > 0 {
			for k, v := range requiredDirs {
				fmt.Printf("%s: %#v\n", k, v)
			}
			t.Fatalf("Missing directories")
		}

		for _, file := range files {
			if requiredFile, ok := requiredFiles[file.Name]; ok {
				if !reflect.DeepEqual(file, requiredFile) {
					t.Fatalf("File %s does not match found:\n%#v\nExpected:\n%#v\n", file.Name, file, requiredFile)
				}
				delete(requiredFiles, file.Name)
			}
		}
		if len(requiredFiles) > 0 {
			t.Fatalf("Missing files: %v\n", requiredFiles)
		}
	})

	t.Run("CheckRecursive", func(t *testing.T) {
		requiredDirs := map[string]*types.Directory{}
		requiredFiles := map[string]*types.File{}

		for k, v := range expectedDirs {
			requiredDirs[k] = v
		}
		for k, v := range expectedFiles {
			requiredFiles[k] = v
		}
		for k, v := range expectedExtraDirs {
			requiredDirs[k] = v
		}
		files, dirs, err := filen.ListRecursive(context.Background(), baseTestDir)
		if err != nil {
			t.Fatal(err)
		}

		for _, dir := range dirs {
			if requiredDir, ok := requiredDirs[dir.Name]; ok {
				if !reflect.DeepEqual(dir, requiredDir) {
					t.Fatalf("Directory %s does not match found:\n%#v\nExpected:\n%#v\n", dir.Name, dir, requiredDir)
				}
				delete(requiredDirs, dir.Name)
			}
		}

		if len(requiredDirs) > 0 {
			for k, v := range requiredDirs {
				fmt.Printf("%s: %#v\n", k, v)
			}
			t.Fatalf("Missing directories")
		}

		for _, file := range files {
			if requiredFile, ok := requiredFiles[file.Name]; ok {
				if !reflect.DeepEqual(file, requiredFile) {
					t.Fatalf("File %s does not match found:\n%#v\nExpected:\n%#v\n", file.Name, file, requiredFile)
				}
				delete(requiredFiles, file.Name)
			}
		}
		if len(requiredFiles) > 0 {
			t.Fatalf("Missing files: %v\n", requiredFiles)
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		for _, dir := range expectedDirs {
			err := filen.TrashDirectory(context.Background(), dir)
			if err != nil {
				t.Fatal(err)
			}
		}

		for _, file := range expectedFiles {
			err := filen.TrashFile(context.Background(), *file)
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestSerialization(t *testing.T) {
	buffer := bytes.NewBuffer(make([]byte, 0, 1024*1024))
	err := filen.SerializeTo(buffer)
	if err != nil {
		t.Fatal(err)
	}
	deserialized, err := sdk.DeserializeFrom(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if !filenEqual(filen, deserialized) {
		t.Fatalf("Filen objects are not equal:\nOriginal:%#v\nDeserialized:%#v\n", filen, deserialized)
	}
	t.Run("TSConfig", func(t *testing.T) {
		masterKeys := make([]string, max(len(filen.MasterKeys), 1))
		switch filen.AuthVersion {
		case 1, 2:
			for i, masterKey := range filen.MasterKeys {
				masterKeys[i] = string(masterKey.Bytes[:])
			}
		case 3:
			masterKeys[0] = hex.EncodeToString(filen.DEK.Bytes[:])
		default:
			panic("Unknown auth version")
		}

		privateKeyRaw, err := x509.MarshalPKCS8PrivateKey(&filen.PrivateKey)
		if err != nil {
			t.Fatal(err)
		}
		privateKeyEncoded := base64.StdEncoding.EncodeToString(privateKeyRaw)

		publicKeyRaw := x509.MarshalPKCS1PublicKey(&filen.PublicKey)
		publicKeyEncoded := base64.StdEncoding.EncodeToString(publicKeyRaw)

		tsConfig := &sdk.TSConfig{
			Email:          filen.Email,
			MasterKeys:     masterKeys,
			APIKey:         filen.Client.APIKey,
			PrivateKey:     privateKeyEncoded,
			PublicKey:      publicKeyEncoded,
			AuthVersion:    int(filen.AuthVersion),
			BaseFolderUUID: filen.BaseFolder.GetUUID(),
		}

		tsDeserialized, err := sdk.NewFromTSConfig(*tsConfig)
		if err != nil {
			t.Fatal(err)
		}

		if !filenEqual(filen, tsDeserialized) {
			t.Fatalf("Filen objects are not equal:\nOriginal:%#v\nDeserialized:%#v\n", filen, tsDeserialized)
		}
	})
}

func TestDirectoryActions(t *testing.T) {
	newPath := "go/abc/def/ghi"
	var directory *types.Directory
	t.Run("GetBaseFolder", func(t *testing.T) {
		dirOrRoot, err := filen.FindDirectory(context.Background(), "")
		if err != nil {
			t.Fatal(err)
		}
		if dir, ok := dirOrRoot.(*types.RootDirectory); ok {
			if dir.GetUUID() != filen.BaseFolder.GetUUID() {
				t.Fatalf("root directory did not match")
			}
		} else {
			t.Fatal("dirOrRoot is not a root directory")
		}
	})
	t.Run("Create FindDirectoryOrCreate", func(t *testing.T) {
		dirOrRoot, err := filen.FindDirectoryOrCreate(context.Background(), newPath)
		if err != nil {
			t.Fatal(err)
		}
		if dir, ok := dirOrRoot.(*types.Directory); ok {
			directory = dir
		} else {
			t.Fatal("dirOrRoot is not a normal directory")
		}
	})

	t.Run("Find FindDirectoryOrCreate", func(t *testing.T) {
		dirOrRoot, err := filen.FindDirectoryOrCreate(context.Background(), newPath)
		if err != nil {
			t.Fatal(err)
		}
		if dir, ok := dirOrRoot.(*types.Directory); ok {
			if !reflect.DeepEqual(dir, directory) {
				t.Fatalf("directories are not equal:\nCreated:%#v\nFound:%#v\n", directory, dir)
			}
		} else {
			t.Fatal("dirOrRoot is not a normal directory")
		}
	})
	t.Run("Rename", func(t *testing.T) {
		tempPath := "go/abc/g/d"
		tempDir, err := filen.FindDirectoryOrCreate(context.Background(), tempPath)
		if err != nil {
			t.Fatal(err)
		}
		dir, ok := tempDir.(*types.Directory)
		if !ok {
			t.Fatal("dir was not a basic dir")
		}
		err = filen.Rename(context.Background(), dir, "e")
		if err != nil {
			t.Fatal(err)
		}
		foundDir, err := filen.FindDirectoryOrCreate(context.Background(), "go/abc/g/e")
		if err != nil {
			t.Fatal(err)
		}
		if foundDir == nil {
			t.Fatal("failed to rename")
		}
	})
	t.Run("Trash", func(t *testing.T) {
		err := filen.TrashDirectory(context.Background(), directory)
		if err != nil {
			t.Fatal(err)
		}

		dir, err := filen.FindDirectory(context.Background(), newPath)
		if err != nil {
			t.Fatal("failed to gracefully handle missing directory: ", err)
		}
		if dir != nil {
			t.Fatal("Directory not trashed")
		}
	})
	t.Run("Cleanup", func(t *testing.T) {
		dir, err := filen.FindDirectory(context.Background(), "go/abc")
		if err != nil {
			t.Fatal(err)
		}
		err = filen.TrashDirectory(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestCaseInsensitivePaths(t *testing.T) {
	var (
		directory *types.Directory
		file      *types.File
	)
	if !t.Run("Create", func(t *testing.T) {
		dirOrRoot, err := filen.FindDirectoryOrCreate(context.Background(), "go/CaseDir")
		if err != nil {
			t.Fatal(err)
		}
		dir, ok := dirOrRoot.(*types.Directory)
		if !ok {
			t.Fatal("dirOrRoot is not a normal directory")
		}
		directory = dir
	}) {
		return
	}
	defer func() {
		err := filen.TrashDirectory(context.Background(), directory)
		if err != nil {
			t.Fatal(err)
		}
	}()

	t.Run("FindDirectoryOrCreate", func(t *testing.T) {
		dirOrRoot, err := filen.FindDirectoryOrCreate(context.Background(), "go/casedir")
		if err != nil {
			t.Fatal(err)
		}
		if dirOrRoot.GetUUID() != directory.GetUUID() {
			t.Fatalf("expected directory %s, got %s", directory.GetUUID(), dirOrRoot.GetUUID())
		}
	})

	t.Run("FindDirectory", func(t *testing.T) {
		dirOrRoot, err := filen.FindDirectory(context.Background(), "/GO/CASEDIR/")
		if err != nil {
			t.Fatal(err)
		}
		if dirOrRoot == nil {
			t.Fatal("directory not found")
		}
		if dirOrRoot.GetUUID() != directory.GetUUID() {
			t.Fatalf("expected directory %s, got %s", directory.GetUUID(), dirOrRoot.GetUUID())
		}
	})

	if !t.Run("Upload", func(t *testing.T) {
		incompleteFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "Case.txt", "", time.Now(), time.Now(), directory)
		if err != nil {
			t.Fatal(err)
		}
		file, err = filen.UploadFile(context.Background(), incompleteFile, bytes.NewReader([]byte("case")))
		if err != nil {
			t.Fatal(err)
		}
	}) {
		return
	}

	t.Run("FindFile", func(t *testing.T) {
		foundFile, err := filen.FindFile(context.Background(), "go/casedir/case.TXT")
		if err != nil {
			t.Fatal(err)
		}
		if foundFile == nil {
			t.Fatal("file not found")
		}
		if foundFile.GetUUID() != file.GetUUID() {
			t.Fatalf("expected file %s, got %s", file.GetUUID(), foundFile.GetUUID())
		}
	})

	t.Run("FindItemThroughFile", func(t *testing.T) {
		for _, p := range []string{"go/CaseDir/Case.txt/x", "go/casedir/CASE.TXT/x"} {
			item, err := filen.FindItem(context.Background(), p)
			if err != nil {
				t.Fatal(err)
			}
			if item != nil {
				t.Fatalf("expected no item at %s, got %#v", p, item)
			}
		}
	})

	t.Run("MoveIntoOwnParent", func(t *testing.T) {
		err := filen.MoveItem(context.Background(), file, directory.GetUUID(), true)
		if err != nil {
			t.Fatal(err)
		}
		foundFile, err := filen.FindFile(context.Background(), "go/CaseDir/Case.txt")
		if err != nil {
			t.Fatal(err)
		}
		if foundFile == nil {
			t.Fatal("file was trashed by moving it into its own parent")
		}
	})

	t.Run("MoveDirectoryIntoOwnParent", func(t *testing.T) {
		err := filen.MoveItem(context.Background(), directory, baseTestDir.GetUUID(), true)
		if err != nil {
			t.Fatal(err)
		}
		foundDir, err := filen.FindDirectory(context.Background(), "go/CaseDir")
		if err != nil {
			t.Fatal(err)
		}
		if foundDir == nil {
			t.Fatal("directory was trashed by moving it into its own parent")
		}
	})
}

func TestMoveDirectoryOverwrite(t *testing.T) {
	parentOrRoot, err := filen.FindDirectoryOrCreate(context.Background(), "go/overwrite")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := filen.TrashDirectory(context.Background(), parentOrRoot)
		if err != nil {
			t.Fatal(err)
		}
	}()
	existing, err := filen.FindDirectoryOrCreate(context.Background(), "go/overwrite/dir")
	if err != nil {
		t.Fatal(err)
	}
	movedOrRoot, err := filen.FindDirectoryOrCreate(context.Background(), "go/overwrite/src/dir")
	if err != nil {
		t.Fatal(err)
	}
	moved, ok := movedOrRoot.(*types.Directory)
	if !ok {
		t.Fatal("movedOrRoot is not a normal directory")
	}

	err = filen.MoveItem(context.Background(), moved, parentOrRoot.GetUUID(), true)
	if err != nil {
		t.Fatal(err)
	}
	foundDir, err := filen.FindDirectory(context.Background(), "go/overwrite/dir")
	if err != nil {
		t.Fatal(err)
	}
	if foundDir == nil {
		t.Fatal("moved directory not found")
	}
	if foundDir.GetUUID() != moved.GetUUID() {
		t.Fatalf("expected moved directory %s, got %s (existing was %s)", moved.GetUUID(), foundDir.GetUUID(), existing.GetUUID())
	}
}

func TestEmptyFileActions(t *testing.T) {
	osFile, err := os.Open("test_files/empty.txt")
	if err != nil {
		t.Fatal(err)
	}
	incompleteFile, err := types.NewIncompleteFileFromOSFile(filen.FileEncryptionVersion, osFile, baseTestDir)
	if err != nil {
		t.Fatal(err)
	}
	var file *types.File

	if !t.Run("Upload", func(t *testing.T) {
		file, err = filen.UploadFile(context.Background(), incompleteFile, osFile)
		if err != nil {
			t.Fatal(err)
		}
	}) {
		return
	}

	t.Run("Find", func(t *testing.T) {
		foundObj, err := filen.FindItem(context.Background(), "go/empty.txt")

		if err != nil {
			t.Fatal(err)
		}
		if foundObj == nil {
			t.Fatal("File not found")
		}
		foundFile, ok := foundObj.(*types.File)
		if !ok {
			t.Fatal("File not found")
		}
		if foundFile.Size != 0 {
			t.Fatalf("File size is not zero: %v", foundFile.Size)
		}
	})

	t.Run("Download", func(t *testing.T) {
		err = filen.DownloadToPath(context.Background(), file, "downloaded/empty.txt")
		if err != nil {
			t.Fatal(err)
		}
		downloadedFile, err := os.Open("downloaded/empty.txt")
		if err != nil {
			t.Fatal(err)
		}
		fileData, err := io.ReadAll(downloadedFile)
		if err != nil {
			t.Fatal(err)
		}
		if len(fileData) != 0 {
			t.Fatalf("File size is not zero: %v", len(fileData))
		}
	})
	t.Run("Trash", func(t *testing.T) {
		err = filen.TrashFile(context.Background(), *file)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestNewWithAPIKey(t *testing.T) {
	filen2, err := sdk.NewWithAPIKey(context.Background(), os.Getenv("TEST_EMAIL"), os.Getenv("TEST_PASSWORD"), filen.Client.APIKey)
	if err != nil {
		t.Fatal(err)
	}
	if !filenEqual(filen, filen2) {
		t.Fatalf("Filen objects are not equal:\nOriginal:%#v\nDeserialized:%#v\n", filen, filen2)
	}
}

func TestFileActions(t *testing.T) {
	fileName := "large_sample-20mb.txt"
	osFile, err := os.Open("test_files/" + fileName)

	incompleteFile, err := types.NewIncompleteFileFromOSFile(filen.FileEncryptionVersion, osFile, baseTestDir)
	if err != nil {
		t.Fatal(err)
	}

	var (
		file *types.File
	)

	if !t.Run("Upload", func(t *testing.T) {
		file, err = filen.UploadFile(context.Background(), incompleteFile, osFile)
		if err != nil {
			t.Fatal(err)
		}
	}) {
		return
	}

	t.Run("ChangeMeta", func(t *testing.T) {
		file.Created = file.Created.Add(time.Second)
		file.LastModified = file.LastModified.Add(time.Second)

		err = filen.UpdateMeta(context.Background(), file)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Find", func(t *testing.T) {
		foundFile, err := filen.FindFile(context.Background(), path.Join("go", fileName))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(file, foundFile) {
			t.Fatalf("Uploaded \n%#v\n and Downloaded \n%#v\n file info did not match", file, foundFile)
		}
	})

	t.Run("Download", func(t *testing.T) {
		downloadPath := "downloaded/" + fileName
		err := filen.DownloadToPath(context.Background(), file, downloadPath)
		if err != nil {
			t.Fatal(err)
		}
		downloadedFile, err := os.Open(downloadPath)
		if err != nil {
			t.Fatal(err)
		}
		eq, err := assertFilesEqual(osFile, downloadedFile)
		if err != nil {
			t.Fatal(err)
		}
		if !eq {
			t.Fatalf("Uploaded \n%#v\n and downloaded file contents did not match", file)
		}
	})

	t.Run("Trash", func(t *testing.T) {
		err = filen.TrashFile(context.Background(), *file)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestPartialRead(t *testing.T) {
	fileName := "partial_read.txt"

	incompleteFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, fileName, "", time.Now(), time.Now(), baseTestDir)
	if err != nil {
		t.Fatal(err)
	}

	myString := "Sample!"
	contents := make([]byte, sdk.ChunkSize+len(myString))
	copy(contents[:], myString)
	copy(contents[sdk.ChunkSize:], myString)

	file, err := filen.UploadFile(context.Background(), incompleteFile, bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}

	reader := filen.GetDownloadReaderWithOffset(context.Background(), file, 0, int64(len(myString)))

	readBytes := make([]byte, sdk.ChunkSize*2)
	n, err := reader.Read(readBytes)
	if err != io.EOF && err != nil {
		t.Fatal(err)
	}

	if string(readBytes[:n]) != myString {
		t.Fatalf("Expected %s, got %s", myString, string(readBytes[:n]))
	}

	reader = filen.GetDownloadReaderWithOffset(context.Background(), file, sdk.ChunkSize, sdk.ChunkSize*2)
	readBytes = make([]byte, sdk.ChunkSize*2)
	n, err = reader.Read(readBytes)
	if err != io.EOF && err != nil {
		t.Fatal(err)
	}

	if string(readBytes[:n]) != myString {
		t.Fatalf("Expected %s, got %s", myString, string(readBytes[:n]))
	}

	err = filen.TrashFile(context.Background(), *file)
	if err != nil {
		t.Fatal(err)
	}
}

func requireShared(ctx context.Context, item types.NonRootFileSystemObject) error {
	if shared, err := filen.IsItemShared(ctx, item); err != nil {
		return err
	} else if !shared {
		return fmt.Errorf("item %s not shared", item.GetName())
	} else {
		return nil
	}
}

func requireLinked(ctx context.Context, item types.NonRootFileSystemObject) error {
	if linked, err := filen.IsItemLinked(ctx, item); err != nil {
		return err
	} else if !linked {
		return fmt.Errorf("item %s not linked", item.GetName())
	} else {
		return nil
	}
}

func TestShareAndLink(t *testing.T) {
	// set up
	// /go/share
	// /go/share/file1.txt
	// /go/share/dir1
	// /go/share/dir1/file2.txt
	// /go/share/dir1/dir2
	// /go/share/dir1/dir2/file3.txt
	files := make([]types.File, 0, 3)
	var addedFile *types.File
	var dir2 *types.Directory
	var file3 *types.File
	dirs := make([]types.Directory, 0, 2)
	var shareDir *types.Directory
	shareUser := os.Getenv("TEST_SHARE_EMAIL")
	if shareUser == "" {
		t.Skip("TEST_SHARE_EMAIL not set")
	}
	t.Run("Setup", func(t *testing.T) {
		maybeShareDir, err := filen.FindDirectory(context.Background(), path.Join(baseTestDir.Name, "share"))
		if err != nil {
			t.Fatal(err)
		}
		if maybeShareDir != nil {
			err = filen.TrashDirectory(context.Background(), maybeShareDir)
			if err != nil {
				t.Fatal(err)
			}
		}
		shareDir, err = filen.CreateDirectory(context.Background(), baseTestDir, "share")
		if err != nil {
			t.Fatal(err)
		}

		iFile1, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "file1.txt", "", time.Now(), time.Now(), shareDir)
		if err != nil {
			t.Fatal(err)
		}
		file1, err := filen.UploadFile(context.Background(), iFile1, bytes.NewReader([]byte("Sample!")))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, *file1)

		dir1, err := filen.CreateDirectory(context.Background(), shareDir, "dir1")
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, *dir1)

		iFile2, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "file2.txt", "", time.Now(), time.Now(), dir1)
		if err != nil {
			t.Fatal(err)
		}
		file2, err := filen.UploadFile(context.Background(), iFile2, bytes.NewReader(nil))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, *file2)

		dir2, err = filen.CreateDirectory(context.Background(), dir1, "dir2")
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, *dir2)

		iFile3, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "file3.txt", "", time.Now(), time.Now(), dir2)
		if err != nil {
			t.Fatal(err)
		}
		file3, err = filen.UploadFile(context.Background(), iFile3, bytes.NewReader([]byte("Sample!")))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, *file3)
	})
	t.Run("Share", func(t *testing.T) {
		err := filen.ShareItemToUser(context.Background(), shareDir, shareUser)
		if err != nil {
			t.Fatal(err)
		}

		g, ctx := errgroup.WithContext(context.Background())
		for _, file := range files {
			g.Go(func() error {
				return requireShared(ctx, file)
			})
		}
		for _, dir := range dirs {
			g.Go(func() error {
				return requireShared(ctx, dir)
			})
		}
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Link", func(t *testing.T) {
		_, err := filen.PublicLinkItem(context.Background(), shareDir)
		if err != nil {
			t.Fatal(err)
		}

		g, ctx := errgroup.WithContext(context.Background())
		for _, file := range files {
			g.Go(func() error {
				return requireLinked(ctx, file)
			})
		}
		for _, dir := range dirs {
			g.Go(func() error {
				return requireLinked(ctx, dir)
			})
		}
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Add", func(t *testing.T) {
		addedDir, err := filen.CreateDirectory(context.Background(), shareDir, "added")
		if err != nil {
			t.Fatal(err)
		}

		addedIFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "added.txt", "", time.Now(), time.Now(), addedDir)
		if err != nil {
			t.Fatal(err)
		}
		addedFile, err = filen.UploadFile(context.Background(), addedIFile, bytes.NewReader([]byte("Sample!")))
		if err != nil {
			t.Fatal(err)
		}

		g, ctx := errgroup.WithContext(context.Background())
		g.Go(func() error {
			return requireLinked(ctx, addedDir)
		})
		g.Go(func() error {
			return requireLinked(ctx, addedFile)
		})
		g.Go(func() error {
			return requireShared(ctx, addedDir)
		})
		g.Go(func() error {
			return requireShared(ctx, addedFile)
		})
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Move", func(t *testing.T) {
		moveTarget, err := filen.CreateDirectory(context.Background(), shareDir, "move")
		if err != nil {
			t.Fatal(err)
		}

		err = filen.MoveItem(context.Background(), addedFile, moveTarget.GetUUID(), true)
		if err != nil {
			t.Fatal(err)
		}
		if addedFile.GetParent() != moveTarget.GetUUID() {
			t.Fatal("file parent uuid not locally updated")
		}

		foundAddedFile, err := filen.FindItem(
			context.Background(),
			path.Join(baseTestDir.GetName(), shareDir.GetName(), moveTarget.GetName(), addedFile.GetName()),
		)
		if err != nil {
			t.Fatal(err)
		}
		if foundAddedFile == nil {
			t.Fatal("moved file not found")
		}
		if foundAddedFile.GetUUID() != addedFile.GetUUID() {
			t.Fatal("file uuid not locally updated")
		}

		err = filen.MoveItem(context.Background(), dir2, moveTarget.GetUUID(), true)
		if err != nil {
			t.Fatal(err)
		}
		if dir2.GetParent() != moveTarget.GetUUID() {
			t.Fatal("dir parent uuid not locally updated")
		}

		foundDir2, err := filen.FindDirectory(
			context.Background(),
			path.Join(baseTestDir.GetName(), shareDir.GetName(), moveTarget.GetName(), dir2.GetName()),
		)
		if err != nil {
			t.Fatal(err)
		}
		if foundDir2 == nil {
			t.Fatal("moved dir not found")
		}
		if foundDir2.GetUUID() != dir2.GetUUID() {
			t.Fatal("dir uuid not locally updated")
		}

		movedFiles, _, err := filen.ReadDirectory(context.Background(), foundDir2)
		if err != nil {
			t.Fatal(err)
		}
		if len(movedFiles) != 1 {
			t.Fatal("moved file not found")
		}
		if movedFiles[0].GetUUID() != file3.GetUUID() {
			t.Fatal("moved file uuid not locally updated")
		}
	})
}

// TestChunkedUploadIntoLinkedDir uploads a file chunk by chunk with
// NewFileUpload, UploadChunk and CompleteFileUpload, the way a caller that
// sends chunks in parallel does (rclone's multi-thread upload, for example).
// The file must join the parent directory's public link, just like a file
// sent with UploadFile.
func TestChunkedUploadIntoLinkedDir(t *testing.T) {
	ctx := context.Background()
	leftover, err := filen.FindDirectory(ctx, path.Join(baseTestDir.Name, "chunkedLinked"))
	if err != nil {
		t.Fatal(err)
	}
	if leftover != nil {
		if err = filen.TrashDirectory(ctx, leftover); err != nil {
			t.Fatal(err)
		}
	}
	linkedDir, err := filen.CreateDirectory(ctx, baseTestDir, "chunkedLinked")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = filen.PublicLinkItem(ctx, linkedDir); err != nil {
		t.Fatal(err)
	}

	iFile, err := types.NewIncompleteFile(filen.FileEncryptionVersion, "chunked.txt", "", time.Now(), time.Now(), linkedDir)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("Sample!")
	fu := filen.NewFileUpload(iFile)
	fu.Hasher.Write(data)
	resp, err := filen.UploadChunk(ctx, fu, 0, data)
	if err != nil {
		t.Fatal(err)
	}
	bucketAndRegion := make(chan client.V3UploadResponse, 1)
	bucketAndRegion <- *resp
	close(bucketAndRegion)
	file, err := filen.CompleteFileUpload(ctx, fu, bucketAndRegion, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	if err = requireLinked(ctx, file); err != nil {
		t.Fatal(err)
	}
}

func writeTestData(writer io.Writer, length int) error {
	data := make([]byte, 0)
	for i := 0; i < length; i++ {
		data = append(data, []byte(fmt.Sprintf("%v\n", i))...)
	}
	_, err := writer.Write(data)
	return err
}

func writeTestFiles() error {
	_, err := os.Create("test_files/empty.txt")
	if err != nil {
		return fmt.Errorf("failed to create empty.txt: %w", err)
	}
	smallFile, err := os.Create("test_files/large_sample-1mb.txt")
	if err != nil {
		return fmt.Errorf("failed to create large_sample-1mb: %w", err)
	}
	defer func() { _ = smallFile.Close() }()
	if err = writeTestData(smallFile, 100_000); err != nil {
		return err
	}
	normalFile, err := os.Create("test_files/large_sample-3mb.txt")
	if err != nil {
		return fmt.Errorf("failed to create large_sample-3mb: %w", err)
	}
	if err = writeTestData(normalFile, 350_000); err != nil {
		return err
	}
	bigFile, err := os.Create("test_files/large_sample-20mb.txt")
	if err != nil {
		return fmt.Errorf("failed to create large_sample-20mb: %w", err)
	}
	if err = writeTestData(bigFile, 2_700_000); err != nil {
		return err
	}
	return nil
}

func assertFilesEqual(f1 *os.File, f2 *os.File) (bool, error) {
	const chunkSize = 1024
	b1 := make([]byte, chunkSize)
	b2 := make([]byte, chunkSize)
	i := 0
	_, err1 := f1.Seek(0, io.SeekStart)
	_, err2 := f2.Seek(0, io.SeekStart)

	if err1 != nil || err2 != nil {
		return false, fmt.Errorf("seek error: %v, %v", err1, err2)
	}
	for {
		i++
		_, err1 = f1.Read(b1)
		_, err2 = f2.Read(b2)

		if err1 != nil || err2 != nil {
			if err1 == io.EOF && err2 == io.EOF {
				return true, nil
			} else if err1 == io.EOF || err2 == io.EOF {
				return false, nil
			} else {
				return false, fmt.Errorf("read error: %v, %v", err1, err2)
			}
		}

		if !bytes.Equal(b1, b2) {
			fmt.Printf("Chunk %d did not match\n", i)
			fmt.Printf("b1: %v\nb2: %v\n", b1, b2)
			return false, nil
		}
	}
}

func filenEqual(f1 *sdk.Filen, f2 *sdk.Filen) bool {
	return reflect.DeepEqual(f1.AuthVersion, f2.AuthVersion) &&
		reflect.DeepEqual(f1.Client.APIKey, f2.Client.APIKey) &&
		reflect.DeepEqual(f1.Email, f2.Email) &&
		reflect.DeepEqual(f1.MasterKeys, f2.MasterKeys) &&
		reflect.DeepEqual(f1.DEK, f2.DEK) &&
		reflect.DeepEqual(f1.PrivateKey, f2.PrivateKey) &&
		reflect.DeepEqual(f1.PublicKey, f2.PublicKey) &&
		reflect.DeepEqual(f1.HMACKey, f2.HMACKey) &&
		reflect.DeepEqual(f1.BaseFolder, f2.BaseFolder)
}
