package secretcrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeKeyList builds a key list entry from a byte sequence so that no key literal is stored.
func fakeKeyList(version int, start byte) string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = start + byte(i)
	}

	return fmt.Sprintf("%d:%s", version, base64.StdEncoding.EncodeToString(k))
}

func mustNew(t *testing.T, raw string, opts ...Option) *Crypto {
	t.Helper()

	c, err := New(raw, opts...)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return c
}

func Test_New(t *testing.T) {
	good := fakeKeyList(1, 0x01)
	marker := "DISTINCTIVE-MARKER-TEXT"

	tests := []struct {
		name string
		raw  string

		expectErr      bool
		expectHasKey   bool
		expectCurrent  int
		expectVersions []int
	}{
		{name: "empty", raw: "", expectHasKey: false},
		{name: "spaces only", raw: "   ", expectHasKey: false},
		{name: "single", raw: good, expectHasKey: true, expectCurrent: 1, expectVersions: []int{1}},
		{name: "two versions highest wins", raw: fakeKeyList(2, 0x20) + "," + good, expectHasKey: true, expectCurrent: 2, expectVersions: []int{1, 2}},
		{name: "empty entries are skipped", raw: "," + good + ", ,", expectHasKey: true, expectCurrent: 1, expectVersions: []int{1}},
		{name: "no colon", raw: marker, expectErr: true},
		{name: "version not integer", raw: marker + ":AAAA", expectErr: true},
		{name: "version zero", raw: "0:" + strings.SplitN(good, ":", 2)[1], expectErr: true},
		{name: "version negative", raw: "-1:" + strings.SplitN(good, ":", 2)[1], expectErr: true},
		{name: "bad base64", raw: "1:" + marker + "!!", expectErr: true},
		{name: "wrong length", raw: "1:" + base64.StdEncoding.EncodeToString([]byte(marker)), expectErr: true},
		{name: "duplicate version", raw: good + "," + good, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.raw)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				if !errors.Is(err, ErrInvalidKeySet) {
					t.Errorf("Wrong match. expect: ErrInvalidKeySet, got: %v", err)
				}
				// the error never contains any text of the input
				for _, part := range []string{marker, tt.raw, strings.SplitN(good, ":", 2)[1]} {
					if part != "" && strings.Contains(err.Error(), part) {
						t.Errorf("Wrong match. the error contains input text: %q", part)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if c.HasKey() != tt.expectHasKey {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectHasKey, c.HasKey())
			}
			if c.CurrentVersion() != tt.expectCurrent {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectCurrent, c.CurrentVersion())
			}
			got := c.Versions()
			if len(got) != len(tt.expectVersions) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectVersions, got)
			}
			for i := range got {
				if got[i] != tt.expectVersions[i] {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectVersions, got)
				}
			}
		})
	}
}

func Test_nilCrypto(t *testing.T) {
	var c *Crypto

	if c.HasKey() {
		t.Errorf("Wrong match. expect: false, got: true")
	}
	if c.CurrentVersion() != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", c.CurrentVersion())
	}
	if c.Versions() != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", c.Versions())
	}
	if _, _, err := c.Encrypt("x"); !errors.Is(err, ErrKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrKeyNotConfigured, got: %v", err)
	}
	if _, err := c.Decrypt(make([]byte, 40), 1); !errors.Is(err, ErrKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrKeyNotConfigured, got: %v", err)
	}
}

func Test_noKey(t *testing.T) {
	c := mustNew(t, "")

	if _, _, err := c.Encrypt("x"); !errors.Is(err, ErrKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrKeyNotConfigured, got: %v", err)
	}
	if _, err := c.Decrypt(make([]byte, 40), 1); !errors.Is(err, ErrKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrKeyNotConfigured, got: %v", err)
	}
}

func Test_EncryptDecrypt(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		pt   string
		opts []Option

		decryptVer int // 0: use the version Encrypt returned

		expectEncryptErr error
		expectDecryptErr error
	}{
		{name: "round trip", raw: fakeKeyList(1, 1), pt: "plain value"},
		{name: "empty plaintext", raw: fakeKeyList(1, 1), pt: ""},
		{name: "unknown version", raw: fakeKeyList(1, 1), pt: "v", decryptVer: 9, expectDecryptErr: ErrDecrypt},
		{name: "too long", raw: fakeKeyList(1, 1), pt: "12345", opts: []Option{WithMaxPlaintext(4)}, expectEncryptErr: ErrTooLong},
		{name: "exactly the limit", raw: fakeKeyList(1, 1), pt: "1234", opts: []Option{WithMaxPlaintext(4)}},
		{name: "unlimited by default", raw: fakeKeyList(1, 1), pt: strings.Repeat("a", 100000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := mustNew(t, tt.raw, tt.opts...)

			blob, ver, err := c.Encrypt(tt.pt)
			if tt.expectEncryptErr != nil {
				if !errors.Is(err, tt.expectEncryptErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectEncryptErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(blob) != 12+len(tt.pt)+16 {
				t.Errorf("Wrong match. expect: %d, got: %d", 12+len(tt.pt)+16, len(blob))
			}

			useVer := ver
			if tt.decryptVer != 0 {
				useVer = tt.decryptVer
			}

			got, err := c.Decrypt(blob, useVer)
			if tt.expectDecryptErr != nil {
				if !errors.Is(err, tt.expectDecryptErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectDecryptErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got != tt.pt {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.pt, got)
			}
		})
	}
}

func Test_EncryptUsesFreshNonce(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 1))

	a, _, _ := c.Encrypt("same")
	b, _, _ := c.Encrypt("same")

	if bytes.Equal(a[:12], b[:12]) {
		t.Errorf("Wrong match. the nonce was reused")
	}
	if bytes.Equal(a, b) {
		t.Errorf("Wrong match. identical blobs for identical input")
	}
}

