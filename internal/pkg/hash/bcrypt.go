package hash

import (
	"go-standard/internal/apperror"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plaintext password using bcrypt with the default cost.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", apperror.Internal("hash: failed to hash password", err)
	}
	return string(b), nil
}

// CheckPassword compares a bcrypt hash against a plaintext password.
// Returns nil on match, apperror.Unauthorized on mismatch or error.
func CheckPassword(hashed, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)); err != nil {
		return apperror.Unauthorized("invalid credentials")
	}
	return nil
}
