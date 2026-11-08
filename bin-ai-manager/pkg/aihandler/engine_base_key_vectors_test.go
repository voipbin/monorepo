package aihandler

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"

	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"
)

// The vectors are deterministic and fake: key version 1 is 32 bytes of 0x01, key version 2 is 32
// bytes of 0x02, the nonce is 00 01 02 .. 0b, the plaintexts are "fake-secret-1" and
// "fake-secret-2". The stored value is version (2 bytes, big endian) || nonce || ciphertext || tag,
// where nonce || ciphertext || tag is gcm.Seal(nil, nonce, plaintext, nil) of the standard library
// prefixed by the nonce.
const (
	vectorKeys    = "1:AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=,2:AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
	vectorPacked1 = "0001000102030405060708090a0bddc7fe430d8d631a8c0f93e9118ed93318ef4de79eb5e793433167a976"
	vectorPacked2 = "0002000102030405060708090a0b2f69899ffc3ae92759ed343bac23d638790a10471d5c1d7de88bee80a7"
)

func mustVectorCrypto(t *testing.T) *secretcrypto.Crypto {
	t.Helper()

	res, err := mcpserverhandler.NewSecretCrypto(vectorKeys)
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}

	return res
}

func Test_engineBaseKeyVectors_storedValuesOpen(t *testing.T) {
	crypto := mustVectorCrypto(t)
	h := &aiHandler{crypto: crypto}

	tests := []struct {
		name   string
		packed string
		expect string
	}{
		{"key version 1", vectorPacked1, "fake-secret-1"},
		{"key version 2", vectorPacked2, "fake-secret-2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed, err := hex.DecodeString(tt.packed)
			if err != nil {
				t.Fatalf("could not decode the vector. err: %v", err)
			}

			res, err := h.openEngineBaseKey(packed)
			if err != nil {
				t.Fatalf("could not open the vector. err: %v", err)
			}
			if res != tt.expect {
				t.Errorf("wrong plaintext. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// Test_openEngineBaseKey_versionZero verifies that a stored value with the version 0 is refused as a
// malformed value. The version 0 is never written.
func Test_openEngineBaseKey_versionZero(t *testing.T) {
	h := &aiHandler{crypto: newTestCrypto(t, 1)}

	_, err := h.openEngineBaseKey(make([]byte, 30))
	if !errors.Is(err, errEngineBaseKeyBadCiphertext) {
		t.Errorf("expected errEngineBaseKeyBadCiphertext, got %v", err)
	}
}

// Test_engineBaseKeyVectors_newValuesReadByPreviousLogic verifies the rolling deploy in the other
// direction: a value written now has the 2 byte big endian version, a 12 byte nonce and the
// ciphertext with its tag, which the previous logic splits by hand and opens.
func Test_engineBaseKeyVectors_newValuesReadByPreviousLogic(t *testing.T) {
	h := &aiHandler{crypto: mustVectorCrypto(t)}

	packed, err := h.sealEngineBaseKey("fake-secret-2")
	if err != nil {
		t.Fatalf("could not seal. err: %v", err)
	}

	version := int(binary.BigEndian.Uint16(packed[:2]))
	nonce := packed[2:14]
	ct := packed[14:]
	if version != 2 {
		t.Errorf("wrong key version. expect: 2, got: %d", version)
	}

	block, err := aes.NewCipher(bytes.Repeat([]byte{byte(version)}, 32))
	if err != nil {
		t.Fatalf("could not create the cipher. err: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("could not create GCM. err: %v", err)
	}

	res, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		t.Fatalf("the previous logic could not open the new value. err: %v", err)
	}
	if string(res) != "fake-secret-2" {
		t.Errorf("wrong plaintext. got: %s", res)
	}
}

// Test_engineBaseKeyVectors_versionRange verifies the version range of the stored prefix: 65535 is
// written, 65536 and a missing key are refused with the same errors as before.
func Test_engineBaseKeyVectors_versionRange(t *testing.T) {
	raw := "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="

	tests := []struct {
		name   string
		keys   string
		expect error
	}{
		{"highest version", "65535:" + raw, nil},
		{"version out of range", "65536:" + raw, errEngineBaseKeyBadVersion},
		{"no key", "", errEngineBaseKeyCryptoUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto, err := mcpserverhandler.NewSecretCrypto(tt.keys)
			if err != nil {
				t.Fatalf("could not build the crypto. err: %v", err)
			}
			h := &aiHandler{crypto: crypto}

			_, err = h.sealEngineBaseKey("fake-secret-1")
			if !errors.Is(err, tt.expect) {
				t.Errorf("wrong error. expect: %v, got: %v", tt.expect, err)
			}
		})
	}
}
