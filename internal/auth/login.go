package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)


func GetBearerToken(headers http.Header) (string, error) {
    return getAuthorizationToken("Bearer")(headers)
}

func GetApiKeyToken(headers http.Header) (string,error) {
    return getAuthorizationToken("ApiKey")(headers)
}

func getAuthorizationToken(key string) func(headers http.Header) (string, error) {
	return func(headers http.Header) (string, error) {
		if headers == nil {
			return "", errors.New("Empty header")
		}
		authHeader := headers.Get("Authorization")
		if authHeader == "" {
			return "", errors.New("Couldn't find authorization header")
		}
		authHeader = strings.Trim(authHeader, " ")
		splitString := strings.Split(authHeader, " ")
		if len(splitString) != 2 {
			return "", fmt.Errorf("Malformed header: [%s]", strings.Join(splitString, ","))
		}
        if splitString[0] != key {
            return "", errors.New("Key - Token mismatch")
        }
		return splitString[1], nil

	}
}
