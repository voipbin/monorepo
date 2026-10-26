package providerhandler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/secretcrypto"
)

var (
	testActor = uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000a1")

	errTestDB = errors.New("db failure")
)

func testKeys(versions ...int) string {
	items := []string{}
	for _, v := range versions {
		k := make([]byte, 32)
		for i := range k {
			k[i] = byte(v)
		}
		items = append(items, fmt.Sprintf("%d:%s", v, base64.StdEncoding.EncodeToString(k)))
	}
	return strings.Join(items, ",")
}

// fakeDB keeps the provider rows in memory. Only the provider methods are implemented.
type fakeDB struct {
	dbhandler.DBHandler

	rows map[number.ProviderName]*provider.Provider

	getErr    error
	ensureErr error
	updateErr error

	// updates records the columns of every update
	updates [][]provider.Field
}

func newFakeDB() *fakeDB {
	return &fakeDB{rows: map[number.ProviderName]*provider.Provider{}}
}

func (f *fakeDB) NumberProviderGet(_ context.Context, name number.ProviderName) (*provider.Provider, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	row, ok := f.rows[name]
	if !ok {
		return nil, dbhandler.ErrNotFound
	}
	c := *row
	return &c, nil
}

func (f *fakeDB) NumberProviderEnsure(_ context.Context, p *provider.Provider) error {
	if f.ensureErr != nil {
		return f.ensureErr
	}
	if _, ok := f.rows[p.ProviderName]; ok {
		return nil
	}
	c := *p
	f.rows[p.ProviderName] = &c
	return nil
}

func (f *fakeDB) NumberProviderUpdate(_ context.Context, name number.ProviderName, fields map[provider.Field]any) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	row, ok := f.rows[name]
	if !ok {
		return dbhandler.ErrNotFound
	}

	cols := []provider.Field{}
	now := time.Now()
	row.TMUpdate = &now
	for k, v := range fields {
		cols = append(cols, k)
		switch k {
		case provider.FieldEnabled:
			row.Enabled = v.(bool)
		case provider.FieldPriority:
			row.Priority = v.(int)
		case provider.FieldConfig:
			row.Config = v.(map[string]string)
		case provider.FieldSecretCiphertext:
			row.SecretCiphertext = v.([]byte)
		case provider.FieldKeyVersion:
			row.KeyVersion = v.(int)
		case provider.FieldUpdatedBy:
			a := v.(uuid.UUID)
			row.UpdatedBy = &a
		case provider.FieldSecretUpdatedBy:
			a := v.(uuid.UUID)
			row.SecretUpdatedBy = &a
		case provider.FieldTMSecretUpdate:
			tm := v.(*time.Time)
			row.TMSecretUpdate = tm
		}
	}
	f.updates = append(f.updates, cols)
	return nil
}

func newTestHandler(t *testing.T, db *fakeDB, keys string, env map[number.ProviderName]provider.Credentials) *providerHandler {
	t.Helper()

	crypto, err := secretcrypto.New(keys)
	if err != nil {
		t.Fatal(err)
	}

	h, ok := NewProviderHandler(nil, db, crypto, env).(*providerHandler)
	if !ok {
		t.Fatal("wrong type")
	}
	return h
}

func envTelnyx() map[number.ProviderName]provider.Credentials {
	return NewEnvCredentials("env-conn", "env-prof", "env-token", "", "")
}

