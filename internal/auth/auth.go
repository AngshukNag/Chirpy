package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	BearerToken = "Bearer "
	ApiKey      = "ApiKey "
)

func MakeRefreshToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)

	if err != nil {
		return "", err
	}

	return hash, nil
}

func CheckPasswordHash(password string, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)

	if err != nil {
		return false, err
	}

	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			// A usual scenario is to set the expiration time relative to the current time
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "chirpy-access",
			Subject:   userID.String(),
		})

	signedToken, err := token.SignedString([]byte(tokenSecret))

	return signedToken, err
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.UUID{}, err
	} else if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok == true {
		userIDString, err_sub := claims.GetSubject()
		if err_sub != nil {
			return uuid.UUID{}, err_sub
		}

		userID, err_uuid_parse := uuid.Parse(userIDString)
		if err_uuid_parse != nil {
			return uuid.UUID{}, errors.New("Invalid UUID")
		}
		return userID, nil
	} else {
		return uuid.UUID{}, errors.New("invalid JWT")
	}
}

func GetToken(headers http.Header, authType string) (string, error) {
	authHeader := headers.Get("Authorization")
	if len(authHeader) < max(len(BearerToken), len(ApiKey)) {
		return "", errors.New("Invalid auth type")
	}
	var tokenKey string
	var tokenValue string
	switch authType {
	case BearerToken, ApiKey:
		tokenKey, tokenValue = authHeader[:len(authType)], authHeader[len(authType):]
		if tokenKey != authType {
			return "", errors.New("Invalid auth type in API request")
		}
	default:
		return "", errors.New("Invalid auth type")
	}

	return tokenValue, nil
}
