package utilhandler

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt cost limits (from bcrypt package constants)
const (
	bcryptMinCost = 4  // bcrypt.MinCost
	bcryptMaxCost = 31 // bcrypt.MaxCost
)

// HashGenerate returns the hashed string
func (h *utilHandler) HashGenerate(org string, cost int) (string, error) {
	return HashGenerate(org, cost)
}

// HashCheckPassword returns true if the given hashstring is correct
func (h *utilHandler) HashCheckPassword(password, hashString string) bool {
	return HashCheckPassword(password, hashString)
}

// HashGenerate generates a bcrypt hash of the given string.
// Cost must be between 4 and 31 inclusive.
func HashGenerate(org string, cost int) (string, error) {
	if cost < bcryptMinCost || cost > bcryptMaxCost {
		return "", fmt.Errorf("bcrypt cost must be between %d and %d, got %d", bcryptMinCost, bcryptMaxCost, cost)
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(org), cost)
	return string(bytes), err
}

func HashCheckPassword(password, hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
		return false
	}

	return true
}
