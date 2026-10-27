package secretcrypto

import (
	"testing"
)

func Test_New_emptyEntries(t *testing.T) {
	tests := []struct {
		name string
		raw  string

		expectVersion int
	}{
		{"trailing comma", "1:" + testKey(1) + ",", 1},
		{"leading comma", "," + "2:" + testKey(2), 2},
		{"empty entry between keys", "1:" + testKey(1) + ", ,3:" + testKey(3), 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := New(tt.raw)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !res.HasKey() || res.currentVersion != tt.expectVersion {
				t.Errorf("Wrong match. expect current version: %d, got: %d", tt.expectVersion, res.currentVersion)
			}
		})
	}
}

// Test_invalidKeyLength builds the struct directly, since New rejects a key that is not 32 bytes.
func Test_invalidKeyLength(t *testing.T) {
	c := &SecretCrypto{keys: map[int][]byte{1: []byte("short")}, currentVersion: 1}

	t.Run("newGCM", func(t *testing.T) {
		if _, err := newGCM([]byte("short")); err == nil || err.Error() != "could not create the cipher" {
			t.Errorf("Wrong match. expect: could not create the cipher, got: %v", err)
		}
	})

	t.Run("Encrypt", func(t *testing.T) {
		ct, version, err := c.Encrypt("fake-secret-value", "telnyx")
		if err == nil || err.Error() != "could not create the cipher" {
			t.Errorf("Wrong match. expect: could not create the cipher, got: %v", err)
		}
		if ct != nil || version != 0 {
			t.Errorf("Wrong match. expect: no data, got: %v, %d", ct, version)
		}
	})

	t.Run("Decrypt", func(t *testing.T) {
		pt, err := c.Decrypt([]byte("any-data-longer-than-a-nonce"), 1, "telnyx")
		if err == nil || err.Error() != "could not create the cipher" {
			t.Errorf("Wrong match. expect: could not create the cipher, got: %v", err)
		}
		if pt != "" {
			t.Errorf("Wrong match. expect: no plaintext, got: %q", pt)
		}
	})
}
