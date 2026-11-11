package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func CheckPasswordHash(attempt, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(attempt, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		&jwt.RegisteredClaims{
			Issuer:    "chirpy-access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			Subject:   userID.String(),
		})
	jwt.SigningMethodES256.Hash.String()
	signedString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return signedString, err
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(tkn *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.UUID{}, err
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok {
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.UUID{}, err
		}
		return userID, nil
	}
	return uuid.UUID{}, errors.New("Unable to parse token claims")
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Authorization header missing")
	}
	bearerStringSplit := strings.Split(authHeader, " ")
	if len(bearerStringSplit) != 2 {
		return "", errors.New("Malformatted header")
	}
	return bearerStringSplit[1], nil
}
