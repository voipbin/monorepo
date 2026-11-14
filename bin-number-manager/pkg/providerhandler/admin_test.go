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
	h := newTestHandler(t, newFakeDB())

	list, err := h.List(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("Wrong match. expect: two providers, got: %v, %v", list, err)
	}
	if list[0].ProviderName != number.ProviderNameTelnyx || list[1].ProviderName != number.ProviderNameTwilio {
		t.Errorf("Wrong order. got: %s, %s", list[0].ProviderName, list[1].ProviderName)
	}

	tel := list[0]
	if !tel.Enabled || tel.Priority != 10 || tel.HasSecret || tel.CredentialSource != provider.CredentialSourceNone || tel.Usable {
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
		h := newTestHandler(t, db)

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
		h := newTestHandler(t, db)

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
				h := newTestHandler(t, db)

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
		h := newTestHandler(t, newFakeDB())
		if _, err := h.Update(ctx, "nope", testActor, boolPtr(true), nil, nil); !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: not found, got: %v", err)
		}
	})

	t.Run("deleting a required key is accepted and the provider is not usable without a value", func(t *testing.T) {
		setCredentialEnv(t, map[string]string{})

		db := newFakeDB()
		h := newTestHandler(t, db)

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": "c1"}); err != nil {
			t.Fatal(err)
		}

		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": ""})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if _, ok := db.rows[number.ProviderNameTelnyx].Config["connection_id"]; ok {
			t.Errorf("The key was not removed from the row")
		}
		if info.Usable {
			t.Errorf("Wrong match. expect: not usable without a connection id, got: %+v", info)
		}
	})

	t.Run("a required key that a variable fills keeps the provider usable", func(t *testing.T) {
		setCredentialEnv(t, map[string]string{envTelnyxAPIKey: "env-token", envTelnyxConnectionID: "env-conn"})

		db := newFakeDB()
		h := newTestHandler(t, db)

		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, nil, map[string]string{"connection_id": ""})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if !info.Usable {
			t.Errorf("Wrong match. expect: usable, got: %+v", info)
		}
	})

	t.Run("a db error on the read does not create defaults", func(t *testing.T) {
		db := newFakeDB()
		db.updateErr = errTestDB
		h := newTestHandler(t, db)

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil); !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})

	t.Run("ensure failure", func(t *testing.T) {
		db := newFakeDB()
		db.ensureErr = errTestDB
		h := newTestHandler(t, db)

		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil); !errors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})
}

func Test_Update_boundaries(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		priority int
		ok       bool
	}{{0, true}, {1, true}, {maxPriority, true}, {maxPriority + 1, false}, {-1, false}} {
		h := newTestHandler(t, newFakeDB())
		_, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, nil, intPtr(tt.priority), nil)
		if (err == nil) != tt.ok {
			t.Errorf("priority %d: expect ok=%v, got: %v", tt.priority, tt.ok, err)
		}
	}

	t.Run("an update of one field keeps the registry defaults of the others on a new row", func(t *testing.T) {
		db := newFakeDB()
		h := newTestHandler(t, db)

		info, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(false), nil, nil)
		if err != nil || info.Enabled || info.Priority != 10 {
			t.Errorf("Wrong match. expect: disabled with priority 10, got: %+v, %v", info, err)
		}
	})

	t.Run("the number of config keys is limited at eight", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB())
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
