package utilhandler

import (
	"golang.org/x/crypto/bcrypt"
)

// HashGenerate returns the hashed string
func (h *utilHandler) HashGenerate(org string, cost int) (string, error) {
	return HashGenerate(org, cost)
}

// HashCheckPassword returns true if the given hashstring is correct
func (h *utilHandler) HashCheckPassword(password, hashString string) bool {
	return HashCheckPassword(password, hashString)
}

func HashGenerate(org string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(org), cost)
	return string(bytes), err
}

func HashCheckPassword(password, hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
		return false
	}

	return true
}
