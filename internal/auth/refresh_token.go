package auth

import (
	"crypto/rand"
	"encoding/hex"
)

func MakeRefreshToken() (string, error) {
    var seedData [32]byte
    _,err := rand.Read(seedData[:])
    if err != nil {
        return "",err
    }
    return hex.EncodeToString(seedData[:]), nil
}
