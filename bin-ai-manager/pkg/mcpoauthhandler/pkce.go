package mcpoauthhandler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// generateRandomToken returns a crypto-random, URL-safe token of n raw
// bytes, base64url-encoded without padding -- the same RNG shape already
// used for mcpserverhandler's secret nonces (design §7a Layer 1).
func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generatePKCE returns a PKCE code_verifier (43-128 chars, design §7b) and
// its S256 code_challenge. S256 exclusively -- never "plain" -- per design
// §7b: both GitHub and Linear accept S256, so no vendor branch is needed.
func generatePKCE() (verifier string, challenge string, err error) {
	// 32 raw bytes -> 43 base64url chars, comfortably within the
	// 43-128 char range the PKCE spec requires.
	verifier, err = generateRandomToken(32)
	if err != nil {
		return "", "", err
	}

	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])

	return verifier, challenge, nil
}
