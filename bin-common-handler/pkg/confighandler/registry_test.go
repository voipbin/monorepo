package confighandler_test

import (
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/confighandler"
)

func ptr(n int64) *int64 { return &n }

func Test_NewRegistry(t *testing.T) {
	secretKey := func(name string) confighandler.Key {
		return confighandler.Key{Name: name, Kind: confighandler.KindSecret, Type: confighandler.TypeString, Env: "SOME_API_KEY", MaxLen: 64}
	}
	selectBool := confighandler.Key{Name: "p.enabled", Kind: confighandler.KindSelect, Type: confighandler.TypeBool, Group: "p", Atomic: true}

	tests := []struct {
		name string
		keys []confighandler.Key

		expectErr string // empty: no error; otherwise a part of the error text
	}{
		{name: "empty registry", keys: nil},
		{name: "valid secret", keys: []confighandler.Key{secretKey("a.api_key")}},
		{name: "valid param int", keys: []confighandler.Key{{Name: "a.limit", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Default: 5, Min: ptr(1), Max: ptr(10)}}},
		{name: "valid enum", keys: []confighandler.Key{{Name: "a.mode", Kind: confighandler.KindSelect, Type: confighandler.TypeEnum, Enum: []string{"x", "y"}, Default: "x"}}},
		{name: "valid duration", keys: []confighandler.Key{{Name: "a.timeout", Kind: confighandler.KindParam, Type: confighandler.TypeDuration, Default: 3 * time.Second, Min: ptr(int64(time.Second))}}},

		{name: "duplicate name", keys: []confighandler.Key{secretKey("a.k"), secretKey("a.k")}, expectErr: "more than once"},
		{name: "same environment name", keys: []confighandler.Key{secretKey("a.k1"), secretKey("a.k2")}, expectErr: "same environment name"},
		{name: "enum value repeated", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeEnum, Enum: []string{"x", "x"}}}, expectErr: "non-empty and unique"},
		{name: "enum value empty", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeEnum, Enum: []string{"x", ""}}}, expectErr: "non-empty and unique"},
		{name: "upper case name", keys: []confighandler.Key{secretKey("A.k")}, expectErr: "malformed"},
		{name: "name with dash", keys: []confighandler.Key{secretKey("a-k")}, expectErr: "malformed"},
		{name: "empty name", keys: []confighandler.Key{secretKey("")}, expectErr: "malformed"},
		{name: "name too long", keys: []confighandler.Key{secretKey("a" + strings.Repeat("b", 128))}, expectErr: "malformed"},
		{name: "unknown kind", keys: []confighandler.Key{{Name: "a.k", Kind: "x", Type: confighandler.TypeInt}}, expectErr: "unknown kind"},
		{name: "unknown type", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: "x"}}, expectErr: "unknown type"},

		{name: "reserved env DATABASE_DSN", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 5, Env: "DATABASE_DSN"}}, expectErr: "bootstrap"},
		{name: "reserved env REDIS_PASSWORD", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 5, Env: "REDIS_PASSWORD"}}, expectErr: "bootstrap"},
		{name: "reserved env LOG_LEVEL", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 5, Env: "LOG_LEVEL"}}, expectErr: "bootstrap"},
		{name: "reserved env suffix", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 5, Env: "MY_SECRET_ENCRYPTION_KEYS"}}, expectErr: "bootstrap"},

		{name: "default of the wrong type", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Default: "x"}}, expectErr: "default"},
		{name: "default out of range", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Default: 50, Max: ptr(10)}}, expectErr: "default"},
		{name: "default not in enum", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeEnum, Enum: []string{"x"}, Default: "z"}}, expectErr: "default"},
		{name: "min greater than max", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Min: ptr(5), Max: ptr(1)}}, expectErr: "min is greater"},
		{name: "min on a string", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 4, Min: ptr(1)}}, expectErr: "min and max"},
		{name: "enum without values", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeEnum}}, expectErr: "needs values"},
		{name: "enum values on a string", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 4, Enum: []string{"x"}}}, expectErr: "enum type only"},
		{name: "string without max_len", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeString}}, expectErr: "max_len"},

		{name: "secret not a string", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSecret, Type: confighandler.TypeInt, MaxLen: 4}}, expectErr: "must be a string"},
		{name: "secret with a default", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: 4, Default: "x"}}, expectErr: "default"},
		{name: "secret without max_len", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSecret, Type: confighandler.TypeString}}, expectErr: "max_len"},
		{name: "secret max_len too large", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: 2021}}, expectErr: "max_len"},
		{name: "secret max_len at the limit", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: 2020}}},

		// the bool rule: only a select key inside an atomic group.
		{name: "bool inside an atomic group is allowed", keys: []confighandler.Key{secretKeyInGroup("p.api_key", "p"), selectBool}},
		{name: "standalone bool select", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSelect, Type: confighandler.TypeBool}}, expectErr: "bool"},
		{name: "bool param", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeBool, Group: "g", Atomic: true}}, expectErr: "bool"},
		{name: "bool secret", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSecret, Type: confighandler.TypeBool, MaxLen: 4}}, expectErr: "bool"},
		{name: "bool select in a non atomic group", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindSelect, Type: confighandler.TypeBool, Group: "g"}}, expectErr: "bool"},

		{name: "atomic without group", keys: []confighandler.Key{{Name: "a.k", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Atomic: true}}, expectErr: "no group"},
		{name: "group mixes atomic and not", keys: []confighandler.Key{secretKeyInGroup("p.api_key", "p"), {Name: "p.model", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 8, Group: "p"}}, expectErr: "mixes"},
		{name: "atomic group without an anchor", keys: []confighandler.Key{{Name: "p.model", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 8, Group: "p", Atomic: true}}, expectErr: "anchor"},
		{name: "atomic group with two secrets", keys: []confighandler.Key{secretKeyInGroup("p.a", "p"), secretKeyInGroup("p.b", "p")}, expectErr: "ambiguous"},
		{name: "atomic group with two selects and no secret", keys: []confighandler.Key{
			{Name: "p.s1", Kind: confighandler.KindSelect, Type: confighandler.TypeEnum, Enum: []string{"x"}, Group: "p", Atomic: true},
			{Name: "p.s2", Kind: confighandler.KindSelect, Type: confighandler.TypeEnum, Enum: []string{"x"}, Group: "p", Atomic: true},
		}, expectErr: "anchor"},
		{name: "atomic group anchored by a select", keys: []confighandler.Key{
			{Name: "p.s1", Kind: confighandler.KindSelect, Type: confighandler.TypeEnum, Enum: []string{"x"}, Group: "p", Atomic: true},
			{Name: "p.model", Kind: confighandler.KindParam, Type: confighandler.TypeString, MaxLen: 8, Group: "p", Atomic: true},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := confighandler.NewRegistry(tt.keys)
			if tt.expectErr == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if len(reg.Keys()) != len(tt.keys) {
					t.Errorf("Wrong match. expect: %d keys, got: %d", len(tt.keys), len(reg.Keys()))
				}
				return
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error containing %q, got: nil", tt.expectErr)
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func secretKeyInGroup(name, group string) confighandler.Key {
	return confighandler.Key{Name: name, Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: 64, Group: group, Atomic: true}
}

func Test_RegistryLookupAndKeys(t *testing.T) {
	reg, err := confighandler.NewRegistry([]confighandler.Key{
		{Name: "b.key", Kind: confighandler.KindSecret, Type: confighandler.TypeString, MaxLen: 8},
		{Name: "a.limit", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Default: 7},
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	keys := reg.Keys()
	if len(keys) != 2 || keys[0].Name != "b.key" || keys[1].Name != "a.limit" {
		t.Errorf("Wrong match. declaration order expected, got: %v", keys)
	}
	keys[0].Name = "changed"
	if k, ok := reg.Lookup("b.key"); !ok || k.Name != "b.key" {
		t.Errorf("Wrong match. Keys() returned the internal slice")
	}

	if k, ok := reg.Lookup("a.limit"); !ok || k.Default != int64(7) {
		t.Errorf("Wrong match. the default is normalized to int64, got: %v (%T)", k.Default, k.Default)
	}
	if _, ok := reg.Lookup("nothing"); ok {
		t.Errorf("Wrong match. expect: not found")
	}
}
