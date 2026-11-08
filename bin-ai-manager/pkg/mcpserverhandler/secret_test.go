package mcpserverhandler

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/secretcrypto"
)

func fakeKey(version byte) string {
	return fmt.Sprintf("%d:%s", version, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{version}, 32)))
}

func Test_NewSecretCrypto(t *testing.T) {
	tests := []struct {
		name string
		raw  string

		expectErr      bool
		expectHasKey   bool
		expectVersions []int
	}{
		{name: "empty", raw: ""},
		{name: "blank", raw: "   "},
		{name: "only blank entries", raw: " , ,"},
		{name: "single comma", raw: ","},
		{name: "one key", raw: fakeKey(1), expectHasKey: true, expectVersions: []int{1}},
		{name: "two keys with blanks", raw: fakeKey(2) + " ,, " + fakeKey(1) + ",", expectHasKey: true, expectVersions: []int{1, 2}},
		{name: "entry without a colon", raw: "SENTINELNOCOLON", expectErr: true},
		{name: "version is not a number", raw: "SENTINELVER:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)), expectErr: true},
		{name: "version zero", raw: "0:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)), expectErr: true},
		{name: "key is not base64", raw: "1:SENTINELB64", expectErr: true},
		{name: "key is too short", raw: "1:" + base64.StdEncoding.EncodeToString([]byte("SENTINELSHORT")), expectErr: true},
		{name: "version repeated", raw: fakeKey(1) + "," + fakeKey(1), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewSecretCrypto(tt.raw)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Fatalf("expected an error and a nil result, got %v / %v", res, err)
				}
				if !errors.Is(err, secretcrypto.ErrInvalidKeySet) {
					t.Errorf("the error must wrap ErrInvalidKeySet, got %v", err)
				}
				if strings.Contains(err.Error(), "SENTINEL") {
					t.Errorf("the error holds the key list text: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.HasKey() != tt.expectHasKey {
				t.Errorf("wrong HasKey. expect: %v, got: %v", tt.expectHasKey, res.HasKey())
			}
			if fmt.Sprint(res.Versions()) != fmt.Sprint(tt.expectVersions) && (len(tt.expectVersions) > 0 || len(res.Versions()) > 0) {
				t.Errorf("wrong versions. expect: %v, got: %v", tt.expectVersions, res.Versions())
			}
		})
	}
}

func Test_EncryptSecret(t *testing.T) {
	two, err := NewSecretCrypto(fakeKey(1) + "," + fakeKey(2))
	if err != nil {
		t.Fatal(err)
	}
	none, err := NewSecretCrypto("")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		crypto *secretcrypto.Crypto

		expectErr     error
		expectVersion int
	}{
		{name: "current version is the highest", crypto: two, expectVersion: 2},
		{name: "no key", crypto: none, expectErr: secretcrypto.ErrKeyNotConfigured},
		{name: "nil crypto", crypto: nil, expectErr: secretcrypto.ErrKeyNotConfigured},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, nonce, version, err := EncryptSecret(tt.crypto, "fake-secret")
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || ct != nil || nonce != nil || version != 0 {
					t.Errorf("expected %v and empty results, got %v / %v / %d / %v", tt.expectErr, ct, nonce, version, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if version != tt.expectVersion || len(nonce) != 12 || len(ct) != len("fake-secret")+16 {
				t.Errorf("wrong result. version: %d, nonce: %d bytes, ciphertext: %d bytes", version, len(nonce), len(ct))
			}
		})
	}
}

func Test_DecryptSecret(t *testing.T) {
	one, err := NewSecretCrypto(fakeKey(1))
	if err != nil {
		t.Fatal(err)
	}
	two, err := NewSecretCrypto(fakeKey(1) + "," + fakeKey(2))
	if err != nil {
		t.Fatal(err)
	}
	onlyTwo, err := NewSecretCrypto(fakeKey(2))
	if err != nil {
		t.Fatal(err)
	}
	none, err := NewSecretCrypto("")
	if err != nil {
		t.Fatal(err)
	}

	ct, nonce, version, err := EncryptSecret(one, "fake-secret")
	if err != nil {
		t.Fatal(err)
	}
	tampered := append([]byte{}, ct...)
	tampered[0] ^= 0xff

	tests := []struct {
		name    string
		crypto  *secretcrypto.Crypto
		ct      []byte
		nonce   []byte
		version int

		expectErr   error
		expectPlain string
	}{
		{name: "same key", crypto: one, ct: ct, nonce: nonce, version: version, expectPlain: "fake-secret"},
		{name: "row of an older version after a rotation", crypto: two, ct: ct, nonce: nonce, version: version, expectPlain: "fake-secret"},
		{name: "version not in the list", crypto: onlyTwo, ct: ct, nonce: nonce, version: version, expectErr: secretcrypto.ErrDecrypt},
		{name: "tampered ciphertext", crypto: one, ct: tampered, nonce: nonce, version: version, expectErr: secretcrypto.ErrDecrypt},
		{name: "ciphertext shorter than the tag", crypto: one, ct: ct[:8], nonce: nonce, version: version, expectErr: secretcrypto.ErrMalformed},
		{name: "nonce is not 12 bytes", crypto: one, ct: ct, nonce: nonce[:5], version: version, expectErr: secretcrypto.ErrMalformed},
		{name: "nonce column empty", crypto: one, ct: ct, nonce: nil, version: version, expectErr: secretcrypto.ErrMalformed},
		{name: "both columns empty", crypto: one, version: version, expectErr: secretcrypto.ErrDecrypt},
		{name: "no key", crypto: none, ct: ct, nonce: nonce, version: version, expectErr: secretcrypto.ErrKeyNotConfigured},
		{name: "nil crypto", crypto: nil, ct: ct, nonce: nonce, version: version, expectErr: secretcrypto.ErrKeyNotConfigured},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := DecryptSecret(tt.crypto, tt.ct, tt.nonce, tt.version)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != "" {
					t.Errorf("expected %v and an empty result, got %q / %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil || res != tt.expectPlain {
				t.Errorf("expected %q, got %q / %v", tt.expectPlain, res, err)
			}
		})
	}
}

// Test_rotation verifies that new writes use the highest version while an older row still reads.
func Test_rotation(t *testing.T) {
	old, err := NewSecretCrypto(fakeKey(1))
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewSecretCrypto(fakeKey(1) + "," + fakeKey(2))
	if err != nil {
		t.Fatal(err)
	}

	ct1, n1, v1, _ := EncryptSecret(old, "fake-secret")
	ct2, n2, v2, _ := EncryptSecret(rotated, "fake-secret")
	if v1 != 1 || v2 != 2 {
		t.Fatalf("wrong versions. got: %d, %d", v1, v2)
	}

	for _, row := range []struct {
		ct, nonce []byte
		version   int
	}{{ct1, n1, v1}, {ct2, n2, v2}} {
		if res, err := DecryptSecret(rotated, row.ct, row.nonce, row.version); err != nil || res != "fake-secret" {
			t.Errorf("version %d row must read after the rotation, got %q / %v", row.version, res, err)
		}
	}
}
