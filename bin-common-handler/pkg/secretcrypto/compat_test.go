package secretcrypto

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

// The vectors below were produced once, with random nonces, by the cipher code that the services
// used before this package existed (bin-number-manager/pkg/secretcrypto and the ai-manager
// SecretCrypto with the engine base key packing), under a fake key list made from a byte
// sequence: version 3, key bytes 0x10, 0x11, ... The plaintext is a fake value. They pin the
// stored layouts: every row that exists in a database must keep decrypting.
const (
	compatPlaintext = "fake-secret-value-for-compat-test"
	compatVersion   = 3

	// number_providers.secret_ciphertext: nonce || ciphertext || tag in one column. The vector was
	// sealed with the standard library alone (crypto/aes, cipher.NewGCM, Seal with a fixed nonce
	// 0xb1, 0xb2, ... and no associated data), so it does not depend on this package.
	compatNumberBlob = "b1b2b3b4b5b6b7b8b9babbbccefbcfeb0c88638bdc385653a116abc4624bf22edda30b5214c316e28b78045f05f79793cb6446ea4979908b79f917b65c"

	// The same plaintext sealed by the earlier layout of this column, which bound the ciphertext to
	// the provider name "fakeprovider" as associated data. It must not decrypt.
	compatLegacyAADBlob = "7632678b6e5da3a0ef25df1c05baaba9108bb10e3145b87e6872df5d6f702b53c3dbb295b105a763985c24fd6745b4c412f2fdaaefc7123863bd5090d5"

	// ai_mcp_servers: ciphertext and nonce in two columns,
	compatMCPCiphertext = "d0ac392ee93c2196a8564395dc4fd05e858b783c23c919b1addff6d913652b305ae693c6f40e619ed56a8977e2fcc0537b"
	compatMCPNonce      = "950785eb6d016e24a835afc9"

	// ai_ais.engine_base_key_enc: version(2) | nonce(12) | ciphertext+tag, no associated data
	compatEnginePacked = "000311e0a038c17fbb0039b2c862ad5bff6e8894d84ecef8e04c2cd63a8cd69e587767f3578936501bfd470427b5963186a39b95df5b7dbdeaaafcd254d804"
)

func compatCrypto(t *testing.T) *Crypto {
	t.Helper()

	return mustNew(t, fakeKeyList(compatVersion, 0x10))
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()

	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("Wrong match. bad vector: %v", err)
	}

	return b
}

func Test_compatNumberManagerLayout(t *testing.T) {
	c := compatCrypto(t)
	blob := mustHex(t, compatNumberBlob)

	got, err := c.Decrypt(blob, compatVersion)
	if err != nil || got != compatPlaintext {
		t.Fatalf("Wrong match. expect: %q, got: %q, %v", compatPlaintext, got, err)
	}

	// another version, a damaged blob or a truncated blob fails.
	if _, err := c.Decrypt(blob, compatVersion+1); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. other version expect: ErrDecrypt, got: %v", err)
	}
	damaged := bytes.Clone(blob)
	damaged[20] ^= 0x01
	if _, err := c.Decrypt(damaged, compatVersion); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. damaged expect: ErrDecrypt, got: %v", err)
	}
	if _, err := c.Decrypt(blob[:len(blob)-1], compatVersion); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. truncated expect: ErrDecrypt, got: %v", err)
	}

	// the library writes the same layout back (with the plaintext limit of the service).
	cc := mustNew(t, fakeKeyList(compatVersion, 0x10), WithMaxPlaintext(1024))
	again, ver, err := cc.Encrypt(compatPlaintext)
	if err != nil || ver != compatVersion || len(again) != len(blob) {
		t.Fatalf("Wrong match. got: %d, %d, %v", len(again), ver, err)
	}
	if got, err := c.Decrypt(again, ver); err != nil || got != compatPlaintext {
		t.Errorf("Wrong match. round trip: %q, %v", got, err)
	}
}

// Test_compatLegacyAADBlobFails pins that a row that was sealed with associated data does not
// decrypt: there is no read path for it.
func Test_compatLegacyAADBlobFails(t *testing.T) {
	c := compatCrypto(t)

	if _, err := c.Decrypt(mustHex(t, compatLegacyAADBlob), compatVersion); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. expect: ErrDecrypt, got: %v", err)
	}
}

func Test_compatAIManagerMCPLayout(t *testing.T) {
	c := compatCrypto(t)
	ct := mustHex(t, compatMCPCiphertext)
	nonce := mustHex(t, compatMCPNonce)

	joined, err := JoinSplit(ct, nonce)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, err := c.Decrypt(joined, compatVersion)
	if err != nil || got != compatPlaintext {
		t.Fatalf("Wrong match. expect: %q, got: %q, %v", compatPlaintext, got, err)
	}

	// a write in the new library splits back into the two columns.
	blob, ver, err := c.Encrypt(compatPlaintext)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	newCT, newNonce, err := SplitBlob(blob)
	if err != nil || len(newNonce) != 12 || len(newCT) != len(ct) {
		t.Fatalf("Wrong match. got: %d, %d, %v", len(newCT), len(newNonce), err)
	}
	back, err := JoinSplit(newCT, newNonce)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got, err := c.Decrypt(back, ver); err != nil || got != compatPlaintext {
		t.Errorf("Wrong match. round trip: %q, %v", got, err)
	}
}

func Test_compatEngineBaseKeyLayout(t *testing.T) {
	c := compatCrypto(t)
	packed := mustHex(t, compatEnginePacked)

	ver, blob, err := UnpackVersioned(packed)
	if err != nil || ver != compatVersion {
		t.Fatalf("Wrong match. expect: %d, got: %d, %v", compatVersion, ver, err)
	}

	got, err := c.Decrypt(blob, ver)
	if err != nil || got != compatPlaintext {
		t.Fatalf("Wrong match. expect: %q, got: %q, %v", compatPlaintext, got, err)
	}

	repacked, err := PackVersioned(ver, blob)
	if err != nil || !bytes.Equal(repacked, packed) {
		t.Errorf("Wrong match. the repacked value differs: %v", err)
	}
}
