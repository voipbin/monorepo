package mcpserverhandler

import (
	"strings"
	"testing"
)

func Test_NewSecretCrypto_skipsEmptyEntries(t *testing.T) {
	c, err := NewSecretCrypto(testCoverageKeys + ", ,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.currentVersion != 1 || len(c.keys) != 1 {
		t.Errorf("currentVersion = %d, keys = %d; expect 1 and 1", c.currentVersion, len(c.keys))
	}
}

func Test_SecretCrypto_Encrypt_errors(t *testing.T) {
	tests := []struct {
		name   string
		crypto *SecretCrypto

		expectMsg string
	}{
		{name: "no key configured", crypto: &SecretCrypto{keys: map[int][]byte{}}, expectMsg: "no MCP secret encryption key is configured"},
		{name: "current version missing from key map", crypto: &SecretCrypto{keys: map[int][]byte{1: make([]byte, 32)}, currentVersion: 2}, expectMsg: "version 2 not found"},
		{name: "key has invalid length", crypto: &SecretCrypto{keys: map[int][]byte{1: make([]byte, 5)}, currentVersion: 1}, expectMsg: "could not create AES cipher"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, nonce, version, err := tt.crypto.Encrypt("plain")
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Fatalf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
			if ct != nil || nonce != nil || version != 0 {
				t.Errorf("expected zero results, got %v %v %d", ct, nonce, version)
			}
		})
	}
}

func Test_SecretCrypto_Decrypt_errors(t *testing.T) {
	good, err := NewSecretCrypto(testCoverageKeys)
	if err != nil {
		t.Fatal(err)
	}
	ct, nonce, version, err := good.Encrypt("plain")
	if err != nil {
		t.Fatal(err)
	}

	tampered := append([]byte(nil), ct...)
	tampered[0] ^= 0xff

	tests := []struct {
		name   string
		crypto *SecretCrypto

		ciphertext []byte
		nonce      []byte
		version    int

		expectMsg string
	}{
		{name: "unknown version", crypto: good, ciphertext: ct, nonce: nonce, version: 9, expectMsg: "version 9 is not configured"},
		{name: "invalid key length", crypto: &SecretCrypto{keys: map[int][]byte{1: make([]byte, 5)}, currentVersion: 1}, ciphertext: ct, nonce: nonce, version: 1, expectMsg: "could not create AES cipher"},
		{name: "tampered ciphertext", crypto: good, ciphertext: tampered, nonce: nonce, version: version, expectMsg: "could not decrypt secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.crypto.Decrypt(tt.ciphertext, tt.nonce, tt.version)
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Fatalf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
			if res != "" {
				t.Errorf("expected empty plaintext, got %q", res)
			}
		})
	}
}

func Test_SecretCrypto_roundTripValue(t *testing.T) {
	c, err := NewSecretCrypto(testCoverageKeys)
	if err != nil {
		t.Fatal(err)
	}
	ct, nonce, version, err := c.Encrypt("fake-token-value")
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Errorf("version = %d, expect: 1", version)
	}
	if strings.Contains(string(ct), "fake-token-value") {
		t.Errorf("ciphertext contains plaintext")
	}
	got, err := c.Decrypt(ct, nonce, version)
	if err != nil || got != "fake-token-value" {
		t.Errorf("Decrypt = %q, %v", got, err)
	}
}
