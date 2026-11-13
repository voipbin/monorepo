package providerhandler

import (
	"context"
	"encoding/base64"
	"encoding/hex"
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

func newTestHandler(t *testing.T, db *fakeDB, keys string) *providerHandler {
	t.Helper()

	crypto, err := NewSecretCrypto(keys)
	if err != nil {
		t.Fatal(err)
	}

	h, ok := NewProviderHandler(nil, db, crypto).(*providerHandler)
	if !ok {
		t.Fatal("wrong type")
	}
	return h
}

// seedRow stores a row with an encrypted secret, sealed with the key list testKeys(1).
func seedRow(t *testing.T, db *fakeDB, name number.ProviderName, enabled bool, priority int, config map[string]string) {
	t.Helper()

	crypto, err := NewSecretCrypto(testKeys(1))
	if err != nil {
		t.Fatal(err)
	}
	ct, ver, err := crypto.Encrypt("db-token")
	if err != nil {
		t.Fatal(err)
	}

	db.rows[name] = &provider.Provider{
		ProviderName:     name,
		Enabled:          enabled,
		Priority:         priority,
		Config:           config,
		SecretCiphertext: ct,
		KeyVersion:       ver,
	}
}

// seedTelnyx stores an enabled Telnyx row with a secret and the connection id.
func seedTelnyx(t *testing.T, db *fakeDB) {
	t.Helper()
	seedRow(t, db, number.ProviderNameTelnyx, true, 10, map[string]string{configConnectionID: "db-conn"})
}

// legacyAADRow returns a row whose secret was sealed with the provider name as associated data by
// an earlier layout (the fixed vector of secret_test.go). It does not decrypt.
func legacyAADRow(t *testing.T, config map[string]string) *provider.Provider {
	t.Helper()

	ct, err := hex.DecodeString(legacyAADVectorTwoKeyA)
	if err != nil {
		t.Fatalf("Wrong match. bad vector: %v", err)
	}

	return &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 5, Config: config, SecretCiphertext: ct, KeyVersion: 2}
}

func Test_resolve(t *testing.T) {
	ctx := context.Background()

	// a row with a DB secret is built by the handler itself, so it is encrypted with the test keys
	cryptoV1, _ := NewSecretCrypto(testKeys(1))
	ct, ver, _ := cryptoV1.Encrypt("db-token")

	dbRow := func(enabled bool, config map[string]string) *provider.Provider {
		return &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: enabled, Priority: 5, Config: config, SecretCiphertext: ct, KeyVersion: ver}
	}

	tests := []struct {
		name string

		row  *provider.Provider
		keys string

		expectSource   provider.CredentialSource
		expectUsable   bool
		expectEnabled  bool
		expectPriority int
		expectSecret   string
		expectConn     string
	}{
		{
			name:         "no row: telnyx defaults, no credential",
			keys:         testKeys(1),
			expectSource: provider.CredentialSourceNone, expectUsable: false, expectEnabled: true, expectPriority: 10,
		},
		{
			name:         "row without secret has no credential and its config is ignored",
			row:          &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 3, Config: map[string]string{"connection_id": "row-conn"}},
			keys:         testKeys(1),
			expectSource: provider.CredentialSourceNone, expectUsable: false, expectEnabled: true, expectPriority: 3,
		},
		{
			name:         "row with secret uses the DB secret and the row config only",
			row:          dbRow(true, map[string]string{"connection_id": "row-conn"}),
			keys:         testKeys(1),
			expectSource: provider.CredentialSourceDB, expectUsable: true, expectEnabled: true, expectPriority: 5,
			expectSecret: "db-token", expectConn: "row-conn",
		},
		{
			name:         "row with secret and without the required config is not usable",
			row:          dbRow(true, map[string]string{}),
			keys:         testKeys(1),
			expectSource: provider.CredentialSourceDB, expectUsable: false, expectEnabled: true, expectPriority: 5,
			expectSecret: "db-token",
		},
		{
			name:         "disabled row is not usable",
			row:          dbRow(false, map[string]string{"connection_id": "row-conn"}),
			keys:         testKeys(1),
			expectSource: provider.CredentialSourceDB, expectUsable: false, expectEnabled: false, expectPriority: 5,
			expectSecret: "db-token", expectConn: "row-conn",
		},
		{
			name:         "undecryptable secret is unusable",
			row:          dbRow(true, map[string]string{"connection_id": "row-conn"}),
			keys:         testKeys(2),
			expectSource: provider.CredentialSourceDBUndecryptable, expectUsable: false, expectEnabled: true, expectPriority: 5,
		},
		{
			name:         "secret sealed with associated data by an earlier layout is undecryptable and is unusable",
			row:          legacyAADRow(t, map[string]string{"connection_id": "row-conn"}),
			keys:         fakeKeyList(1, 2),
			expectSource: provider.CredentialSourceDBUndecryptable, expectUsable: false, expectEnabled: true, expectPriority: 5,
		},
		{
			name:         "no key configured: a DB secret is undecryptable",
			row:          dbRow(true, map[string]string{"connection_id": "row-conn"}),
			keys:         "",
			expectSource: provider.CredentialSourceDBUndecryptable, expectUsable: false, expectEnabled: true, expectPriority: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newFakeDB()
			if tt.row != nil {
				db.rows[number.ProviderNameTelnyx] = tt.row
			}
			h := newTestHandler(t, db, tt.keys)

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
		h := newTestHandler(t, newFakeDB(), "")
		if _, err := h.resolve(ctx, "nope"); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})

	t.Run("a db read error is never treated as no row", func(t *testing.T) {
		db := newFakeDB()
		db.getErr = errTestDB
		h := newTestHandler(t, db, "")
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

	t.Run("a Telnyx row with a secret is the only candidate", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		h := newTestHandler(t, db, testKeys(1))

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", res, err)
		}
	})

	t.Run("without any row there is no candidate (no environment fallback)", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "")

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v, %v", res, err)
		}
	})

	t.Run("an enabled twilio row without a secret is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = &provider.Provider{ProviderName: number.ProviderNameTwilio, Enabled: true, Priority: 1, Config: map[string]string{configSID: testTwilioSID, configTrunkSID: "TK1"}}
		h := newTestHandler(t, db, testKeys(1))

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
		h := newTestHandler(t, db, "")

		if res, _ := h.PurchaseCandidates(ctx); len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v", res)
		}
	})
}