func Test_resolve(t *testing.T) {
	ctx := context.Background()

	// a row with a DB secret is built by the handler itself, so it is encrypted with the test keys
	cryptoV1, _ := secretcrypto.New(testKeys(1))
	ct, ver, _ := cryptoV1.Encrypt("db-token", "telnyx")

	dbRow := func(enabled bool, config map[string]string) *provider.Provider {
		return &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: enabled, Priority: 5, Config: config, SecretCiphertext: ct, KeyVersion: ver}
	}

	tests := []struct {
		name string

		row  *provider.Provider
		keys string
		env  map[number.ProviderName]provider.Credentials

		expectSource   provider.CredentialSource
		expectUsable   bool
		expectEnabled  bool
		expectPriority int
		expectSecret   string
		expectConn     string
	}{
		{
			name: "no row and env token: telnyx defaults, env source",
			keys: testKeys(1), env: envTelnyx(),
			expectSource: provider.CredentialSourceEnv, expectUsable: true, expectEnabled: true, expectPriority: 10,
			expectSecret: "env-token", expectConn: "env-conn",
		},
		{
			name: "no row and no env token: none",
			keys: testKeys(1), env: NewEnvCredentials("", "", "", "", ""),
			expectSource: provider.CredentialSourceNone, expectUsable: false, expectEnabled: true, expectPriority: 10,
		},
		{
			name: "env source needs only the token, not the connection id",
			keys: testKeys(1), env: NewEnvCredentials("", "", "env-token", "", ""),
			expectSource: provider.CredentialSourceEnv, expectUsable: true, expectEnabled: true, expectPriority: 10,
			expectSecret: "env-token",
		},
		{
			name: "row without secret uses env credentials and ignores the row config",
			row:  &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 3, Config: map[string]string{"connection_id": "row-conn"}},
			keys: testKeys(1), env: envTelnyx(),
			expectSource: provider.CredentialSourceEnv, expectUsable: true, expectEnabled: true, expectPriority: 3,
			expectSecret: "env-token", expectConn: "env-conn",
		},
		{
			name: "row with secret uses the DB secret and the row config only",
			row:  dbRow(true, map[string]string{"connection_id": "row-conn"}),
			keys: testKeys(1), env: envTelnyx(),
			expectSource: provider.CredentialSourceDB, expectUsable: true, expectEnabled: true, expectPriority: 5,
			expectSecret: "db-token", expectConn: "row-conn",
		},
		{
			name: "row with secret and without the required config is not usable and does not use env",
			row:  dbRow(true, map[string]string{}),
			keys: testKeys(1), env: envTelnyx(),
			expectSource: provider.CredentialSourceDB, expectUsable: false, expectEnabled: true, expectPriority: 5,
			expectSecret: "db-token",
		},
		{
			name: "disabled row is not usable",
			row:  dbRow(false, map[string]string{"connection_id": "row-conn"}),
			keys: testKeys(1), env: envTelnyx(),
			expectSource: provider.CredentialSourceDB, expectUsable: false, expectEnabled: false, expectPriority: 5,
			expectSecret: "db-token", expectConn: "row-conn",
		},
		{
			name: "undecryptable secret never falls back to env",
			row:  dbRow(true, map[string]string{"connection_id": "row-conn"}),
			keys: testKeys(2), env: envTelnyx(),
			expectSource: provider.CredentialSourceDBUndecryptable, expectUsable: false, expectEnabled: true, expectPriority: 5,
		},
		{
			name: "no key configured: a DB secret is undecryptable",
			row:  dbRow(true, map[string]string{"connection_id": "row-conn"}),
			keys: "", env: envTelnyx(),
			expectSource: provider.CredentialSourceDBUndecryptable, expectUsable: false, expectEnabled: true, expectPriority: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newFakeDB()
			if tt.row != nil {
				db.rows[number.ProviderNameTelnyx] = tt.row
			}
			h := newTestHandler(t, db, tt.keys, tt.env)

			r, err := h.resolve(ctx, number.ProviderNameTelnyx)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if r.source != tt.expectSource || r.usable != tt.expectUsable || r.enabled != tt.expectEnabled || r.priority != tt.expectPriority {
				t.Errorf("Wrong match. got source=%s usable=%v enabled=%v priority=%d", r.source, r.usable, r.enabled, r.priority)
			}
			if r.creds.Secret != tt.expectSecret || r.creds.Config["connection_id"] != tt.expectConn {
				t.Errorf("Wrong credentials. got secret-match=%v conn=%q", r.creds.Secret == tt.expectSecret, r.creds.Config["connection_id"])
			}
		})
	}
}

func Test_resolve_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown provider", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", nil)
		if _, err := h.resolve(ctx, "nope"); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})

	t.Run("a db read error is never treated as no row", func(t *testing.T) {
		db := newFakeDB()
		db.getErr = errTestDB
		h := newTestHandler(t, db, "", envTelnyx())
		if _, err := h.resolve(ctx, number.ProviderNameTelnyx); !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
		if _, err := h.PurchaseCandidates(ctx); !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})
}

