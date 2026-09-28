package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "my-secure-password"

	hash, err := HashPassword(password)

	if err != nil {
		t.Fatalf("HashPassword() returned an error: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword() returned an empty hash")
	}

	if hash == password {
		t.Fatal("HashPassword() returned the plaintext password")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	password := "my-secure-password"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() returned an error: %v", err)
	}

	t.Run("correct password", func(t *testing.T) {
		match, err := CheckPasswordHash(password, hash)

		if err != nil {
			t.Fatalf("CheckPasswordHash() returned an error: %v", err)
		}

		if !match {
			t.Fatal("CheckPasswordHash() returned false for the correct password")
		}
	})

	t.Run("incorrect password", func(t *testing.T) {
		match, err := CheckPasswordHash("wrong-password", hash)

		if err != nil {
			t.Fatalf("CheckPasswordHash() returned an error: %v", err)
		}

		if match {
			t.Fatal("CheckPasswordHash() returned true for an incorrect password")
		}
	})
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, secret, expiresIn)

	if err != nil {
		t.Fatalf("MakeJWT() returned an error: %v", err)
	}

	if tokenString == "" {
		t.Fatal("MakeJWT() returned an empty token")
	}

	// Make sure the token we generated can actually be validated.
	gotUserID, err := ValidateJWT(tokenString, secret)

	if err != nil {
		t.Fatalf("ValidateJWT() returned an error: %v", err)
	}

	if gotUserID != userID {
		t.Fatalf(
			"ValidateJWT() returned user ID %v, want %v",
			gotUserID,
			userID,
		)
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "my-secret"

	t.Run("valid token", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT() returned an error: %v", err)
		}

		gotUserID, err := ValidateJWT(token, secret)

		if err != nil {
			t.Fatalf("ValidateJWT() returned an error: %v", err)
		}

		if gotUserID != userID {
			t.Fatalf(
				"ValidateJWT() returned user ID %v, want %v",
				gotUserID,
				userID,
			)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT() returned an error: %v", err)
		}

		_, err = ValidateJWT(token, "wrong-secret")

		if err == nil {
			t.Fatal("ValidateJWT() succeeded with the wrong secret")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := MakeJWT(userID, secret, -time.Hour)
		if err != nil {
			t.Fatalf("MakeJWT() returned an error: %v", err)
		}

		_, err = ValidateJWT(token, secret)

		if err == nil {
			t.Fatal("ValidateJWT() succeeded with an expired token")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := ValidateJWT("this-is-not-a-valid-jwt", secret)

		if err == nil {
			t.Fatal("ValidateJWT() succeeded with an invalid token")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := ValidateJWT("", secret)

		if err == nil {
			t.Fatal("ValidateJWT() succeeded with an empty token")
		}
	})
}