func Test_DecryptRejectsDamage(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 1))
	blob, ver, _ := c.Encrypt("value")

	tests := []struct {
		name string
		blob func() []byte
	}{
		{name: "flipped ciphertext bit", blob: func() []byte { b := bytes.Clone(blob); b[13] ^= 0x01; return b }},
		{name: "flipped tag bit", blob: func() []byte { b := bytes.Clone(blob); b[len(b)-1] ^= 0x01; return b }},
		{name: "flipped nonce bit", blob: func() []byte { b := bytes.Clone(blob); b[0] ^= 0x01; return b }},
		{name: "truncated", blob: func() []byte { return bytes.Clone(blob[:len(blob)-1]) }},
		{name: "shorter than nonce and tag", blob: func() []byte { return make([]byte, 27) }},
		{name: "nil", blob: func() []byte { return nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := c.Decrypt(tt.blob(), ver); !errors.Is(err, ErrDecrypt) {
				t.Errorf("Wrong match. expect: ErrDecrypt, got: %v", err)
			}
		})
	}
}

func Test_rotation(t *testing.T) {
	oldC := mustNew(t, fakeKeyList(1, 0x01))
	blob, ver, _ := oldC.Encrypt("value")

	// the new key list adds version 2; the row written under version 1 is still readable.
	newC := mustNew(t, fakeKeyList(1, 0x01)+","+fakeKeyList(2, 0x40))
	got, err := newC.Decrypt(blob, ver)
	if err != nil || got != "value" {
		t.Errorf("Wrong match. expect: value, got: %q, %v", got, err)
	}

	_, newVer, _ := newC.Encrypt("x")
	if newVer != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", newVer)
	}

	// version 1 removed: the old row can not be read.
	only2 := mustNew(t, fakeKeyList(2, 0x40))
	if _, err := only2.Decrypt(blob, ver); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. expect: ErrDecrypt, got: %v", err)
	}
}

// Test_stdlibInterop pins that the stored blob is plain AES-256-GCM with no associated data: a blob
// made here opens with the standard library alone, and a blob sealed with the standard library alone
// opens here. Any associated data would break both directions.
func Test_stdlibInterop(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 0x01))

	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x01 + byte(i)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	blob, ver, err := c.Encrypt("value")
	if err != nil || ver != 1 {
		t.Fatalf("Wrong match. got: %d, %v", ver, err)
	}
	pt, err := gcm.Open(nil, blob[:12], blob[12:], nil)
	if err != nil || string(pt) != "value" {
		t.Errorf("Wrong match. the standard library could not open the blob: %q, %v", pt, err)
	}

	nonce := make([]byte, 12)
	for i := range nonce {
		nonce[i] = 0xa0 + byte(i)
	}
	sealed := gcm.Seal(nonce, nonce, []byte("sealed elsewhere"), nil)
	got, err := c.Decrypt(sealed, 1)
	if err != nil || got != "sealed elsewhere" {
		t.Errorf("Wrong match. expect: sealed elsewhere, got: %q, %v", got, err)
	}
}

// Test_stdlibWithAADFails pins that a blob sealed with any associated data does not decrypt.
func Test_stdlibWithAADFails(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 0x01))

	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x01 + byte(i)
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	nonce := make([]byte, 12)
	sealed := gcm.Seal(nonce, nonce, []byte("value"), []byte("some data"))
	if _, err := c.Decrypt(sealed, 1); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. expect: ErrDecrypt, got: %v", err)
	}
}

func Test_ErrorsCarryNoSecretText(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 1))
	plain := "PLAINTEXT-MARKER-VALUE"
	blob, ver, _ := c.Encrypt(plain)
	blob[14] ^= 0x01

	_, err := c.Decrypt(blob, ver)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if strings.Contains(err.Error(), plain) {
		t.Errorf("Wrong match. the error contains the plaintext")
	}

	c2 := mustNew(t, fakeKeyList(1, 1), WithMaxPlaintext(3))
	_, _, err = c2.Encrypt(plain)
	if err == nil || strings.Contains(err.Error(), plain) {
		t.Errorf("Wrong match. got: %v", err)
	}
}

