package providerhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
)

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

func Test_List_Get_defaults(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())

	list, err := h.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("Wrong match. expect: two providers, got: %v, %v", list, err)
	}
	if list[0].ProviderName != number.ProviderNameTelnyx || list[1].ProviderName != number.ProviderNameTwilio {
		t.Errorf("Wrong order. got: %s, %s", list[0].ProviderName, list[1].ProviderName)
	}

	tel := list[0]
	if !tel.Enabled || tel.Priority != 10 || tel.HasSecret || tel.CredentialSource != provider.CredentialSourceEnv || !tel.Usable {
		t.Errorf("Wrong telnyx defaults. got: %+v", tel)
	}
	if list[1].Enabled || list[1].Priority != 20 || list[1].Usable {
		t.Errorf("Wrong twilio defaults. got: %+v", list[1])
	}

	if _, err := h.Get(ctx, "nope"); !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: not found, got: %v", err)
	}
}

func Test_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("a priority-only update on a missing row keeps the registry default for enabled", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, intPtr(50), nil)
		if err != nil {
			t.Fatal(err)
		}
		if !info.Enabled || info.Priority != 50 || info.UpdatedBy == nil || *info.UpdatedBy != testActor || info.TMUpdate == nil {
			t.Errorf("Wrong match. got: %+v", info)
		}
		// only the changed columns (and the audit columns) are written
		if len(db.updates) != 1 || len(db.updates[0]) != 2 {
			t.Errorf("Wrong columns. got: %v", db.updates)
		}
	})

	t.Run("config merge: a key is set, another is kept, an empty value deletes", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1", "profile_id": "p1"}); err != nil {
			t.Fatal(err)
		}
		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"profile_id": ""})
		if err != nil {
			t.Fatal(err)
		}
		if info.Config["connection_id"] != "c1" || len(info.Config) != 1 {
			t.Errorf("Wrong config. got: %v", info.Config)
		}
	})

	t.Run("validation", func(t *testing.T) {
		tests := []struct {
			name     string
			p        number.ProviderName
			actor    uuid.UUID
			enabled  *bool
			priority *int
			config   map[string]string
		}{
			{"nil actor", number.ProviderNameTelnyx, uuid.Nil, boolPtr(true), nil, nil},
			{"nothing to update", number.ProviderNameTelnyx, testActor, nil, nil, nil},
			{"negative priority", number.ProviderNameTelnyx, testActor, nil, intPtr(-1), nil},
			{"too large priority", number.ProviderNameTelnyx, testActor, nil, intPtr(maxPriority + 1), nil},
			{"config key not allowed", number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"token": "x"}},
			{"config value too long", number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": strings.Repeat("a", maxConfigValueLen+1)}},
			{"too many config keys", number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"a": "1", "b": "1", "c": "1", "d": "1", "e": "1", "f": "1", "g": "1", "h": "1", "i": "1"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				db := newFakeDB()
				h := newTestHandler(t, db, testKeys(1), envTelnyx())

				_, err := h.Update(ctx, tt.p, tt.actor, tt.enabled, tt.priority, tt.config)
				if !errors.Is(err, provider.ErrInvalidArgument) {
					t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
				}
				if len(db.rows) != 0 {
					t.Errorf("A rejected request must not create a row")
				}
			})
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), nil)
		if _, err := h.Update(ctx, "nope", testActor, boolPtr(true), nil, nil); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})

	t.Run("deleting a required key of a DB-secret row is rejected", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"}); err != nil {
			t.Fatal(err)
		}
		if _, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "db-token"); err != nil {
			t.Fatal(err)
		}

		_, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": ""})
		if !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
		}
		if db.rows[number.ProviderNameTelnyx].Config["connection_id"] != "c1" {
			t.Errorf("The rejected update changed the config")
		}
	})

	t.Run("a db error on the read does not create defaults", func(t *testing.T) {
		db := newFakeDB()
		db.updateErr = errTestDB
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil); !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})

	t.Run("ensure failure", func(t *testing.T) {
		db := newFakeDB()
		db.ensureErr = errTestDB
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil); !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})
}

