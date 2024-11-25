package utilhandler

import (
	"crypto/rand"
	"encoding/base64"
)

// StringGenerateRandom generates a random string of a fixed size.
func (h *utilHandler) StringGenerateRandom(size int) (string, error) {
	return StringGenerateRandom(size)
}

// StringGenerateRandom generates a random string of a fixed size.
func StringGenerateRandom(size int) (string, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	randomString := base64.RawURLEncoding.EncodeToString(bytes)
	if len(randomString) > size {
		randomString = randomString[:size]
	}

	return randomString, nil
}
