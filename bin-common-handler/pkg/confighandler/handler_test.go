package confighandler_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
	"monorepo/bin-common-handler/pkg/secretcrypto"
)

var (
	testActor = uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")
	testNow   = time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
)

func testKeyList(version int, start byte) string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = start + byte(i)
	}

	return fmt.Sprintf("%d:%s", version, base64.StdEncoding.EncodeToString(k))
}

func testCrypto(t *testing.T, raw string) *secretcrypto.Crypto {
	t.Helper()

	c, err := secretcrypto.New(raw)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return c
}

func testRegistry(t *testing.T) *confighandler.Registry {
	t.Helper()

	reg, err := confighandler.NewRegistry([]confighandler.Key{
		{Name: "p.api_key", Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "P_API_KEY", MaxLen: 32, Group: "p", Atomic: true},
		{Name: "p.enabled", Kind: confighandler.KindSelect, Type: confighandler.TypeBool, Env: "P_ENABLED", Default: true, Group: "p", Atomic: true},
		{Name: "p.model", Kind: confighandler.KindParam, Type: confighandler.TypeString, Env: "P_MODEL", MaxLen: 32, Default: "model-default", Group: "p", Atomic: true},
		{Name: "s.token", Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "S_TOKEN", MaxLen: 16},
		{Name: "a.limit", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Env: "A_LIMIT", Default: 10, Min: ptr(1), Max: ptr(100)},
		{Name: "a.timeout", Kind: confighandler.KindParam, Type: confighandler.TypeDuration, Env: "A_TIMEOUT", Default: 5 * time.Second, Min: ptr(int64(time.Second))},
		{Name: "a.mode", Kind: confighandler.KindSelect, Type: confighandler.TypeEnum, Env: "A_MODE", Enum: []string{"x", "y"}, Default: "x"},
		{Name: "a.url", Kind: confighandler.KindParam, Type: confighandler.TypeString, Env: "A_URL", MaxLen: 64, Sensitive: true},
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return reg
}

type fixture struct {
	h      confighandler.Handler
	store  *storetest.Memory
	crypto *secretcrypto.Crypto
	env    map[string]string
}

func newFixture(t *testing.T, keyList string, opts ...confighandler.Option) *fixture {
	t.Helper()

	f := &fixture{store: storetest.NewMemory(), crypto: testCrypto(t, keyList), env: map[string]string{}}
	lookup := func(name string) (string, bool) {
		v, ok := f.env[name]
		return v, ok
	}
	f.h = confighandler.NewHandler(testRegistry(t), f.store, f.crypto, lookup, opts...)

	n := 0
	confighandler.SetClock(f.h, func() time.Time { return testNow }, func() uuid.UUID {
		n++
		return uuid.FromStringOrNil(fmt.Sprintf("00000000-0000-4000-8000-%012d", n))
	})

	return f
}

// putSecretRow stores a secret row as an earlier version of the service would have.
func (f *fixture) putSecretRow(t *testing.T, key, plaintext string, revision uint32) {
	t.Helper()

	blob, ver, err := f.crypto.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	f.store.Put(&confighandler.Row{Key: key, ValueType: "secret", SecretCiphertext: blob, KeyVersion: ver, Revision: revision, TMCreate: testNow, TMUpdate: testNow})
}

// putLegacyAADRow stores a secret row sealed with associated data, as the earlier layout did. It
// is made with the standard library alone under the key of testKeyList(1, 1).
func (f *fixture) putLegacyAADRow(t *testing.T, key, plaintext string, revision uint32) {
	t.Helper()

	k := make([]byte, 32)
	for i := range k {
		k[i] = 1 + byte(i)
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	for i := range nonce {
		nonce[i] = 0x70 + byte(i)
	}
	blob := gcm.Seal(nonce, nonce, []byte(plaintext), []byte("svc_configs\x00"+key))
	f.store.Put(&confighandler.Row{Key: key, ValueType: "secret", SecretCiphertext: blob, KeyVersion: 1, Revision: revision, TMCreate: testNow, TMUpdate: testNow})
}

func (f *fixture) putValueRow(key, valueType, json string) {
	f.store.Put(&confighandler.Row{Key: key, ValueType: valueType, ValueJSON: []byte(json), Revision: 1, TMCreate: testNow, TMUpdate: testNow})
}

func Test_resolve_nonSecret(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fixture)
		key   string

		expectInt    int64
		expectSource confighandler.Source
		expectShadow bool
		expectErr    error
	}{
		{name: "default", key: "a.limit", expectInt: 10, expectSource: confighandler.SourceDefault},
		{name: "env", key: "a.limit", setup: func(f *fixture) { f.env["A_LIMIT"] = "25" }, expectInt: 25, expectSource: confighandler.SourceEnv},
		{name: "env empty is not set", key: "a.limit", setup: func(f *fixture) { f.env["A_LIMIT"] = "" }, expectInt: 10, expectSource: confighandler.SourceDefault},
		{name: "db wins over env", key: "a.limit", setup: func(f *fixture) {
			f.env["A_LIMIT"] = "25"
			f.putValueRow("a.limit", "int", "40")
		}, expectInt: 40, expectSource: confighandler.SourceDB, expectShadow: true},
		{name: "db without env", key: "a.limit", setup: func(f *fixture) { f.putValueRow("a.limit", "int", "40") }, expectInt: 40, expectSource: confighandler.SourceDB},
		{name: "env not a number", key: "a.limit", setup: func(f *fixture) { f.env["A_LIMIT"] = "many" }, expectSource: confighandler.SourceEnv, expectErr: confighandler.ErrInvalid},
		{name: "env out of range", key: "a.limit", setup: func(f *fixture) { f.env["A_LIMIT"] = "1000" }, expectSource: confighandler.SourceEnv, expectErr: confighandler.ErrInvalid},
		{name: "db wrong value type", key: "a.limit", setup: func(f *fixture) {
			f.env["A_LIMIT"] = "25"
			f.putValueRow("a.limit", "string", `"40"`)
		}, expectSource: confighandler.SourceDBInvalid, expectShadow: true, expectErr: confighandler.ErrInvalid},
		{name: "db json is not a number", key: "a.limit", setup: func(f *fixture) { f.putValueRow("a.limit", "int", `"forty"`) }, expectSource: confighandler.SourceDBInvalid, expectErr: confighandler.ErrInvalid},
		{name: "db json fraction", key: "a.limit", setup: func(f *fixture) { f.putValueRow("a.limit", "int", `4.5`) }, expectSource: confighandler.SourceDBInvalid, expectErr: confighandler.ErrInvalid},
		{name: "db json null", key: "a.limit", setup: func(f *fixture) { f.putValueRow("a.limit", "int", `null`) }, expectSource: confighandler.SourceDBInvalid, expectErr: confighandler.ErrInvalid},
		{name: "db out of range", key: "a.limit", setup: func(f *fixture) { f.putValueRow("a.limit", "int", `0`) }, expectSource: confighandler.SourceDBInvalid, expectErr: confighandler.ErrInvalid},
		{name: "db invalid never falls back to env", key: "a.limit", setup: func(f *fixture) {
			f.env["A_LIMIT"] = "25"
			f.putValueRow("a.limit", "int", `0`)
		}, expectSource: confighandler.SourceDBInvalid, expectShadow: true, expectErr: confighandler.ErrInvalid},
		{name: "store read error is an error", key: "a.limit", setup: func(f *fixture) { f.store.GetErr = errors.New("database is down") }, expectErr: errTestStore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			if tt.setup != nil {
				tt.setup(f)
			}

			got, res, err := f.h.Int(context.Background(), tt.key)
			if tt.expectErr != nil {
				if !matchesErr(err, tt.expectErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				if tt.expectErr != errTestStore {
					if res.Source != tt.expectSource || res.EnvShadowed != tt.expectShadow {
						t.Errorf("Wrong match. expect: %s shadow %v, got: %s shadow %v", tt.expectSource, tt.expectShadow, res.Source, res.EnvShadowed)
					}
				}
				if errors.Is(err, confighandler.ErrNotFound) {
					t.Errorf("Wrong match. a read error was reported as not found")
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got != tt.expectInt || res.Source != tt.expectSource || res.EnvShadowed != tt.expectShadow {
				t.Errorf("Wrong match. expect: %d %s %v, got: %d %s %v", tt.expectInt, tt.expectSource, tt.expectShadow, got, res.Source, res.EnvShadowed)
			}
		})
	}
}

var errTestStore = errors.New("store error marker")

// matchesErr reports whether err is the expected error; errTestStore stands for the injected store failure.
func matchesErr(err, expect error) bool {
	if err == nil {
		return false
	}
	if expect == errTestStore {
		return strings.Contains(err.Error(), "database is down")
	}

	return errors.Is(err, expect)
}

func Test_resolve_typedReads(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	d, res, err := f.h.Duration(ctx, "a.timeout")
	if err != nil || d != 5*time.Second || res.Source != confighandler.SourceDefault {
		t.Errorf("Wrong match. got: %v %s %v", d, res.Source, err)
	}
	f.env["A_TIMEOUT"] = "30s"
	if d, _, err = f.h.Duration(ctx, "a.timeout"); err != nil || d != 30*time.Second {
		t.Errorf("Wrong match. got: %v %v", d, err)
	}
	f.env["A_TIMEOUT"] = "100ms"
	if _, res, err = f.h.Duration(ctx, "a.timeout"); !errors.Is(err, confighandler.ErrInvalid) || res.Source != confighandler.SourceEnv {
		t.Errorf("Wrong match. got: %s %v", res.Source, err)
	}

	s, res, err := f.h.String(ctx, "a.mode")
	if err != nil || s != "x" || res.Source != confighandler.SourceDefault || res.Kind != confighandler.KindSelect || res.Type != confighandler.TypeEnum {
		t.Errorf("Wrong match. got: %q %+v %v", s, res, err)
	}
	f.env["A_MODE"] = "z"
	if _, _, err = f.h.String(ctx, "a.mode"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. an env value outside the enum expect: ErrInvalid, got: %v", err)
	}

	// no env, no default, no row: source none, empty value, no error.
	u, res, err := f.h.String(ctx, "a.url")
	if err != nil || u != "" || res.Source != confighandler.SourceNone {
		t.Errorf("Wrong match. got: %q %s %v", u, res.Source, err)
	}

	f.putValueRow("a.timeout", "duration", `"45s"`)
	if d, res, err = f.h.Duration(ctx, "a.timeout"); err != nil || d != 45*time.Second || res.Source != confighandler.SourceDB || !res.EnvShadowed {
		t.Errorf("Wrong match. got: %v %+v %v", d, res, err)
	}
}

func Test_resolve_secret(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		keyList string
		setup   func(f *fixture)

		expectValue  string
		expectSource confighandler.Source
		expectShadow bool
		expectErr    error
		expectHas    bool
	}{
		{name: "none", keyList: testKeyList(1, 1), expectSource: confighandler.SourceNone},
		{name: "env", keyList: testKeyList(1, 1), setup: func(f *fixture) { f.env["S_TOKEN"] = "env-token" }, expectValue: "env-token", expectSource: confighandler.SourceEnv, expectHas: true},
		{name: "env works without any master key", keyList: "", setup: func(f *fixture) { f.env["S_TOKEN"] = "env-token" }, expectValue: "env-token", expectSource: confighandler.SourceEnv, expectHas: true},
		{name: "db wins over env", keyList: testKeyList(1, 1), setup: func(f *fixture) {
			f.env["S_TOKEN"] = "env-token"
			f.putSecretRow(t, "s.token", "db-token", 3)
		}, expectValue: "db-token", expectSource: confighandler.SourceDB, expectShadow: true, expectHas: true},
		{name: "no master key never falls back to env", keyList: "", setup: func(f *fixture) {
			f.env["S_TOKEN"] = "env-token"
			// the row was written when a key existed
			g := newFixture(t, testKeyList(1, 1))
			g.putSecretRow(t, "s.token", "db-token", 1)
			row, _ := g.store.Get(ctx, "s.token")
			f.store.Put(row)
		}, expectSource: confighandler.SourceDBUndecryptable, expectShadow: true, expectErr: confighandler.ErrUndecryptable, expectHas: true},
		{name: "unknown key version never falls back to env", keyList: testKeyList(2, 1), setup: func(f *fixture) {
			f.env["S_TOKEN"] = "env-token"
			g := newFixture(t, testKeyList(1, 1))
			g.putSecretRow(t, "s.token", "db-token", 1)
			row, _ := g.store.Get(ctx, "s.token")
			f.store.Put(row)
		}, expectSource: confighandler.SourceDBUndecryptable, expectShadow: true, expectErr: confighandler.ErrUndecryptable, expectHas: true},
		{name: "damaged ciphertext", keyList: testKeyList(1, 1), setup: func(f *fixture) {
			f.env["S_TOKEN"] = "env-token"
			f.putSecretRow(t, "s.token", "db-token", 1)
			row, _ := f.store.Get(ctx, "s.token")
			row.SecretCiphertext[14] ^= 0x01
			f.store.Put(row)
		}, expectSource: confighandler.SourceDBUndecryptable, expectShadow: true, expectErr: confighandler.ErrUndecryptable, expectHas: true},
		{name: "ciphertext sealed with associated data by an earlier layout", keyList: testKeyList(1, 1), setup: func(f *fixture) {
			f.env["S_TOKEN"] = "env-token"
			f.putLegacyAADRow(t, "s.token", "db-token", 1)
		}, expectSource: confighandler.SourceDBUndecryptable, expectShadow: true, expectErr: confighandler.ErrUndecryptable, expectHas: true},
		{name: "secret row without ciphertext", keyList: testKeyList(1, 1), setup: func(f *fixture) {
			f.store.Put(&confighandler.Row{Key: "s.token", ValueType: "secret", KeyVersion: 1, Revision: 1})
		}, expectSource: confighandler.SourceDBInvalid, expectErr: confighandler.ErrInvalid, expectHas: true},
		{name: "secret row with the wrong value type", keyList: testKeyList(1, 1), setup: func(f *fixture) {
			f.putValueRow("s.token", "string", `"plain"`)
		}, expectSource: confighandler.SourceDBInvalid, expectErr: confighandler.ErrInvalid},
		{name: "store read error", keyList: testKeyList(1, 1), setup: func(f *fixture) {
			f.env["S_TOKEN"] = "env-token"
			f.store.GetErr = errors.New("database is down")
		}, expectErr: errTestStore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.keyList)
			if tt.setup != nil {
				tt.setup(f)
			}

			got, res, err := f.h.Secret(ctx, "s.token")
			if tt.expectErr != nil {
				if !matchesErr(err, tt.expectErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				if got.Reveal() != "" {
					t.Errorf("Wrong match. an error returned a value")
				}
				if tt.expectErr != errTestStore {
					if res.Source != tt.expectSource || res.EnvShadowed != tt.expectShadow || res.HasSecret != tt.expectHas {
						t.Errorf("Wrong match. expect: %s %v %v, got: %+v", tt.expectSource, tt.expectShadow, tt.expectHas, res)
					}
				}
				// an undecryptable error carries the library error and never any text of the value.
				if strings.Contains(err.Error(), "db-token") || strings.Contains(err.Error(), "env-token") {
					t.Errorf("Wrong match. the error contains a value: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got.Reveal() != tt.expectValue || res.Source != tt.expectSource || res.EnvShadowed != tt.expectShadow || res.HasSecret != tt.expectHas {
				t.Errorf("Wrong match. expect: %q %s %v %v, got: %q %+v", tt.expectValue, tt.expectSource, tt.expectShadow, tt.expectHas, got.Reveal(), res)
			}
		})
	}
}

func Test_resolve_undecryptableWrapsLibraryError(t *testing.T) {
	ctx := context.Background()

	g := newFixture(t, testKeyList(1, 1))
	g.putSecretRow(t, "s.token", "v", 1)
	row, _ := g.store.Get(ctx, "s.token")

	noKey := newFixture(t, "")
	noKey.store.Put(row)
	_, _, err := noKey.h.Secret(ctx, "s.token")
	if !errors.Is(err, confighandler.ErrUndecryptable) || !errors.Is(err, secretcrypto.ErrKeyNotConfigured) {
		t.Errorf("Wrong match. no key expect both sentinels, got: %v", err)
	}

	otherVersion := newFixture(t, testKeyList(2, 1))
	otherVersion.store.Put(row)
	_, _, err = otherVersion.h.Secret(ctx, "s.token")
	if !errors.Is(err, confighandler.ErrUndecryptable) || !errors.Is(err, secretcrypto.ErrDecrypt) {
		t.Errorf("Wrong match. unknown version expect both sentinels, got: %v", err)
	}
}

func Test_resolve_atomicGroup(t *testing.T) {
	ctx := context.Background()

	t.Run("anchor row present: members read from the database or default only", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.env["P_API_KEY"] = "env-key"
		f.env["P_MODEL"] = "env-model"
		f.env["P_ENABLED"] = "false"
		f.putSecretRow(t, "p.api_key", "db-key", 1)

		key, res, err := f.h.Secret(ctx, "p.api_key")
		if err != nil || key.Reveal() != "db-key" || res.Source != confighandler.SourceDB {
			t.Errorf("Wrong match. got: %q %+v %v", key.Reveal(), res, err)
		}

		// the member has no row: the default, never the env value.
		model, res, err := f.h.String(ctx, "p.model")
		if err != nil || model != "model-default" || res.Source != confighandler.SourceDefault {
			t.Errorf("Wrong match. got: %q %+v %v", model, res, err)
		}
		enabled, res, err := f.h.Bool(ctx, "p.enabled")
		if err != nil || !enabled || res.Source != confighandler.SourceDefault {
			t.Errorf("Wrong match. got: %v %+v %v", enabled, res, err)
		}

		f.putValueRow("p.model", "string", `"db-model"`)
		if model, res, err = f.h.String(ctx, "p.model"); err != nil || model != "db-model" || res.Source != confighandler.SourceDB || !res.EnvShadowed {
			t.Errorf("Wrong match. got: %q %+v %v", model, res, err)
		}
	})

	t.Run("anchor row absent: members read from env or default only, a stray member row is ignored", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.env["P_API_KEY"] = "env-key"
		f.env["P_MODEL"] = "env-model"
		f.putValueRow("p.model", "string", `"stray-db-model"`)

		model, res, err := f.h.String(ctx, "p.model")
		if err != nil || model != "env-model" || res.Source != confighandler.SourceEnv {
			t.Errorf("Wrong match. got: %q %+v %v", model, res, err)
		}
		key, res, err := f.h.Secret(ctx, "p.api_key")
		if err != nil || key.Reveal() != "env-key" || res.Source != confighandler.SourceEnv {
			t.Errorf("Wrong match. got: %q %+v %v", key.Reveal(), res, err)
		}

		delete(f.env, "P_MODEL")
		if model, res, err = f.h.String(ctx, "p.model"); err != nil || model != "model-default" || res.Source != confighandler.SourceDefault {
			t.Errorf("Wrong match. got: %q %+v %v", model, res, err)
		}
	})

	t.Run("undecryptable anchor makes the whole group unusable", func(t *testing.T) {
		f := newFixture(t, testKeyList(2, 1))
		g := newFixture(t, testKeyList(1, 1))
		g.putSecretRow(t, "p.api_key", "db-key", 1)
		row, _ := g.store.Get(ctx, "p.api_key")
		f.store.Put(row)
		f.env["P_MODEL"] = "env-model"

		if _, res, err := f.h.Secret(ctx, "p.api_key"); !errors.Is(err, confighandler.ErrUndecryptable) || res.Source != confighandler.SourceDBUndecryptable {
			t.Errorf("Wrong match. got: %+v %v", res, err)
		}
		model, res, err := f.h.String(ctx, "p.model")
		if err != nil || model != "model-default" || res.Source != confighandler.SourceDefault {
			t.Errorf("Wrong match. the member must not take the env value, got: %q %+v %v", model, res, err)
		}
	})

	t.Run("a store error on the anchor is an error", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.store.GetErr = errors.New("database is down")
		if _, _, err := f.h.String(ctx, "p.model"); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_wrongKindAndUnknown(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	if _, _, err := f.h.String(ctx, "s.token"); !errors.Is(err, confighandler.ErrWrongKind) {
		t.Errorf("Wrong match. String on a secret expect: ErrWrongKind, got: %v", err)
	}
	if _, _, err := f.h.Secret(ctx, "a.limit"); !errors.Is(err, confighandler.ErrWrongKind) {
		t.Errorf("Wrong match. Secret on a param expect: ErrWrongKind, got: %v", err)
	}
	if _, _, err := f.h.Int(ctx, "a.mode"); !errors.Is(err, confighandler.ErrWrongKind) {
		t.Errorf("Wrong match. Int on an enum expect: ErrWrongKind, got: %v", err)
	}
	if _, _, err := f.h.Bool(ctx, "a.limit"); !errors.Is(err, confighandler.ErrWrongKind) {
		t.Errorf("Wrong match. Bool on an int expect: ErrWrongKind, got: %v", err)
	}
	if _, _, err := f.h.Duration(ctx, "a.limit"); !errors.Is(err, confighandler.ErrWrongKind) {
		t.Errorf("Wrong match. Duration on an int expect: ErrWrongKind, got: %v", err)
	}
	if _, _, err := f.h.String(ctx, "nothing"); !errors.Is(err, confighandler.ErrUnknownKey) {
		t.Errorf("Wrong match. expect: ErrUnknownKey, got: %v", err)
	}
	if _, _, err := f.h.Secret(ctx, "nothing"); !errors.Is(err, confighandler.ErrUnknownKey) {
		t.Errorf("Wrong match. expect: ErrUnknownKey, got: %v", err)
	}
	if _, _, err := f.h.Int(ctx, "nothing"); !errors.Is(err, confighandler.ErrUnknownKey) {
		t.Errorf("Wrong match. expect: ErrUnknownKey, got: %v", err)
	}
	if f.h.Registry() == nil {
		t.Errorf("Wrong match. the registry must be available")
	}
}

func Test_Describe(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	f.env["S_TOKEN"] = "env-token"
	f.putSecretRow(t, "p.api_key", "db-key", 4)
	f.putValueRow("a.limit", "int", "40")
	f.putValueRow("a.timeout", "duration", `"bad"`)
	f.putValueRow("unregistered.key", "string", `"x"`)
	f.env["A_LIMIT"] = "7"

	res, err := f.h.Describe(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	byKey := map[string]*confighandler.Resolved{}
	var order []string
	for _, r := range res {
		byKey[r.Key] = r
		order = append(order, r.Key)
	}
	expectOrder := []string{"p.api_key", "p.enabled", "p.model", "s.token", "a.limit", "a.timeout", "a.mode", "a.url"}
	if strings.Join(order, ",") != strings.Join(expectOrder, ",") {
		t.Errorf("Wrong match. expect: %v, got: %v", expectOrder, order)
	}
	if _, ok := byKey["unregistered.key"]; ok {
		t.Errorf("Wrong match. a row of an unregistered key was listed")
	}

	check := func(key string, source confighandler.Source, shadow, has bool) {
		t.Helper()
		r := byKey[key]
		if r == nil || r.Source != source || r.EnvShadowed != shadow || r.HasSecret != has {
			t.Errorf("Wrong match. %s expect: %s %v %v, got: %+v", key, source, shadow, has, r)
		}
	}
	check("p.api_key", confighandler.SourceDB, false, true)
	check("p.model", confighandler.SourceDefault, false, false)
	check("s.token", confighandler.SourceEnv, false, true)
	check("a.limit", confighandler.SourceDB, true, false)
	check("a.timeout", confighandler.SourceDBInvalid, false, false)
	check("a.mode", confighandler.SourceDefault, false, false)
	check("a.url", confighandler.SourceNone, false, false)

	if r := byKey["p.api_key"]; r.Revision != 4 || r.KeyVersion != 1 || r.TMUpdate == nil || !r.TMUpdate.Equal(testNow) {
		t.Errorf("Wrong match. metadata: %+v", r)
	}

	// no value is part of Resolved: its text form carries no value either.
	for _, r := range res {
		text := fmt.Sprintf("%+v", *r)
		for _, v := range []string{"db-key", "env-token"} {
			if strings.Contains(text, v) {
				t.Errorf("Wrong match. Describe leaks a value: %s", text)
			}
		}
	}

	f.store.ListErr = errors.New("database is down")
	if _, err := f.h.Describe(ctx); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_storeContract_memory(t *testing.T) {
	storetest.Contract(t,
		func(t *testing.T) confighandler.Store { return storetest.NewMemory() },
		func(t *testing.T, s confighandler.Store) int { return len(s.(*storetest.Memory).Histories()) },
	)
}
