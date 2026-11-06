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

	// number_providers.secret_ciphertext, AAD = provider name
	compatNumberBlob = "7632678b6e5da3a0ef25df1c05baaba9108bb10e3145b87e6872df5d6f702b53c3dbb295b105a763985c24fd6745b4c412f2fdaaefc7123863bd5090d5"
	compatNumberAAD  = "fakeprovider"

	// ai_mcp_servers: ciphertext and nonce in two columns, no AAD
	compatMCPCiphertext = "d0ac392ee93c2196a8564395dc4fd05e858b783c23c919b1addff6d913652b305ae693c6f40e619ed56a8977e2fcc0537b"
	compatMCPNonce      = "950785eb6d016e24a835afc9"

	// ai_ais.engine_base_key_enc: version(2) | nonce(12) | ciphertext+tag, no AAD
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

	got, err := c.Decrypt(blob, compatVersion, AADLiteral(compatNumberAAD))
	if err != nil || got != compatPlaintext {
		t.Fatalf("Wrong match. expect: %q, got: %q, %v", compatPlaintext, got, err)
	}

	// the same row under another provider name, another version, a damaged blob or no AAD fails.
	if _, err := c.Decrypt(blob, compatVersion, AADLiteral("otherprovider")); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. other AAD expect: ErrDecrypt, got: %v", err)
	}
	if _, err := c.Decrypt(blob, compatVersion, nil); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. no AAD expect: ErrDecrypt, got: %v", err)
	}
	if _, err := c.Decrypt(blob, compatVersion+1, AADLiteral(compatNumberAAD)); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. other version expect: ErrDecrypt, got: %v", err)
	}
	damaged := bytes.Clone(blob)
	damaged[20] ^= 0x01
	if _, err := c.Decrypt(damaged, compatVersion, AADLiteral(compatNumberAAD)); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. damaged expect: ErrDecrypt, got: %v", err)
	}
	if _, err := c.Decrypt(blob[:len(blob)-1], compatVersion, AADLiteral(compatNumberAAD)); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. truncated expect: ErrDecrypt, got: %v", err)
	}

	// the new library writes the same layout back (with the provider name limit of the service).
	cc := mustNew(t, fakeKeyList(compatVersion, 0x10), WithMaxPlaintext(1024))
	again, ver, err := cc.Encrypt(compatPlaintext, AADLiteral(compatNumberAAD))
	if err != nil || ver != compatVersion || len(again) != len(blob) {
		t.Fatalf("Wrong match. got: %d, %d, %v", len(again), ver, err)
	}
	if got, err := c.Decrypt(again, ver, AADLiteral(compatNumberAAD)); err != nil || got != compatPlaintext {
		t.Errorf("Wrong match. round trip: %q, %v", got, err)
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

	got, err := c.Decrypt(joined, compatVersion, nil)
	if err != nil || got != compatPlaintext {
		t.Fatalf("Wrong match. expect: %q, got: %q, %v", compatPlaintext, got, err)
	}

	// the legacy mode has no AAD: any AAD fails.
	if _, err := c.Decrypt(joined, compatVersion, AAD("ai_mcp_servers", "secret")); !errors.Is(err, ErrDecrypt) {
		t.Errorf("Wrong match. with AAD expect: ErrDecrypt, got: %v", err)
	}

	// a write in the new library splits back into the two columns.
	blob, ver, err := c.Encrypt(compatPlaintext, nil)
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
	if got, err := c.Decrypt(back, ver, nil); err != nil || got != compatPlaintext {
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

	got, err := c.Decrypt(blob, ver, nil)
	if err != nil || got != compatPlaintext {
		t.Fatalf("Wrong match. expect: %q, got: %q, %v", compatPlaintext, got, err)
	}

	repacked, err := PackVersioned(ver, blob)
	if err != nil || !bytes.Equal(repacked, packed) {
		t.Errorf("Wrong match. the repacked value differs: %v", err)
	}
}

// Test_compatAADContract pins the byte rule of the configs row AAD: table, 0x00, key. The vector
// was sealed with the standard library alone (AES-256-GCM, a fixed nonce, and the AAD bytes
// written out by hand), so it does not depend on this package. Changing the rule would make every
// stored configs row unreadable.
func Test_compatAADContract(t *testing.T) {
	c := compatCrypto(t)

	blob := mustHex(t, "a1a2a3a4a5a6a7a8a9aaabacf70cc315209c625beb3152de9a04de755e4f8dd1")

	got, err := c.Decrypt(blob, compatVersion, AAD("email_configs", "mailgun.api_key"))
	if err != nil || got != "fake" {
		t.Fatalf("Wrong match. expect: fake, got: %q, %v", got, err)
	}

	got, err = c.Decrypt(blob, compatVersion, []byte("email_configs\x00mailgun.api_key"))
	if err != nil || got != "fake" {
		t.Errorf("Wrong match. the hand written AAD expect: fake, got: %q, %v", got, err)
	}
}
