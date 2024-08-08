package crypto

import (
	"encoding/hex"
	"fmt"
)

func DeriveKeyFromPassword(password string, salt string, iterations int, bitLength int) []byte {
	return runPBKDF2(password, salt, iterations, bitLength)
}

func GeneratePasswordAndMasterKey(rawPassword string, salt string) (derivedMasterKey string, derivedPassword string) {
	derivedKey := hex.EncodeToString(DeriveKeyFromPassword(rawPassword, salt, 200000, 512))
	derivedMasterKey, derivedPassword = derivedKey[:len(derivedKey)/2], derivedKey[len(derivedKey)/2:]
	derivedPassword = fmt.Sprintf("%032x", runSHA521(derivedPassword))
	return
}
