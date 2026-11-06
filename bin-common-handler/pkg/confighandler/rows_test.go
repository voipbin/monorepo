package confighandler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/confighandler"
)

// Test_resolve_storedValues covers every type decoded from a stored row, valid and invalid.
func Test_resolve_storedValues(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		valueType string
		json      string

		expectErr bool
		check     func(t *testing.T, f *fixture)
	}{
		{name: "string", key: "a.url", valueType: "string", json: `"https://example.invalid"`, check: func(t *testing.T, f *fixture) {
			if s, _, err := f.h.String(ctx, "a.url"); err != nil || s != "https://example.invalid" {
				t.Errorf("Wrong match. got: %q %v", s, err)
			}
		}},
		{name: "string too long", key: "p.model", valueType: "string", json: `"0123456789012345678901234567890123456789"`, expectErr: true},
		{name: "string given a number", key: "a.url", valueType: "string", json: `5`, expectErr: true},
		{name: "enum", key: "a.mode", valueType: "enum", json: `"y"`, check: func(t *testing.T, f *fixture) {
			if s, _, err := f.h.String(ctx, "a.mode"); err != nil || s != "y" {
				t.Errorf("Wrong match. got: %q %v", s, err)
			}
		}},
		{name: "enum outside", key: "a.mode", valueType: "enum", json: `"z"`, expectErr: true},
		{name: "duration", key: "a.timeout", valueType: "duration", json: `"2s"`, check: func(t *testing.T, f *fixture) {
			if d, _, err := f.h.Duration(ctx, "a.timeout"); err != nil || d != 2*time.Second {
				t.Errorf("Wrong match. got: %v %v", d, err)
			}
		}},
		{name: "duration below min", key: "a.timeout", valueType: "duration", json: `"1ms"`, expectErr: true},
		{name: "duration not a string", key: "a.timeout", valueType: "duration", json: `5`, expectErr: true},
		{name: "duration malformed", key: "a.timeout", valueType: "duration", json: `"soon"`, expectErr: true},
		{name: "empty json", key: "a.limit", valueType: "int", json: ``, expectErr: true},
		{name: "int above max", key: "a.limit", valueType: "int", json: `101`, expectErr: true},
		{name: "wrong stored type name", key: "a.mode", valueType: "string", json: `"x"`, expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			f.putValueRow(tt.key, tt.valueType, tt.json)
			if tt.key == "p.model" {
				// a member of an atomic group is read from the database only when the anchor has a row.
				f.putSecretRow(t, "p.api_key", "db-key", testTable, 1)
			}

			var err error
			var res confighandler.Resolved
			switch tt.key {
			case "a.limit":
				_, res, err = f.h.Int(ctx, tt.key)
			case "a.timeout":
				_, res, err = f.h.Duration(ctx, tt.key)
			default:
				_, res, err = f.h.String(ctx, tt.key)
			}

			if tt.expectErr {
				if !errors.Is(err, confighandler.ErrInvalid) || res.Source != confighandler.SourceDBInvalid {
					t.Errorf("Wrong match. expect: db_invalid, got: %s %v", res.Source, err)
				}
				return
			}
			if err != nil || res.Source != confighandler.SourceDB {
				t.Fatalf("Wrong match. expect: ok, got: %s %v", res.Source, err)
			}
			tt.check(t, f)
		})
	}
}

// Test_resolve_boolMember reads the bool select key of an atomic group from the database and from the env.
func Test_resolve_boolMember(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	f.env["P_ENABLED"] = "false"
	if b, res, err := f.h.Bool(ctx, "p.enabled"); err != nil || b || res.Source != confighandler.SourceEnv {
		t.Errorf("Wrong match. got: %v %s %v", b, res.Source, err)
	}
	f.env["P_ENABLED"] = "maybe"
	if _, _, err := f.h.Bool(ctx, "p.enabled"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. expect: ErrInvalid, got: %v", err)
	}
	delete(f.env, "P_ENABLED")

	f.putSecretRow(t, "p.api_key", "db-key", testTable, 1)
	f.putValueRow("p.enabled", "bool", `false`)
	if b, res, err := f.h.Bool(ctx, "p.enabled"); err != nil || b || res.Source != confighandler.SourceDB {
		t.Errorf("Wrong match. got: %v %s %v", b, res.Source, err)
	}
	f.putValueRow("p.enabled", "bool", `"yes"`)
	if _, _, err := f.h.Bool(ctx, "p.enabled"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. expect: ErrInvalid, got: %v", err)
	}
}

func Test_describe_storeErrorOnWriteResult(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	// the write succeeds, then reading back for the result fails: the error is returned.
	f.store.GetErr = nil
	if _, err := f.h.Set(ctx, testActor, "a.limit", "5"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	f.store.GetErr = errors.New("database is down")
	if _, err := f.h.Clear(ctx, testActor, "a.limit"); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}
