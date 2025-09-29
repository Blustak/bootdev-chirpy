package auth

import (
	"errors"

	"github.com/alexedwards/argon2id"
)

var Params = argon2id.Params{
	Memory:     12 * 1024,
	Iterations: 3,
	Parallelism: 1,
    SaltLength: 16,
    KeyLength: 32,
}

func HashPassword(password string) (string, error) {
    if len(password) < 1 {
        return "",errors.New("password cannot be empty.")
    }
	return argon2id.CreateHash(password, &Params)
}

func CheckPasswordHash(password, hash string) (bool,error) {
    if len(password) < 1 {
        return false,errors.New("password cannot be empty")
    }
    return argon2id.ComparePasswordAndHash(password,hash)
}
