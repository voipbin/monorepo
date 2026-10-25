package secretcrypto

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-number-manager/models/provider"
)

func testKey(b byte) string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = b
	}
	return base64.StdEncoding.EncodeToString(k)
}

func Test_New(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("short"))

	tests := []struct {
		name string
		raw  string

		expectErr     bool
		expectVersion int
	}{
		{"empty", "", false, 0},
		{"spaces", "   ", false, 0},
		{"single", "1:" + testKey(1), false, 1},
		{"highest wins regardless of order", "2:" + testKey(2) + ",1:" + testKey(1), false, 2},
		{"no colon", "abc", true, 0},
		{"version not integer", "x:" + testKey(1), true, 0},
		{"version zero", "0:" + testKey(1), true, 0},
		{"version negative", "-1:" + testKey(1), true, 0},
		{"bad base64", "1:!!!", true, 0},
		{"not 32 bytes", "1:" + short, true, 0},
		{"duplicate version", "1:" + testKey(1) + ",1:" + testKey(2), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := New(tt.raw)
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected err: %v, got: %v", tt.expectErr, err)
			}
			if err != nil {
				// the error must not echo the key material or the raw entry
				if strings.Contains(err.Error(), testKey(1)) || strings.Contains(err.Error(), short) {
					t.Errorf("error leaks key material: %v", err)
				}
				return
			}
			if res.currentVersion != tt.expectVersion {
				t.Errorf("version: expected %d, got %d", tt.expectVersion, res.currentVersion)
			}
		})
	}
}

func Test_EncryptDecrypt(t *testing.T) {
	c, err := New("1:" + testKey(1))
	if err != nil {
		t.Fatal(err)
	}

	data, version, err := c.Encrypt("fake-secret-value", "telnyx")
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Errorf("version: expected 1, got %d", version)
	}
	if strings.Contains(string(data), "fake-secret-value") {
		t.Errorf("ciphertext contains plaintext")
	}

	got, err := c.Decrypt(data, version, "telnyx")
	if err != nil || got != "fake-secret-value" {
		t.Fatalf("decrypt: got %q, err %v", got, err)
	}

	// the same plaintext encrypts to a different ciphertext (random nonce)
	data2, _, _ := c.Encrypt("fake-secret-value", "telnyx")
	if string(data) == string(data2) {
		t.Errorf("two encryptions are identical")
	}

	// AAD mismatch (row swap) fails
	if _, err := c.Decrypt(data, version, "twilio"); err == nil {
		t.Errorf("expected failure for aad mismatch")
	}

	// tampered data fails
	bad := append([]byte{}, data...)
	bad[len(bad)-1] ^= 0xff
	if _, err := c.Decrypt(bad, version, "telnyx"); err == nil {
		t.Errorf("expected failure for tampered data")
	}

	// too short
	if _, err := c.Decrypt([]byte{1, 2}, version, "telnyx"); err == nil {
		t.Errorf("expected failure for short data")
	}
}

func Test_KeyRotation(t *testing.T) {
	old, _ := New("1:" + testKey(1))
	data, version, err := old.Encrypt("fake", "telnyx")
	if err != nil {
		t.Fatal(err)
	}

	both, _ := New("1:" + testKey(1) + ",2:" + testKey(2))
	// the old row decrypts with its own version; new writes use the latest
	if got, err := both.Decrypt(data, version, "telnyx"); err != nil || got != "fake" {
		t.Fatalf("old row must decrypt: %q %v", got, err)
	}
	if _, v, _ := both.Encrypt("x", "telnyx"); v != 2 {
		t.Errorf("new writes must use the latest version, got %d", v)
	}

	// removing the old version makes the old row undecryptable
	onlyNew, _ := New("2:" + testKey(2))
	if _, err := onlyNew.Decrypt(data, version, "telnyx"); err == nil {
		t.Errorf("expected failure when the key version was removed")
	}
}

func Test_EncryptErrors(t *testing.T) {
	none, _ := New("")
	if _, _, err := none.Encrypt("x", "telnyx"); !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
		t.Errorf("expected ErrSecretKeyNotConfigured, got %v", err)
	}
	if _, err := none.Decrypt([]byte("abcdefghijklmnopqrstuvwxyz"), 1, "telnyx"); err == nil {
		t.Errorf("expected decrypt failure without keys")
	}

	c, _ := New("1:" + testKey(1))
	if _, _, err := c.Encrypt(strings.Repeat("a", MaxPlaintextBytes+1), "telnyx"); !errors.Is(err, provider.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument for a long secret, got %v", err)
	}
	if _, _, err := c.Encrypt(strings.Repeat("a", MaxPlaintextBytes), "telnyx"); err != nil {
		t.Errorf("the maximum length must be accepted: %v", err)
	}

	var nilCrypto *SecretCrypto
	if nilCrypto.HasKey() {
		t.Errorf("nil crypto must have no key")
	}
	if _, err := nilCrypto.Decrypt(nil, 1, "x"); err == nil {
		t.Errorf("nil crypto must fail to decrypt")
	}
}
