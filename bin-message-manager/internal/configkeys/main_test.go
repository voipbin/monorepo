package configkeys

import (
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/confighandler"
)

func Test_NewRegistry(t *testing.T) {
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(reg.Keys()) != 2 {
		t.Errorf("Wrong match. expect: 2 keys, got: %d", len(reg.Keys()))
	}
}

// Test_goldenKinds pins the kind, the type and the environment name of every key. A key name is
// never moved to another kind: rows and history that exist under the old meaning would be
// read as the new one. Declare a new key name instead, and update this table in the same change.
func Test_goldenKinds(t *testing.T) {
	expect := []struct {
		name   string
		kind   confighandler.Kind
		typ    confighandler.Type
		env    string
		maxLen int
	}{
		{"provider.messagebird.auth_token", confighandler.KindSecret, confighandler.TypeString, "AUTHTOKEN_MESSAGEBIRD", 512},
		{"provider.telnyx.auth_token", confighandler.KindSecret, confighandler.TypeString, "AUTHTOKEN_TELNYX", 512},
	}

	keys := Keys()
	if len(keys) != len(expect) {
		t.Fatalf("Wrong match. expect: %d keys, got: %d", len(expect), len(keys))
	}
	for i, e := range expect {
		k := keys[i]
		if k.Name != e.name || k.Kind != e.kind || k.Type != e.typ || k.Env != e.env || k.MaxLen != e.maxLen {
			t.Errorf("Wrong match. key %d expect: %+v, got: %+v", i, e, k)
		}
		if k.Default != nil {
			t.Errorf("Wrong match. %s must have no default", k.Name)
		}
	}
}

// Every key has a one line description for operators, and it never names a value.
func Test_descriptions(t *testing.T) {
	for _, k := range Keys() {
		if k.Description == "" {
			t.Errorf("Wrong match. %s has no description", k.Name)
		}
		if strings.ContainsAny(k.Description, "\r\n") || len(k.Description) > 300 {
			t.Errorf("Wrong match. %s: the description must be one short line, got: %q", k.Name, k.Description)
		}
	}
}

func Test_tableNames(t *testing.T) {
	if Table != "message_configs" || HistoryTable != "message_config_histories" {
		t.Errorf("Wrong match. the table names are part of the stored secret contract: %s %s", Table, HistoryTable)
	}
}
