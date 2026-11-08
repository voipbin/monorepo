package mcpserverhandler

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"testing"
)

// Fixed vectors of the stored layouts. They are deterministic and fake: key version 1 is 32 bytes of
// 0x01, key version 2 is 32 bytes of 0x02, the nonce is 00 01 02 .. 0b, the plaintexts are
// "fake-secret-1" (version 1) and "fake-secret-2" (version 2). Each value is
// gcm.Seal(nil, nonce, plaintext, nil) of the standard library, which is what the stored
// ciphertext column holds.
const (
	vectorKeys   = "1:AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=,2:AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
	vectorNonce  = "000102030405060708090a0b"
	vectorCT1    = "ddc7fe430d8d631a8c0f93e9118ed93318ef4de79eb5e793433167a976"
	vectorCT2    = "2f69899ffc3ae92759ed343bac23d638790a10471d5c1d7de88bee80a7"
	vectorPlain1 = "fake-secret-1"
	vectorPlain2 = "fake-secret-2"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()

	res, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("could not decode the vector. err: %v", err)
	}

	return res
}

// Test_secretVectors_storedValuesDecrypt verifies that values written by the previous implementation
// (ciphertext and nonce in two columns) are read by the shared library.
func Test_secretVectors_storedValuesDecrypt(t *testing.T) {
	crypto, err := NewSecretCrypto(vectorKeys)
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}

	tests := []struct {
		name    string
		ct      string
		version int
		expect  string
	}{
		{"key version 1", vectorCT1, 1, vectorPlain1},
		{"key version 2", vectorCT2, 2, vectorPlain2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := DecryptSecret(crypto, mustHex(t, tt.ct), mustHex(t, vectorNonce), tt.version)
			if err != nil {
				t.Fatalf("could not decrypt the vector. err: %v", err)
			}
			if res != tt.expect {
				t.Errorf("wrong plaintext. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// openAsBefore reads a value the way the previous implementation did: the key of the row version is
// 32 bytes of that version number (the fake keys of this file), and the ciphertext with its tag is
// opened with the nonce of the second column and no additional data. It is the only copy of that
// logic and exists to prove that new values stay readable by instances that still run it.
func openAsBefore(t *testing.T, version int, ct []byte, nonce []byte) string {
	t.Helper()

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

	return string(res)
}

// Test_secretVectors_newValuesReadByPreviousLogic verifies the rolling deploy in the other direction:
// a value written now is a 12 byte nonce and a ciphertext that the previous logic opens, under the
// highest key version.
func Test_secretVectors_newValuesReadByPreviousLogic(t *testing.T) {
	crypto, err := NewSecretCrypto(vectorKeys)
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}

	ct, nonce, version, err := EncryptSecret(crypto, vectorPlain2)
	if err != nil {
		t.Fatalf("could not encrypt. err: %v", err)
	}
	if version != 2 {
		t.Errorf("wrong key version. expect: 2, got: %d", version)
	}
	if len(nonce) != 12 {
		t.Errorf("wrong nonce length. expect: 12, got: %d", len(nonce))
	}

	if res := openAsBefore(t, version, ct, nonce); res != vectorPlain2 {
		t.Errorf("wrong plaintext. expect: %s, got: %s", vectorPlain2, res)
	}
}
