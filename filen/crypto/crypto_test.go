package crypto

import (
	"encoding/hex"
	"testing"
)

func TestDeriveKeyFromPassword(t *testing.T) {
	key := hex.EncodeToString(DeriveKeyFromPassword("mypassword123", "somesalt", 200000, 512))
	if key != "725f1536062d83a4a396e071668080eb5ea8f1a2a69908231ef12305c9fccf7178876f8bae3adeecadd39ec5567c2d5cf53e122237f620f24163ad81ba63a9a6" {
		t.Fatalf("derived key did not match: %v", key)
	}
}

func TestGeneratePasswordAndMasterKey(t *testing.T) {
	rawPassword := "mypassword123"
	salt := "somesalt"
	derivedMasterKeys, derivedPassword := GeneratePasswordAndMasterKey(rawPassword, salt)
	if derivedMasterKeys != "725f1536062d83a4a396e071668080eb5ea8f1a2a69908231ef12305c9fccf71" {
		t.Fatalf("derivedMasterKeys did not match: %v", derivedMasterKeys)
	}
	if derivedPassword != "918f78aff194a340a461278270dc94a58f0e5d390f68fe31c3ab47b8930165bc32152d68097a45f49f147132d8cd8e3b21475b0a8d9cc3ed06052c117d9d6a24" {
		t.Fatalf("derivedPassword did not match: %v", derivedPassword)
	}
}

func TestUpdateKeys(t *testing.T) {
	/*apiKey := "32aaaab3eeed5cc69040de8c8f1e6dacc7d6ec299ed63f717cbe6116fc713dc5"
	masterKeys := []string{"e4fcd396fd2d214b391dadf3bf237ebcf0a5deae9dc65ceedc66337464debfe0"}*/
	//masterKeys := []string{"002QWzFMevnzvfD2lpp/1nFb/c6aYD+UFTxf8EA1F3ESF8D"}
	//publicKey := []string{}
	//TODO implement
}
