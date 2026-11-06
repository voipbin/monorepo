package providerhandler

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
)

// The vectors below were produced once by the previous service copy of the cipher code
// (bin-number-manager/pkg/secretcrypto, unchanged on origin/main d43e84007) under fake keys:
// version 1 is the bytes 0x40+i and version 2 is the bytes 0x90+i (i = 0..31). They pin the stored
// layout of number_providers.secret_ciphertext (nonce || ciphertext || tag, AAD = provider name):
// every row that exists in a database must keep decrypting.
const (
	vectorTwoKeyA = "1f3d9d300d26711ff38e185ae943659cb74f8a1ccd87a0bd877e02bf0feca56d8431defd89fbfe00326fbc14d8bf725e32a7a793909af912bc3803995323d4d7104f"
	vectorTwoKeyB = "c2c58809b756f16f395d18959abf223f2fe573a889844fd64ebe22cb619b0e7c8844aca01f1e4fe207fc6d85fd095979a79f9a1574a5750469115ef8073f99c8756e"
	vectorOneKeyC = "140fa1ecb76a9b2f8c49638cd41bbd3d7ad2190fcec42286e4498c77e29a9a4b7419cf04a7d19a4dadf00f84a9a594b04b8777bd7da3002f0289e44a206331d1d4f91de0d1"
	vectorOneKeyD = "ddab6d66374d2de68ba3dc16daff85067c295a7b49141fb7e4fb13695dae865f812de3e8000df51221394ee2d6a5e7492c9074e6e6b72d04"
	vectorOneKeyE = "50d3725dd561250029596c5ca49d6e7a64abef90d872623b72bbc5b980d81d4c758fc8455fa3920f8e0a9a8620268a5e" +
		"769fe494b36ea5c9e951357864976136e12fb645578ed6b92150a1d439bae71006194a7c0892cb5171d42f833abce1c3" +
		"90ee2fb979fe8ea01bb0bca072f3b3e5fd0abc4bb08d521721bf4699151510ab48aca5b875b4fcce4caeeb2e5ef6a98b" +
		"048d8938034110eb8d17e7e93493e4d9fc14be64009aa6de4fe388c7cf8c7fb62e7db07567b0f520ca815c0e39c73dc7" +
		"c5e4c17b811ec21679cdaa9b530a2d2ae75f88807f5bec9574c779c7c5dc26e494c9d958ef56bef52213a49278e7e254" +
		"379a1603d22b15e49bfa354e862d601b15f05a1b8bc8b35fbf20a50470c49328740ace720f60f8ab78ba5642ed4568a7" +
		"d62517d859a7b8a8c2a1d392f23ab7e04b119944c486972666e2ed59cea6b7ee65a64fe469bfab84c116f605a83d4bc7" +
		"90b44884ebca42eb75e0eb2a33d5b1103daf774d96ebf1b2d3c5255405271ce1d0dcf93e0a3d8cc02067b6f1d62963b6" +
		"1f44d2c0711269a4d780f49b318c11608b7f8242d194c0de791dd0fb20f5ff393989b0b8fe4c6f9a012dcdd3587a8203" +
		"5a0ef38eacea54338a73590c81a941d4c840b77ed59c2a46795980c54f3ac377d5e20df673bdae292cfd3e3cdf62ccff" +
		"3cdce571322822d75e5e70ed6a0e2f9bba08e1be7353c24d89e4a874debaa77a6bb37d32f1870d898243a490439675e5" +
		"fcb824a00fb1938c29284a22822fc6e59a07da162638cea631efbbd8338487705e24ac0dab355096d19d0380a0c898f3" +
		"ceb2ace304e4226a773735fc1468a1a57cab3cdac8080fb7e13eb183b42018abbfe61218e05c2247a382ff5bbe72343d" +
		"1f8babcd663b898f070fc65e36d3fbbd82507d2fef8cdc21071ea3ef713a8bf91b3ceed13ff12d3d369120ea5d100aa0" +
		"e88081eac087fb015004c5ca9c9b933b9ae3e631dc3a514b9e2e7518b760f24d2a9d283e764608caf344059fd9424605" +
		"a1671ac0c2b7a2cb29d2bc6d375c610a5fe953a1c48d6ab7d4c972fd424356dc31c261369623943329aeb5773d1e054c" +
		"93a255860167d9da504ce235282ba1db64908732f94a754a1be2486307dad53a59c05ab111b4814e839c1fca7f5e721a" +
		"53feabc4e00d783e739886ecdf65581a1d984eff9c445a6740fcc6f5387d14e916c90560ded5c7377235cbfa17be7a0b" +
		"a127a9c0e3c1af312042bdbd856162471e8031068e2a5dbb5f8e02929c27484c0c793b960722060df2b25dbc3f794752" +
		"d9a35735197c92cdd8a8374986eea6d9da1ad945976355d8885ce1d83599bea65997cf92d00c534eac65d12a91e400c4" +
		"7101e9d39a3a39cf6edb11b2bae400666707847385f775918bb740aba2da2c84205887574b0d74ffe012acaef5ea4b2e" +
		"dc2cad25cd3160711f6df91d39aa668174b2ee4a0257645246255c4557e029a4988d339d2c6cee64de5d3f94"
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

		crypto       *secretcrypto.Crypto
		vector       string
		version      int
		providerName number.ProviderName

		expectRes string
	}{
		{"telnyx version 2", two, vectorTwoKeyA, 2, number.ProviderNameTelnyx, "fake-provider-secret-for-vector-telnyx"},
		{"twilio version 2", two, vectorTwoKeyB, 2, number.ProviderNameTwilio, "fake-provider-secret-for-vector-twilio"},
		{"telnyx version 1", one, vectorOneKeyC, 1, number.ProviderNameTelnyx, "fake-provider-secret-for-vector-telnyx-v1"},
		{"non-ASCII plaintext", one, vectorOneKeyD, 1, number.ProviderNameTwilio, "가짜-비밀-ünïcode-fake"},
		{"plaintext at the limit", one, vectorOneKeyE, 1, number.ProviderNameTelnyx, strings.Repeat("a", maxSecretBytes)},
		{"old row after a rotation", two, vectorOneKeyC, 1, number.ProviderNameTelnyx, "fake-provider-secret-for-vector-telnyx-v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob, err := hex.DecodeString(tt.vector)
			if err != nil {
				t.Fatalf("Wrong match. bad vector: %v", err)
			}

			res, err := decryptSecret(tt.crypto, blob, tt.version, tt.providerName)
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
	damaged := append([]byte{}, blobA...)
	damaged[len(damaged)-1] ^= 0xff

	tests := []struct {
		name string

		crypto  *secretcrypto.Crypto
		blob    []byte
		version int
		pname   number.ProviderName
	}{
		{"another provider name", two, blobA, 2, number.ProviderNameTwilio},
		{"damaged blob", two, damaged, 2, number.ProviderNameTelnyx},
		{"key version removed", onlyTwo, blobC, 1, number.ProviderNameTelnyx},
		{"wrong row version", two, blobA, 1, number.ProviderNameTelnyx},
		{"no key configured", none, blobA, 2, number.ProviderNameTelnyx},
		{"nil crypto", nil, blobA, 2, number.ProviderNameTelnyx},
		{"blob of 12 bytes", two, make([]byte, 12), 2, number.ProviderNameTelnyx},
		{"blob of 27 bytes", two, make([]byte, 27), 2, number.ProviderNameTelnyx},
		{"empty blob", two, nil, 2, number.ProviderNameTelnyx},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := decryptSecret(tt.crypto, tt.blob, tt.version, tt.pname)
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

	blob, version, err := encryptSecret(c, "fake-secret-value", number.ProviderNameTelnyx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if version != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", version)
	}
	if strings.Contains(string(blob), "fake-secret-value") {
		t.Errorf("Wrong match. the ciphertext holds the plaintext")
	}

	res, err := decryptSecret(c, blob, version, number.ProviderNameTelnyx)
	if err != nil || res != "fake-secret-value" {
		t.Fatalf("Wrong match. expect: round trip, got: %q, %v", res, err)
	}
	if _, err := decryptSecret(c, blob, version, number.ProviderNameTwilio); err == nil {
		t.Errorf("Wrong match. expect: failure for another provider name, got: nil")
	}
}

func Test_encryptSecret_errors(t *testing.T) {
	withKey := mustNewSecretCrypto(t, fakeKeyList(1))
	noKey := mustNewSecretCrypto(t, "")

	t.Run("no key", func(t *testing.T) {
		_, _, err := encryptSecret(noKey, "x", number.ProviderNameTelnyx)
		if !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
			t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
		}
	})

	t.Run("nil crypto", func(t *testing.T) {
		_, _, err := encryptSecret(nil, "x", number.ProviderNameTelnyx)
		if !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
			t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
		}
	})

	t.Run("key text of the sentinel", func(t *testing.T) {
		_, _, err := encryptSecret(noKey, "x", number.ProviderNameTelnyx)
		if err == nil || err.Error() != "secret encryption key is not configured" {
			t.Errorf("Wrong match. expect: the unchanged text, got: %v", err)
		}
	})

	t.Run("longest secret is accepted", func(t *testing.T) {
		if _, _, err := encryptSecret(withKey, strings.Repeat("a", maxSecretBytes), number.ProviderNameTelnyx); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("one byte more is invalid", func(t *testing.T) {
		_, _, err := encryptSecret(withKey, strings.Repeat("a", maxSecretBytes+1), number.ProviderNameTelnyx)
		if !errors.Is(err, provider.ErrInvalidArgument) {
			t.Fatalf("Wrong match. expect: ErrInvalidArgument, got: %v", err)
		}
		if err.Error() != "invalid argument: the secret is longer than 1024 bytes" {
			t.Errorf("Wrong match. expect: the unchanged text, got: %v", err)
		}
	})

	t.Run("no key wins over the length, as before", func(t *testing.T) {
		_, _, err := encryptSecret(noKey, strings.Repeat("a", maxSecretBytes+1), number.ProviderNameTelnyx)
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

	if _, _, err := encryptSecret(fallback, "x", number.ProviderNameTelnyx); !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", err)
	}
}