func Test_UpdateSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("normal flow: config first, then the secret; the response carries no secret", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1, 2), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"}); err != nil {
			t.Fatal(err)
		}
		info, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake-db-token")
		if err != nil {
			t.Fatal(err)
		}

		if !info.HasSecret || info.KeyVersion != 2 || info.CredentialSource != provider.CredentialSourceDB || !info.Usable {
			t.Errorf("Wrong info. got: %+v", info)
		}
		if info.SecretUpdatedBy == nil || *info.SecretUpdatedBy != testActor || info.TMSecretUpdate == nil {
			t.Errorf("Wrong audit. got: %+v", info)
		}

		// the stored value is a ciphertext, never the plaintext
		row := db.rows[number.ProviderNameTelnyx]
		if strings.Contains(string(row.SecretCiphertext), "fake-db-token") || len(row.SecretCiphertext) == 0 {
			t.Errorf("The secret is not stored as a ciphertext")
		}

		// resolving gives back the secret from the DB, not the env one
		r, _ := h.resolve(ctx, number.ProviderNameTelnyx)
		if r.creds.Secret != "fake-db-token" || r.source != provider.CredentialSourceDB {
			t.Errorf("Wrong resolution. source: %s", r.source)
		}
	})

	t.Run("no key configured rejects the secret and creates no row", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, "", envTelnyx())

		_, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake")
		if !errors.Is(err, provider.ErrSecretKeyNotConfigured) {
			t.Errorf("Wrong match. expect: key not configured, got: %v", err)
		}
		if len(db.rows) != 0 {
			t.Errorf("A rejected request must not create a row")
		}
	})

	t.Run("missing required config is rejected, also for a disabled provider", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil); err != nil {
			t.Fatal(err)
		}
		_, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake")
		if !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
		}
		if len(db.rows[number.ProviderNameTelnyx].SecretCiphertext) != 0 {
			t.Errorf("A rejected request stored a secret")
		}
	})

	t.Run("a provider without required config accepts the secret directly", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), NewEnvCredentials("", "", "", testTwilioSID, ""))
		addPlain(h)

		info, err := h.UpdateSecret(ctx, plainProvider, testActor, "fake-twilio")
		if err != nil || !info.HasSecret || info.Enabled {
			t.Errorf("Wrong match. got: %+v, %v", info, err)
		}
	})

	t.Run("validation", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())

		if _, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, uuid.Nil, "x"); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("nil actor: expect invalid argument, got: %v", err)
		}
		if _, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, ""); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("empty secret: expect invalid argument, got: %v", err)
		}
		if _, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, strings.Repeat("a", 1025)); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("long secret: expect invalid argument, got: %v", err)
		}
		if _, err := h.UpdateSecret(ctx, "nope", testActor, "x"); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("unknown provider: expect not found, got: %v", err)
		}
	})

	t.Run("errors do not contain the secret", func(t *testing.T) {
		db := newFakeDB()
		db.updateErr = errors.New("db failure")
		h := newTestHandler(t, db, testKeys(1), envTelnyx())
		_, _ = h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"})
		db.updateErr = nil
		_, _ = h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"})

		db.updateErr = errors.New("db failure")
		_, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake-leak-check")
		if err == nil || strings.Contains(err.Error(), "fake-leak-check") {
			t.Errorf("Wrong error: %v", err)
		}
	})
}

func Test_DeleteSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("deleting returns the provider to the env credentials", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		_, _ = h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"})
		if _, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake-db-token"); err != nil {
			t.Fatal(err)
		}

		info, err := h.DeleteSecret(ctx, number.ProviderNameTelnyx, testActor)
		if err != nil {
			t.Fatal(err)
		}
		if info.HasSecret || info.KeyVersion != 0 || info.CredentialSource != provider.CredentialSourceEnv || !info.Usable {
			t.Errorf("Wrong info. got: %+v", info)
		}
	})

	t.Run("an undecryptable secret can be deleted to recover", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())
		_, _ = h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"})
		_, _ = h.UpdateSecret(ctx, number.ProviderNameTelnyx, testActor, "fake-db-token")

		// the key of the stored secret is gone
		h2 := newTestHandler(t, db, testKeys(2), envTelnyx())
		if r, _ := h2.resolve(ctx, number.ProviderNameTelnyx); r.source != provider.CredentialSourceDBUndecryptable {
			t.Fatalf("expect an undecryptable secret, got: %s", r.source)
		}

		info, err := h2.DeleteSecret(ctx, number.ProviderNameTelnyx, testActor)
		if err != nil || info.CredentialSource != provider.CredentialSourceEnv || !info.Usable {
			t.Errorf("Wrong match. got: %+v, %v", info, err)
		}
	})

	t.Run("no row is not found", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())
		if _, err := h.DeleteSecret(ctx, number.ProviderNameTelnyx, testActor); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})

	t.Run("validation and failure", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.DeleteSecret(ctx, "nope", testActor); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("unknown provider: expect not found, got: %v", err)
		}
		if _, err := h.DeleteSecret(ctx, number.ProviderNameTelnyx, uuid.Nil); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("nil actor: expect invalid argument, got: %v", err)
		}

		db.updateErr = errTestDB
		if _, err := h.DeleteSecret(ctx, number.ProviderNameTelnyx, testActor); !errors.Is(err, errTestDB) {
			t.Errorf("db failure: expect db failure, got: %v", err)
		}
	})
}

