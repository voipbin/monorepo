package providerhandler

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-number-manager/models/provider"
)

// The vectors below were sealed once with the standard library alone (crypto/aes, cipher.NewGCM,
// Seal with a fixed nonce and no associated data) under fake keys: version 1 is the bytes 0x40+i and
// version 2 is the bytes 0x90+i (i = 0..31). The nonce of each vector starts at 0x10 (A), 0x20 (B),
// 0x30 (C), 0x40 (D) and 0x50 (E) and counts up. They pin the stored layout of
// number_providers.secret_ciphertext (nonce || ciphertext || tag): every row that exists in a
// database must keep decrypting.
const (
	vectorTwoKeyA = "101112131415161718191a1bf1f36c58d0f79aa581ef2b3f3687a7e81ba82f048bdd543115f267191e72aa854269626dbafd71149c948b36d9d8db78cb5a7b71104b"
	vectorTwoKeyB = "202122232425262728292a2b97a302f472df370e5475daf876ec0d9860c3412320c2eb93e092aec202707932d1da17e18d0799abd9bf540506dc6e82f14e8d1bd002"
	vectorOneKeyC = "303132333435363738393a3be7b6895a55c8b98c8370b06e98789e9cfb4b2111c9fcbe17efc8516bad7da0dbbbf1129202e13c28846ebe20e2013ea73328e04ceb85fbc3c9"
	vectorOneKeyD = "404142434445464748494a4b07162248b1ac5a8be485d51dad5659dd20b498d22e7b9022ae751005cee368396dc7232c5405e9d810800187"
	vectorOneKeyE = "505152535455565758595a5bf72a7363b9d0135dba8aa24aff25e2dc002b69b258195dbba588ecac571022a1df0a7478" +
		"cd7fb049312f2599e0d1d22a8e12880f0e21fe9a84b73a277ae00cf32bf4b4607f55e3b00bc968aa4ef75026ec037900" +
		"fa4f2e5f35aac26429829c5fc0ba957d23520cff77cd02f0643414bbea32a83c90b8e49383d0f4d9ad26e3bc6555f5f2" +
		"9437822389dbef16bdd0d69d2a326e540f072b89c772b6af0c794f15584060686b97442ed3d8d58badbcabce77cd5a1e" +
		"002c0317c85fdf0fa402d3bf23222c7115304885559f176974a567c920512e380f01bfd6f416fd863a70b28b52171499" +
		"e1cde8d778142f4954b675bba9a2f43873113b91c5d8792f216b9a1478a73a9544301452cd4ed36d049ed5e400b6d50e" +
		"788c8c605dbe8c47c63add97a728a9a9f4fc647a936838a75df187be3776e32f62874bd201b2c5331e29c9d4774cde9e" +
		"7630ab215388521d1355a3a9089aae036ef9def2db11788f3acae5bdf102e3b44e175251495f332fd6f220200e51430d" +
		"a8d69ad13cc9f25b4cdab28c99a0d7bf7072d56864ceafd03474f9ad0d20e683e3e1b32b41c33dcfb3828ff9b99311a1" +
		"53cd6e4d8f1ab3cfd75a832f72b761fceea79d70ba23b7499d762d3514e6f051515ea71c4dabd1155629c694420bc58b" +
		"fee8ea50a1094db6e3f7ce636b885c25dca59c0e6226ac656e45e39a53a4b609bed95e19ee78d74c2589f6dbaa479e5b" +
		"9734d439a54cef792f4d0f95c9fa4f307b2c5c85fad51ebb6952c976c67f951e0702c0bda7567710cbfd8e9abc478c2f" +
		"7b41fd16fd5e3b42f4b38debaace9f7278f1014bc1d5440179b903c9a92b2a572249ae017a47c143217c2c34640bf966" +
		"98b3c43ad0fe0d15f8680869b36357a8035b429d2d67df2e0b31b97795033bef00ad8cbc0c8e3fd25ae043044a1c208e" +
		"e3dabbc22e40cd09223b4d66189393a297aec0d982dbcfa299010564fad5f28c0f0b4a26c3c0aa6843fa64c2d2bf40f1" +
		"2c8ce581b3e1d6268742e394b9368a34a444bb62f2f4507b6f2bced04e07b68757531a1022c33d003c2bfda7bc42ccf4" +
		"90a2f25f4bab5fe2f2ec7d1144e4ac28ca63f6ccfd2d48411a10d92a28cee24637101bfb4198e714f385cc37e9d9717a" +
		"baa577ed805661a8de6db367d35805fe524a1ac5154fe2fb37816d4e118f56f9dfbd680225177a013a12a2308ecd9a41" +
		"211e6c24a7c5a32737f5726ea227d6d5425afd21e3ee735e36626dd35eec55067e19da0f333d797d2b8ab2ffed13919d" +
		"270091f81f19bdc21349378a07ac831eda16af1cdfd78d9be5579bb9185182d1d83bcef72ca4f9bff88dabc2a27803e8" +
		"f1ecdd6ef159c75c3f0eae142ed99863508ffd1099a5f9d0e1c337da5bf670d085da9da38993852df92cc2004bad879b" +
		"1a6389cf560a5f4a1ef75ebf292047fd9df0eda01f5b30e3dd6414c80ac693c17256c7b4c88b3ded47c395f3"

	// legacyAADVectorTwoKeyA is the plaintext of vectorTwoKeyA sealed under the same key with the
	// provider name "telnyx" as associated data, which the earlier layout used. It must not decrypt.
	legacyAADVectorTwoKeyA = "1f3d9d300d26711ff38e185ae943659cb74f8a1ccd87a0bd877e02bf0feca56d8431defd89fbfe00326fbc14d8bf725e32a7a793909af912bc3803995323d4d7104f"
)

