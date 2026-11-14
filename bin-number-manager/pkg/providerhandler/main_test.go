package providerhandler

import (
	"context"
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
		case provider.FieldUpdatedBy:
			a := v.(uuid.UUID)
			row.UpdatedBy = &a
		}
	}
	f.updates = append(f.updates, cols)
	return nil
}

func newTestHandler(t *testing.T, db *fakeDB) *providerHandler {
	t.Helper()

	h, ok := NewProviderHandler(nil, db).(*providerHandler)
	if !ok {
		t.Fatal("wrong type")
	}
	return h
}

// seedRow stores a row.
func seedRow(t *testing.T, db *fakeDB, name number.ProviderName, enabled bool, priority int, config map[string]string) {
	t.Helper()

	db.rows[name] = &provider.Provider{
		ProviderName: name,
		Enabled:      enabled,
		Priority:     priority,
		Config:       config,
	}
}

// seedTelnyx stores an enabled Telnyx row with the connection id.
func seedTelnyx(t *testing.T, db *fakeDB) {
	t.Helper()
	seedRow(t, db, number.ProviderNameTelnyx, true, 10, map[string]string{configConnectionID: "db-conn"})
}

func Test_resolve(t *testing.T) {
	ctx := context.Background()

	row := func(enabled bool, config map[string]string) *provider.Provider {
		return &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: enabled, Priority: 5, Config: config}
	}

	tests := []struct {
		name string

		row *provider.Provider
		env map[string]string

		expectSource   provider.CredentialSource
		expectUsable   bool
		expectEnabled  bool
		expectPriority int
		expectSecret   string
		expectConn     string
	}{
		{
			name:         "no row and no variable: telnyx defaults, no credential",
			expectSource: provider.CredentialSourceNone, expectUsable: false, expectEnabled: true, expectPriority: 10,
		},
		{
			name:         "row without a variable has no credential and its config is ignored",
			row:          &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 3, Config: map[string]string{"connection_id": "row-conn"}},
			expectSource: provider.CredentialSourceNone, expectUsable: false, expectEnabled: true, expectPriority: 3,
		},
		{
			name:         "a row with the variable uses the variable and the row config",
			row:          row(true, map[string]string{"connection_id": "row-conn"}),
			env:          map[string]string{envTelnyxAPIKey: "env-token"},
			expectSource: provider.CredentialSourceEnv, expectUsable: true, expectEnabled: true, expectPriority: 5,
			expectSecret: "env-token", expectConn: "row-conn",
		},
		{
			name:         "the variable without the required config is not usable",
			row:          row(true, map[string]string{}),
			env:          map[string]string{envTelnyxAPIKey: "env-token"},
			expectSource: provider.CredentialSourceEnv, expectUsable: false, expectEnabled: true, expectPriority: 5,
			expectSecret: "env-token",
		},
		{
			name:         "disabled row is not usable",
			row:          row(false, map[string]string{"connection_id": "row-conn"}),
			env:          map[string]string{envTelnyxAPIKey: "env-token"},
			expectSource: provider.CredentialSourceEnv, expectUsable: false, expectEnabled: false, expectPriority: 5,
			expectSecret: "env-token", expectConn: "row-conn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCredentialEnv(t, tt.env)

			db := newFakeDB()
			if tt.row != nil {
				db.rows[number.ProviderNameTelnyx] = tt.row
			}
			h := newTestHandler(t, db)

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
		h := newTestHandler(t, newFakeDB())
		if _, err := h.resolve(ctx, "nope"); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})

	t.Run("a db read error is never treated as no row", func(t *testing.T) {
		db := newFakeDB()
		db.getErr = errTestDB
		h := newTestHandler(t, db)
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
	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: "env-token"})

	t.Run("a Telnyx row with a secret is the only candidate", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		h := newTestHandler(t, db)

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", res, err)
		}
	})

	t.Run("without any row and without the variable there is no candidate", func(t *testing.T) {
		setCredentialEnv(t, nil)
		h := newTestHandler(t, newFakeDB())

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v, %v", res, err)
		}
	})

	t.Run("an enabled twilio row without the variable is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = &provider.Provider{ProviderName: number.ProviderNameTwilio, Enabled: true, Priority: 1, Config: map[string]string{configSID: testTwilioSID, configTrunkSID: "TK1"}}
		h := newTestHandler(t, db)

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
		h := newTestHandler(t, db)

		if res, _ := h.PurchaseCandidates(ctx); len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v", res)
		}
	})
}

func Test_sortOrder(t *testing.T) {
	ctx := context.Background()

	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: "env-token"})

	// a second purchasable provider is registered to exercise the ordering
	db := newFakeDB()
	seedTelnyx(t, db)
	seedRow(t, db, "alpha", true, 10, nil)
	seedRow(t, db, "beta", true, 5, nil)
	h := newTestHandler(t, db)

	h.registry["alpha"] = Descriptor{Factory: h.registry[number.ProviderNameTelnyx].Factory, EnvSecret: envTelnyxAPIKey, Purchasable: true, DefaultEnabled: true, DefaultPriority: 10}
	h.registry["beta"] = Descriptor{Factory: h.registry[number.ProviderNameTelnyx].Factory, EnvSecret: envTelnyxAPIKey, Purchasable: true, DefaultEnabled: true, DefaultPriority: 5}

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
	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: "env-token"})

	t.Run("a disabled provider with the variable can release", func(t *testing.T) {
		db := newFakeDB()
		seedRow(t, db, number.ProviderNameTelnyx, false, 10, map[string]string{configConnectionID: "db-conn"})
		h := newTestHandler(t, db)

		if p, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("the variable without the required config can still release and search", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Config: map[string]string{}}
		h := newTestHandler(t, db)

		if _, err := h.ForSearch(ctx); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("no credential", func(t *testing.T) {
		setCredentialEnv(t, nil)
		h := newTestHandler(t, newFakeDB())

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if _, err := h.ForSearch(ctx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("a row without the variable has no credential", func(t *testing.T) {
		setCredentialEnv(t, nil)
		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Config: map[string]string{configConnectionID: "row-conn"}}
		h := newTestHandler(t, db)

		if _, err := h.ForRelease(ctx, number.ProviderNameTelnyx); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("unknown provider is not found", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB())

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

}

func Test_PurchaseCandidates_factoryFailure(t *testing.T) {
	ctx := context.Background()
	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: "env-token"})
	db := newFakeDB()
	seedTelnyx(t, db)
	seedRow(t, db, "broken", true, 1, nil)
	h := newTestHandler(t, db)

	h.registry["broken"] = Descriptor{
		Factory:         func(provider.Credentials) (Provider, error) { return nil, errors.New("factory failure") },
		EnvSecret:       envTelnyxAPIKey,
		Purchasable:     true,
		DefaultEnabled:  true,
		DefaultPriority: 1,
	}

	res, err := h.PurchaseCandidates(ctx)
	if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
		t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", res, err)
	}
}