func Test_Update_boundaries(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		priority int
		ok       bool
	}{{0, true}, {1, true}, {maxPriority, true}, {maxPriority + 1, false}, {-1, false}} {
		h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())
		_, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, intPtr(tt.priority), nil)
		if (err == nil) != tt.ok {
			t.Errorf("priority %d: expect ok=%v, got: %v", tt.priority, tt.ok, err)
		}
	}

	t.Run("an update of one field keeps the registry defaults of the others on a new row", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil)
		if err != nil || info.Enabled || info.Priority != 10 {
			t.Errorf("Wrong match. expect: disabled with priority 10, got: %+v, %v", info, err)
		}
	})

	t.Run("the number of config keys is limited at eight", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())
		keys := []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
		d := h.registry[number.ProviderNameTelnyx]
		d.AllowedConfig = keys
		h.registry[number.ProviderNameTelnyx] = d

		eight := map[string]string{}
		for _, k := range keys[:8] {
			eight[k] = "v"
		}
		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, eight); err != nil {
			t.Errorf("eight keys must be accepted, got: %v", err)
		}

		nine := map[string]string{}
		for _, k := range keys {
			nine[k] = "v"
		}
		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, nine); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("nine keys must be rejected, got: %v", err)
		}
	})

	t.Run("the secret length is limited at 1024 bytes", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), NewEnvCredentials("", "", "", testTwilioSID, ""))
		addPlain(h)

		if _, err := h.UpdateSecret(ctx, plainProvider, testActor, strings.Repeat("a", 1024)); err != nil {
			t.Errorf("1024 bytes must be accepted, got: %v", err)
		}
		if _, err := h.UpdateSecret(ctx, plainProvider, testActor, strings.Repeat("a", 1025)); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("1025 bytes must be rejected, got: %v", err)
		}
	})
}

func Test_DeleteSecret_audit(t *testing.T) {
	ctx := context.Background()
	db := newFakeDB()
	h := newTestHandler(t, db, testKeys(1), envTelnyx())

	_, _ = h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"})
	if _, err := h.UpdateSecret(ctx, number.ProviderNameTelnyx, uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000b2"), "fake"); err != nil {
		t.Fatal(err)
	}

	info, err := h.DeleteSecret(ctx, number.ProviderNameTelnyx, testActor)
	if err != nil {
		t.Fatal(err)
	}
	if info.SecretUpdatedBy == nil || *info.SecretUpdatedBy != testActor || info.TMSecretUpdate == nil {
		t.Errorf("The deletion must record who deleted the secret and when. got: %+v", info)
	}
}

func Test_UpdateSecret_trimsWhitespace(t *testing.T) {
	ctx := context.Background()
	db := newFakeDB()
	h := newTestHandler(t, db, testKeys(1), NewEnvCredentials("", "", "", testTwilioSID, ""))
	addPlain(h)

	if _, err := h.UpdateSecret(ctx, plainProvider, testActor, "  fake-token\n"); err != nil {
		t.Fatal(err)
	}
	r, _ := h.resolve(ctx, plainProvider)
	if r.creds.Secret != "fake-token" {
		t.Errorf("The stored secret keeps the surrounding whitespace")
	}

	if _, err := h.UpdateSecret(ctx, plainProvider, testActor, " \n\t "); !errors.Is(err, provider.ErrInvalidArgument) {
		t.Errorf("a blank secret must be rejected, got: %v", err)
	}
}

// plainProvider is a test only provider without any required config.
const plainProvider = number.ProviderName("plain")

func addPlain(h *providerHandler) {
	h.registry[plainProvider] = Descriptor{
		Factory:         h.registry[number.ProviderNameTelnyx].Factory,
		AllowedConfig:   []string{"k"},
		DefaultPriority: 50,
	}
}
