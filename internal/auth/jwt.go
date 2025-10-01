package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)


func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
    if tokenSecret == "" {
        return "",  errors.New("tokenSecret cannot be an empty string")
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256,jwt.RegisteredClaims{
        Issuer: "chirpy",
        IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
        Subject: userID.String(),
    })
     return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString,tokenSecret string) (uuid.UUID,error) {
    token,err := jwt.ParseWithClaims(tokenString,&jwt.RegisteredClaims{},func(t *jwt.Token) (any, error) {
        return []byte(tokenSecret),nil
    })
    if err != nil {
        return uuid.UUID{},err
    }
    subject,err := token.Claims.GetSubject()
    if err != nil {
        return uuid.UUID{},err
    }
    return uuid.MustParse(subject),nil
}