func Test_PurchaseCandidates(t *testing.T) {
	ctx := context.Background()

	t.Run("deploy default: telnyx alone from the env credentials", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("c", "p", "t", testTwilioSID, "twilio-token"))

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", res, err)
		}
	})

	t.Run("without any credential there is no candidate", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("", "", "", "", ""))

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v, %v", res, err)
		}
	})

	t.Run("an enabled twilio with the environment credential only is not a candidate", func(t *testing.T) {
		// the environment has no trunk_sid, so a purchase could not route the calls of the number
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = &provider.Provider{ProviderName: number.ProviderNameTwilio, Enabled: true, Priority: 1}
		h := newTestHandler(t, db, "", NewEnvCredentials("c", "p", "t", testTwilioSID, "twilio-token"))

		res, _ := h.PurchaseCandidates(ctx)
		for _, c := range res {
			if c.Name == number.ProviderNameTwilio {
				t.Errorf("Wrong match. twilio must not be a purchase candidate")
			}
		}
	})

	t.Run("a disabled telnyx is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: false, Priority: 1}
		h := newTestHandler(t, db, "", envTelnyx())

		if res, _ := h.PurchaseCandidates(ctx); len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v", res)
		}
	})
}

func Test_sortOrder(t *testing.T) {
	ctx := context.Background()

	// a second purchasable provider is registered to exercise the ordering
	db := newFakeDB()
	h := newTestHandler(t, db, "", envTelnyx())

	h.registry["alpha"] = Descriptor{Factory: h.registry[number.ProviderNameTelnyx].Factory, Purchasable: true, DefaultEnabled: true, DefaultPriority: 10}
	h.registry["beta"] = Descriptor{Factory: h.registry[number.ProviderNameTelnyx].Factory, Purchasable: true, DefaultEnabled: true, DefaultPriority: 5}
	h.env["alpha"] = provider.Credentials{Secret: "a"}
	h.env["beta"] = provider.Credentials{Secret: "b"}

	res, err := h.PurchaseCandidates(ctx)
	if err != nil {
		t.Fatal(err)
	}

	got := []number.ProviderName{}
	for _, c := range res {
		got = append(got, c.Name)
	}
	// priority 5 first, then the two priority 10 providers by name
	expect := []number.ProviderName{"beta", "alpha", number.ProviderNameTelnyx}
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong order. expect: %v, got: %v", expect, got)
	}
}

func Test_ForRelease(t *testing.T) {
	ctx := context.Background()

	t.Run("a disabled provider with a secret can release", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: false}
		h := newTestHandler(t, db, "", envTelnyx())

		if p, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("a DB secret without the required config can still release and search", func(t *testing.T) {
		cryptoV1, _ := secretcrypto.New(testKeys(1))
		ct, ver, _ := cryptoV1.Encrypt("db-token", "telnyx")
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, SecretCiphertext: ct, KeyVersion: ver, Config: map[string]string{}}
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.ForSearch(ctx); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("no credential", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("", "", "", "", ""))

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("undecryptable secret", func(t *testing.T) {
		cryptoV1, _ := secretcrypto.New(testKeys(1))
		ct, ver, _ := cryptoV1.Encrypt("db-token", "telnyx")
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, SecretCiphertext: ct, KeyVersion: ver}
		h := newTestHandler(t, db, testKeys(2), envTelnyx())

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("unknown provider is not found", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", nil)

		if _, err := h.ForRelease(ctx, "nope"); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})
}

func Test_Credentials_redaction(t *testing.T) {
	c := provider.Credentials{Secret: "very-secret-value", Config: map[string]string{"connection_id": "x"}}

	for _, out := range []string{fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c), fmt.Sprintf("%-60s", c), fmt.Sprint(&c)} {
		if strings.Contains(out, "very-secret-value") {
			t.Errorf("A format verb prints the secret: %s", out)
		}
	}

	p := provider.Provider{ProviderName: "telnyx", SecretCiphertext: []byte("very-secret-ciphertext")}
	for _, out := range []string{fmt.Sprintf("%v", p), fmt.Sprintf("%+v", p), fmt.Sprintf("%#v", p), fmt.Sprint(&p)} {
		if strings.Contains(out, "very-secret-ciphertext") {
			t.Errorf("A format verb prints the ciphertext: %s", out)
		}
	}
}

func Test_PurchaseCandidates_factoryFailure(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t, newFakeDB(), "", envTelnyx())

	h.registry["broken"] = Descriptor{
		Factory:         func(provider.Credentials) (Provider, error) { return nil, errors.New("factory failure") },
		Purchasable:     true,
		DefaultEnabled:  true,
		DefaultPriority: 1,
	}
	h.env["broken"] = provider.Credentials{Secret: "b"}

	res, err := h.PurchaseCandidates(ctx)
	if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
		t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", res, err)
	}
}