func fakeKeyList(versions ...int) string {
	parts := make([]string, 0, len(versions))
	for _, ver := range versions {
		key := make([]byte, 32)
		base := byte(0x40)
		if ver == 2 {
			base = 0x90
		}
		for i := range key {
			key[i] = base + byte(i)
		}
		parts = append(parts, strconv.Itoa(ver)+":"+base64.StdEncoding.EncodeToString(key))
	}

	return strings.Join(parts, ",")
}

func mustNewSecretCrypto(t *testing.T, raw string) *secretcrypto.Crypto {
	t.Helper()

	c, err := NewSecretCrypto(raw)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return c
}

func Test_decryptSecret_storedVectors(t *testing.T) {
	two := mustNewSecretCrypto(t, fakeKeyList(1, 2))
	one := mustNewSecretCrypto(t, fakeKeyList(1))

	tests := []struct {
		name string

		crypto  *secretcrypto.Crypto
		vector  string
		version int

		expectRes string
	}{
		{"telnyx version 2", two, vectorTwoKeyA, 2, "fake-provider-secret-for-vector-telnyx"},
		{"twilio version 2", two, vectorTwoKeyB, 2, "fake-provider-secret-for-vector-twilio"},
		{"telnyx version 1", one, vectorOneKeyC, 1, "fake-provider-secret-for-vector-telnyx-v1"},
		{"non-ASCII plaintext", one, vectorOneKeyD, 1, "가짜-비밀-ünïcode-fake"},
		{"plaintext at the limit", one, vectorOneKeyE, 1, strings.Repeat("a", maxSecretBytes)},
		{"old row after a rotation", two, vectorOneKeyC, 1, "fake-provider-secret-for-vector-telnyx-v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob, err := hex.DecodeString(tt.vector)
			if err != nil {
				t.Fatalf("Wrong match. bad vector: %v", err)
			}

			res, err := decryptSecret(tt.crypto, blob, tt.version)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}

func Test_decryptSecret_failures(t *testing.T) {
	two := mustNewSecretCrypto(t, fakeKeyList(1, 2))
	onlyTwo := mustNewSecretCrypto(t, fakeKeyList(2))
	none := mustNewSecretCrypto(t, "")

	blobA, _ := hex.DecodeString(vectorTwoKeyA)
	blobC, _ := hex.DecodeString(vectorOneKeyC)
	blobLegacy, _ := hex.DecodeString(legacyAADVectorTwoKeyA)
	damaged := append([]byte{}, blobA...)
	damaged[len(damaged)-1] ^= 0xff

	tests := []struct {
		name string

		crypto  *secretcrypto.Crypto
		blob    []byte
		version int
	}{
		{"sealed with associated data by an earlier layout", two, blobLegacy, 2},
		{"damaged blob", two, damaged, 2},
		{"key version removed", onlyTwo, blobC, 1},
		{"wrong row version", two, blobA, 1},
		{"no key configured", none, blobA, 2},
		{"nil crypto", nil, blobA, 2},
		{"blob of 12 bytes", two, make([]byte, 12), 2},
		{"blob of 27 bytes", two, make([]byte, 27), 2},
		{"empty blob", two, nil, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := decryptSecret(tt.crypto, tt.blob, tt.version)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != "" {
				t.Errorf("Wrong match. expect: no plaintext, got: %q", res)
			}
		})
	}
}

func Test_encryptSecret(t *testing.T) {
	c := mustNewSecretCrypto(t, fakeKeyList(1, 2))

	blob, version, err := encryptSecret(c, "fake-secret-value")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if version != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", version)
	}
	if strings.Contains(string(blob), "fake-secret-value") {
		t.Errorf("Wrong match. the ciphertext holds the plaintext")
	}

	res, err := decryptSecret(c, blob, version)
	if err != nil || res != "fake-secret-value" {
		t.Fatalf("Wrong match. expect: round trip, got: %q, %v", res, err)
	}
}

