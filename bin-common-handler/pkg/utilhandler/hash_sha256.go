package utilhandler

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashSHA256Hex returns the lowercase hex-encoded SHA-256 hash of the input string.
func (h *utilHandler) HashSHA256Hex(input string) string {
	return HashSHA256Hex(input)
}

// HashSHA256Hex returns the lowercase hex-encoded SHA-256 hash of the input string.
func HashSHA256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
