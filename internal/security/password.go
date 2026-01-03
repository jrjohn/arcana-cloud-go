package security

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default bcrypt cost
	DefaultCost = 12
)

// PasswordHasher handles password hashing and verification
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new PasswordHasher instance
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{cost: DefaultCost}
}

// Hash generates a bcrypt hash of the password
func (h *PasswordHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify checks if the password matches the hash
func (h *PasswordHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
