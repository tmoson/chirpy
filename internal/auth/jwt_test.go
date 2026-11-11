package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWTCreation(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "signingSecret"
	expiresIn, err := time.ParseDuration("24h")
	if err != nil {
		t.Error("Unable to parse time for token creation.")
		return
	}
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("Unable to create token. Got error: %v", err)
		return
	}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) { return []byte(tokenSecret), nil },
	)
	if err != nil {
		t.Error("Unable to parse token for verification")
		return
	}
	tokenClaims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		t.Error("Unable to parse claims for verification")
		return
	}
	if tokenClaims.Issuer != "chirpy-access" {
		t.Errorf("Mismatch in issuer. Expected: chirpy-access, Got: %s", tokenClaims.Issuer)
		return
	}
	if tokenClaims.Subject != userID.String() {
		t.Errorf("Mismatch in token subject (userid). Expected: %s, Got: %s", userID.String(), tokenClaims.Subject)
		return
	}
	actualDuration := tokenClaims.ExpiresAt.Time.Sub(tokenClaims.IssuedAt.Time)
	if actualDuration != expiresIn {
		t.Errorf("Duration not parsed correctly. Expected %v, Got: %v", expiresIn, actualDuration)
		return
	}
}

func TestJWTCreateAndValidate(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "signingSecret"
	expiresIn, err := time.ParseDuration("24h")
	if err != nil {
		t.Error("Unable to parse time for token creation.")
		return
	}
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("Unable to create token. Got error : %v", err)
		return
	}
	validatedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Errorf("Error encountered while validating token: %v", err)
		return
	}
	if validatedID != userID {
		t.Errorf("Could not validate created token. Expected %v, Got: %v", userID, validatedID)
		return
	}
}