func Test_encryptSecret_errors(t *testing.T) {
	withKey := mustNewSecretCrypto(t, fakeKeyList(1))
	noKey := mustNewSecretCrypto(t, "")

	t.Run("no key", func(t *testing.T) {
		_, _, err := encryptSecret(noKey, "x")
		if !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
			t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
		}
	})

	t.Run("nil crypto", func(t *testing.T) {
		_, _, err := encryptSecret(nil, "x")
		if !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
			t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
		}
	})

	t.Run("key text of the sentinel", func(t *testing.T) {
		_, _, err := encryptSecret(noKey, "x")
		if err == nil || err.Error() != "secret encryption key is not configured" {
			t.Errorf("Wrong match. expect: the unchanged text, got: %v", err)
		}
	})

	t.Run("longest secret is accepted", func(t *testing.T) {
		if _, _, err := encryptSecret(withKey, strings.Repeat("a", maxSecretBytes)); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("one byte more is invalid", func(t *testing.T) {
		_, _, err := encryptSecret(withKey, strings.Repeat("a", maxSecretBytes+1))
		if !errors.Is(err, provider.ErrInvalidArgument) {
			t.Fatalf("Wrong match. expect: ErrInvalidArgument, got: %v", err)
		}
		if err.Error() != "invalid argument: the secret is longer than 1024 bytes" {
			t.Errorf("Wrong match. expect: the unchanged text, got: %v", err)
		}
	})

	t.Run("no key wins over the length, as before", func(t *testing.T) {
		_, _, err := encryptSecret(noKey, strings.Repeat("a", maxSecretBytes+1))
		if !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
			t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
		}
	})
}

func Test_NewSecretCrypto(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("short"))
	key1 := strings.TrimPrefix(fakeKeyList(1), "1:")

	tests := []struct {
		name string
		raw  string

		expectErr     bool
		expectHasKey  bool
		expectVersion int
	}{
		{"empty", "", false, false, 0},
		{"spaces", "   ", false, false, 0},
		{"only a comma", ",", false, false, 0},
		{"commas and spaces", " , ,, ", false, false, 0},
		{"single", fakeKeyList(1), false, true, 1},
		{"highest wins regardless of order", "2:" + strings.TrimPrefix(fakeKeyList(2), "2:") + ",1:" + key1, false, true, 2},
		{"trailing comma", fakeKeyList(1) + ",", false, true, 1},
		{"leading comma", "," + fakeKeyList(1), false, true, 1},
		{"empty entry between keys", fakeKeyList(1) + ", ," + "2:" + strings.TrimPrefix(fakeKeyList(2), "2:"), false, true, 2},
		{"no colon", "abc", true, false, 0},
		{"version not integer", "x:" + key1, true, false, 0},
		{"version zero", "0:" + key1, true, false, 0},
		{"version negative", "-1:" + key1, true, false, 0},
		{"bad base64", "1:!!!", true, false, 0},
		{"not 32 bytes", "1:" + short, true, false, 0},
		{"duplicate version", fakeKeyList(1) + ",1:" + key1, true, false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewSecretCrypto(tt.raw)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if err != nil {
				if !errors.Is(err, secretcrypto.ErrInvalidKeySet) {
					t.Errorf("Wrong match. expect: ErrInvalidKeySet, got: %v", err)
				}
				// the error must not echo the key material or the raw entry
				if strings.Contains(err.Error(), key1) || strings.Contains(err.Error(), short) {
					t.Errorf("Wrong match. the error leaks key material: %v", err)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil crypto with an error, got: %v", res)
				}
				return
			}

			if res.HasKey() != tt.expectHasKey || res.CurrentVersion() != tt.expectVersion {
				t.Errorf("Wrong match. expect: %v, %d, got: %v, %d", tt.expectHasKey, tt.expectVersion, res.HasKey(), res.CurrentVersion())
			}
		})
	}
}

// The daemon replaces a key list that cannot be parsed by an empty one and keeps running; the
// provider secrets of the database then stay unusable.
func Test_NewSecretCrypto_daemonFallback(t *testing.T) {
	if _, err := NewSecretCrypto("not-a-key-list"); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	fallback, err := NewSecretCrypto("")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if fallback.HasKey() {
		t.Errorf("Wrong match. expect: no key, got: a key")
	}

	if _, _, err := encryptSecret(fallback, "x"); !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
	}
}
