package helphandler

import "golang.org/x/crypto/bcrypt"

// HashCheck returns true if the given hashstring is correct
func (h *helpHandler) HashCheck(password, hashString string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hashString), []byte(password)); err != nil {
		return false
	}

	return true
}

// HashGenerate generates hash from auth
func (h *helpHandler) HashGenerate(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