func Test_sortOrder(t *testing.T) {
	ctx := context.Background()

	// a second purchasable provider is registered to exercise the ordering
	db := newFakeDB()
	seedTelnyx(t, db)
	seedRow(t, db, "alpha", true, 10, nil)
	seedRow(t, db, "beta", true, 5, nil)
	h := newTestHandler(t, db, testKeys(1))

	h.registry["alpha"] = Descriptor{Factory: h.registry[number.ProviderNameTelnyx].Factory, Purchasable: true, DefaultEnabled: true, DefaultPriority: 10}
	h.registry["beta"] = Descriptor{Factory: h.registry[number.ProviderNameTelnyx].Factory, Purchasable: true, DefaultEnabled: true, DefaultPriority: 5}

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
		seedRow(t, db, number.ProviderNameTelnyx, false, 10, map[string]string{configConnectionID: "db-conn"})
		h := newTestHandler(t, db, testKeys(1))

		if p, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("a DB secret without the required config can still release and search", func(t *testing.T) {
		cryptoV1, _ := NewSecretCrypto(testKeys(1))
		ct, ver, _ := cryptoV1.Encrypt("db-token")
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, SecretCiphertext: ct, KeyVersion: ver, Config: map[string]string{}}
		h := newTestHandler(t, db, testKeys(1))

		if _, err := h.ForSearch(ctx); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("no credential", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "")

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if _, err := h.ForSearch(ctx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("a row without a secret has no credential", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Config: map[string]string{configConnectionID: "row-conn"}}
		h := newTestHandler(t, db, testKeys(1))

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("undecryptable secret", func(t *testing.T) {
		cryptoV1, _ := NewSecretCrypto(testKeys(1))
		ct, ver, _ := cryptoV1.Encrypt("db-token")
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, SecretCiphertext: ct, KeyVersion: ver}
		h := newTestHandler(t, db, testKeys(2))

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("unknown provider is not found", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "")

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
	db := newFakeDB()
	seedTelnyx(t, db)
	seedRow(t, db, "broken", true, 1, nil)
	h := newTestHandler(t, db, testKeys(1))

	h.registry["broken"] = Descriptor{
		Factory:         func(provider.Credentials) (Provider, error) { return nil, errors.New("factory failure") },
		Purchasable:     true,
		DefaultEnabled:  true,
		DefaultPriority: 1,
	}

	res, err := h.PurchaseCandidates(ctx)
	if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
		t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", res, err)
	}
}
