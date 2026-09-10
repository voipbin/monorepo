package mcpserverhandler

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func randomKeyB64(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("could not generate random key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func Test_SecretCrypto_EncryptDecryptRoundTrip(t *testing.T) {
	key := randomKeyB64(t)
	c, err := NewSecretCrypto("1:" + key)
	if err != nil {
		t.Fatalf("NewSecretCrypto error = %v", err)
	}

	plaintext := "super-secret-token-value"
	ciphertext, nonce, version, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt error = %v", err)
	}
	if version != 1 {
		t.Errorf("expected version=1, got %d", version)
	}
	if len(ciphertext) == 0 {
		t.Errorf("expected non-empty ciphertext")
	}
	if len(nonce) == 0 {
		t.Errorf("expected non-empty nonce")
	}

	got, err := c.Decrypt(ciphertext, nonce, version)
	if err != nil {
		t.Fatalf("Decrypt error = %v", err)
	}
	if got != plaintext {
		t.Errorf("expected decrypted plaintext %q, got %q", plaintext, got)
	}
}

func Test_SecretCrypto_MultiKeyVersionDecrypt(t *testing.T) {
	keyV1 := randomKeyB64(t)
	keyV2 := randomKeyB64(t)

	// Encrypt under a config that only knows version 1.
	cV1, err := NewSecretCrypto("1:" + keyV1)
	if err != nil {
		t.Fatalf("NewSecretCrypto (v1 only) error = %v", err)
	}
	plaintext := "old-secret-encrypted-under-v1"
	ciphertext, nonce, version, err := cV1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt error = %v", err)
	}
	if version != 1 {
		t.Fatalf("expected version=1, got %d", version)
	}

	// Rotate: version 2 becomes current, version 1 remains configured for
	// decrypting old rows (design §6 rotation procedure).
	cRotated, err := NewSecretCrypto("1:" + keyV1 + ",2:" + keyV2)
	if err != nil {
		t.Fatalf("NewSecretCrypto (rotated) error = %v", err)
	}
	if cRotated.currentVersion != 2 {
		t.Fatalf("expected current version=2 after rotation, got %d", cRotated.currentVersion)
	}

	// The v1-encrypted row must still decrypt correctly under the rotated config.
	got, err := cRotated.Decrypt(ciphertext, nonce, version)
	if err != nil {
		t.Fatalf("Decrypt (v1 row under rotated config) error = %v", err)
	}
	if got != plaintext {
		t.Errorf("expected decrypted plaintext %q, got %q", plaintext, got)
	}

	// A NEW write under the rotated config uses version 2.
	newCiphertext, newNonce, newVersion, err := cRotated.Encrypt("new-secret-encrypted-under-v2")
	if err != nil {
		t.Fatalf("Encrypt (new write) error = %v", err)
	}
	if newVersion != 2 {
		t.Errorf("expected new writes to use version=2, got %d", newVersion)
	}
	gotNew, err := cRotated.Decrypt(newCiphertext, newNonce, newVersion)
	if err != nil {
		t.Fatalf("Decrypt (v2 row) error = %v", err)
	}
	if gotNew != "new-secret-encrypted-under-v2" {
		t.Errorf("unexpected decrypted value: %q", gotNew)
	}
}

func Test_NewSecretCrypto_MalformedConfigRejected(t *testing.T) {
	validKey := randomKeyB64(t)

	tests := []struct {
		name string
		raw  string
	}{
		{"missing colon", "1" + validKey},
		{"non-integer version", "abc:" + validKey},
		{"zero version", "0:" + validKey},
		{"negative version", "-1:" + validKey},
		{"not base64", "1:not-valid-base64!!!"},
		{"wrong key length (16 bytes)", "1:" + base64.StdEncoding.EncodeToString(make([]byte, 16))},
		{"wrong key length (33 bytes)", "1:" + base64.StdEncoding.EncodeToString(make([]byte, 33))},
		{"duplicate version", "1:" + validKey + ",1:" + validKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewSecretCrypto(tt.raw); err == nil {
				t.Errorf("expected an error for malformed config %q, got nil", tt.raw)
			}
		})
	}
}

func Test_NewSecretCrypto_EmptyConfigValid(t *testing.T) {
	c, err := NewSecretCrypto("")
	if err != nil {
		t.Fatalf("NewSecretCrypto(\"\") error = %v", err)
	}
	if c.currentVersion != 0 {
		t.Errorf("expected currentVersion=0 for empty config, got %d", c.currentVersion)
	}

	if _, _, _, err := c.Encrypt("anything"); err == nil {
		t.Errorf("expected Encrypt to fail with no configured key")
	}
}