func Test_JoinSplit(t *testing.T) {
	nonce := bytes.Repeat([]byte{1}, 12)
	ct := bytes.Repeat([]byte{2}, 20)

	tests := []struct {
		name  string
		ct    []byte
		nonce []byte

		expectNil bool
		expectErr bool
	}{
		{name: "normal", ct: ct, nonce: nonce},
		{name: "both nil", expectNil: true},
		{name: "both empty", ct: []byte{}, nonce: []byte{}, expectNil: true},
		{name: "ct only", ct: ct, expectErr: true},
		{name: "nonce only", nonce: nonce, expectErr: true},
		{name: "nonce 11 bytes", ct: ct, nonce: nonce[:11], expectErr: true},
		{name: "nonce 13 bytes", ct: ct, nonce: append(bytes.Clone(nonce), 9), expectErr: true},
		{name: "ct shorter than tag", ct: ct[:15], nonce: nonce, expectErr: true},
		{name: "ct exactly the tag", ct: ct[:16], nonce: nonce},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := JoinSplit(tt.ct, tt.nonce)
			if tt.expectErr {
				if !errors.Is(err, ErrMalformed) {
					t.Errorf("Wrong match. expect: ErrMalformed, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectNil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %x", res)
				}
				return
			}
			expect := append(bytes.Clone(tt.nonce), tt.ct...)
			if !bytes.Equal(res, expect) {
				t.Errorf("Wrong match. expect: %x, got: %x", expect, res)
			}
		})
	}
}

func Test_SplitBlob(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 1))
	blob, ver, _ := c.Encrypt("value")

	ct, nonce, err := SplitBlob(blob)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(nonce) != 12 || !bytes.Equal(nonce, blob[:12]) || !bytes.Equal(ct, blob[12:]) {
		t.Errorf("Wrong match. the split is wrong")
	}

	// the parts are copies
	ct[0] ^= 0xff
	nonce[0] ^= 0xff
	if blob[12] == ct[0] || blob[0] == nonce[0] {
		t.Errorf("Wrong match. the split shares memory with the blob")
	}

	joined, err := JoinSplit(blob[12:], blob[:12])
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got, err := c.Decrypt(joined, ver); err != nil || got != "value" {
		t.Errorf("Wrong match. expect: value, got: %q, %v", got, err)
	}

	for _, n := range []int{0, 1, 27} {
		if _, _, err := SplitBlob(make([]byte, n)); !errors.Is(err, ErrMalformed) {
			t.Errorf("Wrong match. size %d expect: ErrMalformed, got: %v", n, err)
		}
	}
}

func Test_PackUnpackVersioned(t *testing.T) {
	c := mustNew(t, fakeKeyList(1, 1))
	blob, _, _ := c.Encrypt("value")

	tests := []struct {
		name    string
		version int

		expectErr bool
	}{
		{name: "one", version: 1},
		{name: "max", version: 65535},
		{name: "zero", version: 0, expectErr: true},
		{name: "negative", version: -1, expectErr: true},
		{name: "over max", version: 65536, expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed, err := PackVersioned(tt.version, blob)
			if tt.expectErr {
				if !errors.Is(err, ErrMalformed) {
					t.Errorf("Wrong match. expect: ErrMalformed, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if packed[0] != byte(tt.version>>8) || packed[1] != byte(tt.version) {
				t.Errorf("Wrong match. the version is not big endian: %x", packed[:2])
			}

			v, b, err := UnpackVersioned(packed)
			if err != nil || v != tt.version || !bytes.Equal(b, blob) {
				t.Errorf("Wrong match. got: %d, %v", v, err)
			}
		})
	}

	if _, err := PackVersioned(1, blob[:27]); !errors.Is(err, ErrMalformed) {
		t.Errorf("Wrong match. expect: ErrMalformed, got: %v", err)
	}
	if _, _, err := UnpackVersioned(make([]byte, 60)); !errors.Is(err, ErrMalformed) {
		t.Errorf("Wrong match. version 0 expect: ErrMalformed, got: %v", err)
	}
	if _, _, err := UnpackVersioned(append([]byte{0, 1}, make([]byte, 27)...)); !errors.Is(err, ErrMalformed) {
		t.Errorf("Wrong match. short expect: ErrMalformed, got: %v", err)
	}
}

func Test_New_onlyEmptyEntries(t *testing.T) {
	for _, raw := range []string{",", " , ,", ",,,"} {
		if _, err := New(raw); !errors.Is(err, ErrInvalidKeySet) {
			t.Errorf("Wrong match. raw %q expect: ErrInvalidKeySet, got: %v", raw, err)
		}
	}
}
