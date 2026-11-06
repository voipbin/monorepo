package confighandler_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/secretcrypto"
)

func Test_Set(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		key  string
		raw  string

		expectJSON string
		expectErr  error
	}{
		{name: "int", key: "a.limit", raw: "42", expectJSON: "42"},
		{name: "int with spaces", key: "a.limit", raw: " 42 ", expectJSON: "42"},
		{name: "int not a number", key: "a.limit", raw: "x", expectErr: confighandler.ErrInvalid},
		{name: "int below min", key: "a.limit", raw: "0", expectErr: confighandler.ErrInvalid},
		{name: "int above max", key: "a.limit", raw: "101", expectErr: confighandler.ErrInvalid},
		{name: "duration", key: "a.timeout", raw: "90s", expectJSON: `"1m30s"`},
		{name: "duration below min", key: "a.timeout", raw: "10ms", expectErr: confighandler.ErrInvalid},
		{name: "duration malformed", key: "a.timeout", raw: "soon", expectErr: confighandler.ErrInvalid},
		{name: "enum", key: "a.mode", raw: "y", expectJSON: `"y"`},
		{name: "enum outside", key: "a.mode", raw: "z", expectErr: confighandler.ErrInvalid},
		{name: "string", key: "p.model", raw: "m-1", expectJSON: `"m-1"`},
		{name: "string too long", key: "p.model", raw: strings.Repeat("a", 33), expectErr: confighandler.ErrInvalid},
		{name: "bool", key: "p.enabled", raw: "false", expectJSON: "false"},
		{name: "bool malformed", key: "p.enabled", raw: "maybe", expectErr: confighandler.ErrInvalid},
		{name: "secret key", key: "s.token", raw: "x", expectErr: confighandler.ErrWrongKind},
		{name: "unknown key", key: "nothing", raw: "x", expectErr: confighandler.ErrUnknownKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1), confighandler.WithSource("cli"))

			res, err := f.h.Set(ctx, testActor, tt.key, tt.raw)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
				if len(f.store.Histories()) != 0 {
					t.Errorf("Wrong match. a failed write left a history row")
				}
				if _, getErr := f.store.Get(ctx, tt.key); !errors.Is(getErr, confighandler.ErrNotFound) {
					t.Errorf("Wrong match. a failed write stored a row")
				}
				// the error text never repeats the rejected value
				if err != nil && tt.raw != "x" && len(tt.raw) > 2 && strings.Contains(err.Error(), tt.raw) {
					t.Errorf("Wrong match. the error repeats the value: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			// the members of an atomic group are read from the database only when the anchor key
			// of the group has a row, so the resolved source of such a member stays default here.
			expectSource := confighandler.SourceDB
			if strings.HasPrefix(tt.key, "p.") {
				expectSource = confighandler.SourceDefault
			}
			if res.Source != expectSource {
				t.Errorf("Wrong match. expect: %s, got: %+v", expectSource, res)
			}
			if row, _ := f.store.Get(ctx, tt.key); row == nil || row.Revision != 1 || row.UpdatedBy == nil || *row.UpdatedBy != testActor {
				t.Errorf("Wrong match. row: %+v", row)
			}

			row, _ := f.store.Get(ctx, tt.key)
			if string(row.ValueJSON) != tt.expectJSON || len(row.SecretCiphertext) != 0 || row.KeyVersion != 0 {
				t.Errorf("Wrong match. row: %s %x %d", row.ValueJSON, row.SecretCiphertext, row.KeyVersion)
			}

			hs := f.store.Histories()
			if len(hs) != 1 {
				t.Fatalf("Wrong match. expect: 1 history row, got: %d", len(hs))
			}
			h := hs[0]
			if h.Action != "set" || h.Key != tt.key || h.IsSecret || h.Source != "cli" || h.ActorID != testActor ||
				string(h.NewValueJSON) != tt.expectJSON || h.OldValueJSON != nil || h.OldKeyVersion != 0 || h.NewKeyVersion != 0 || !h.TMCreate.Equal(testNow) ||
				h.ID == uuid.Nil {
				t.Errorf("Wrong match. history: %+v", h)
			}

			// the stored value reads back through the typed read.
			switch tt.key {
			case "a.limit":
				if n, r, err := f.h.Int(ctx, tt.key); err != nil || r.Source != confighandler.SourceDB || fmt.Sprint(n) != tt.expectJSON {
					t.Errorf("Wrong match. got: %d %v", n, err)
				}
			case "a.timeout":
				if d, _, err := f.h.Duration(ctx, tt.key); err != nil || d <= 0 {
					t.Errorf("Wrong match. got: %v %v", d, err)
				}
			}
		})
	}
}

func Test_Set_overwriteKeepsOldValueInHistory(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	if _, err := f.h.Set(ctx, testActor, "a.limit", "10"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	res, err := f.h.Set(ctx, testActor, "a.limit", "20")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Revision != 2 {
		t.Errorf("Wrong match. expect: revision 2, got: %d", res.Revision)
	}

	hs := f.store.Histories()
	if len(hs) != 2 || string(hs[1].OldValueJSON) != "10" || string(hs[1].NewValueJSON) != "20" {
		t.Errorf("Wrong match. history: %+v", hs)
	}

	row, _ := f.store.Get(ctx, "a.limit")
	if !row.TMCreate.Equal(testNow) {
		t.Errorf("Wrong match. tm_create changed")
	}
}

func Test_Set_misuseAndErrors(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))

	if _, err := f.h.Set(ctx, uuid.Nil, "a.limit", "5"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. nil actor expect: ErrInvalid, got: %v", err)
	}
	if _, err := f.h.SetSecret(ctx, uuid.Nil, "s.token", "x"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. nil actor expect: ErrInvalid, got: %v", err)
	}
	if _, err := f.h.Clear(ctx, uuid.Nil, "a.limit"); !errors.Is(err, confighandler.ErrInvalid) {
		t.Errorf("Wrong match. nil actor expect: ErrInvalid, got: %v", err)
	}
	if _, err := f.h.Clear(ctx, testActor, "nothing"); !errors.Is(err, confighandler.ErrUnknownKey) {
		t.Errorf("Wrong match. expect: ErrUnknownKey, got: %v", err)
	}

	f.store.GetErr = errors.New("database is down")
	if _, err := f.h.Set(ctx, testActor, "a.limit", "5"); err == nil || errors.Is(err, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: the read error, got: %v", err)
	}
	if _, err := f.h.Clear(ctx, testActor, "a.limit"); err == nil || errors.Is(err, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: the read error, got: %v", err)
	}
	if len(f.store.Histories()) != 0 {
		t.Errorf("Wrong match. a failed write left a history row")
	}
}

func Test_SetSecret(t *testing.T) {
	ctx := context.Background()
	const value = "SECRET-VAL-12345"

	t.Run("stores a ciphertext bound to the row and a history without values", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1)+","+testKeyList(2, 0x40), confighandler.WithSource("cli"))

		res, err := f.h.SetSecret(ctx, testActor, "s.token", value)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.Source != confighandler.SourceDB || !res.HasSecret || res.KeyVersion != 2 || res.Revision != 1 {
			t.Errorf("Wrong match. resolved: %+v", res)
		}

		row, _ := f.store.Get(ctx, "s.token")
		if row.ValueType != "secret" || len(row.ValueJSON) != 0 || row.KeyVersion != 2 || len(row.SecretCiphertext) != 12+len(value)+16 {
			t.Errorf("Wrong match. row: %+v", row)
		}
		if strings.Contains(string(row.SecretCiphertext), value) {
			t.Errorf("Wrong match. the stored bytes hold the plaintext")
		}

		// the row decrypts only under its own table and key name.
		if got, err := f.crypto.Decrypt(row.SecretCiphertext, row.KeyVersion, secretcrypto.AAD(testTable, "s.token")); err != nil || got != value {
			t.Errorf("Wrong match. got: %q %v", got, err)
		}
		for _, other := range [][]byte{secretcrypto.AAD(testTable, "p.api_key"), secretcrypto.AAD("other_configs", "s.token"), nil} {
			if _, err := f.crypto.Decrypt(row.SecretCiphertext, row.KeyVersion, other); !errors.Is(err, secretcrypto.ErrDecrypt) {
				t.Errorf("Wrong match. another AAD expect: ErrDecrypt, got: %v", err)
			}
		}

		got, _, err := f.h.Secret(ctx, "s.token")
		if err != nil || got.Reveal() != value {
			t.Errorf("Wrong match. read back: %q %v", got.Reveal(), err)
		}

		hs := f.store.Histories()
		if len(hs) != 1 {
			t.Fatalf("Wrong match. expect: 1 history row, got: %d", len(hs))
		}
		h := hs[0]
		if h.Action != "set" || !h.IsSecret || h.OldValueJSON != nil || h.NewValueJSON != nil || h.OldKeyVersion != 0 || h.NewKeyVersion != 2 || h.Source != "cli" {
			t.Errorf("Wrong match. history: %+v", h)
		}
	})

	t.Run("overwrite records the key versions", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		f.putSecretRow(t, "s.token", "old", testTable, 5)

		f2 := confighandler.NewHandler(testRegistry(t), f.store, testCrypto(t, testKeyList(1, 1)+","+testKeyList(2, 0x40)), nil, testTable)
		res, err := f2.SetSecret(ctx, testActor, "s.token", value)
		if err != nil || res.Revision != 6 || res.KeyVersion != 2 {
			t.Fatalf("Wrong match. got: %+v %v", res, err)
		}
		hs := f.store.Histories()
		if len(hs) != 1 || hs[0].OldKeyVersion != 1 || hs[0].NewKeyVersion != 2 || hs[0].OldValueJSON != nil || hs[0].NewValueJSON != nil {
			t.Errorf("Wrong match. history: %+v", hs)
		}
	})

	tests := []struct {
		name    string
		keyList string
		key     string
		secret  string

		expectErr error
	}{
		{name: "no master key", keyList: "", key: "s.token", secret: "x", expectErr: confighandler.ErrKeyNotConfigured},
		{name: "empty secret", keyList: testKeyList(1, 1), key: "s.token", secret: "", expectErr: confighandler.ErrInvalid},
		{name: "too long for the key", keyList: testKeyList(1, 1), key: "s.token", secret: strings.Repeat("a", 17), expectErr: confighandler.ErrTooLong},
		{name: "at the limit", keyList: testKeyList(1, 1), key: "s.token", secret: strings.Repeat("a", 16)},
		{name: "not a secret key", keyList: testKeyList(1, 1), key: "a.limit", secret: "5", expectErr: confighandler.ErrWrongKind},
		{name: "unknown key", keyList: testKeyList(1, 1), key: "nothing", secret: "x", expectErr: confighandler.ErrUnknownKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.keyList)
			_, err := f.h.SetSecret(ctx, testActor, tt.key, tt.secret)
			if tt.expectErr == nil {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if len(f.store.Histories()) != 0 {
				t.Errorf("Wrong match. a failed write left a history row")
			}
			if tt.secret != "" && len(tt.secret) > 1 && strings.Contains(err.Error(), tt.secret) {
				t.Errorf("Wrong match. the error repeats the secret")
			}
		})
	}
}

// Test_history_neverHoldsValues pins the rule that a secret or sensitive key never leaves a value,
// a hash, a length or any derived text in the history, whatever the write path is.
func Test_history_neverHoldsValues(t *testing.T) {
	ctx := context.Background()
	const secretValue = "HIST-SECRET-PLAINTEXT"
	const sensitiveValue = "https://internal.example.invalid/path"

	f := newFixture(t, testKeyList(1, 1))

	steps := []func() error{
		func() error { _, err := f.h.SetSecret(ctx, testActor, "s.token", secretValue[:16]); return err },
		func() error { _, err := f.h.SetSecret(ctx, testActor, "s.token", "SECOND-SECRET-VAL"[:16]); return err },
		func() error { _, err := f.h.Set(ctx, testActor, "a.url", sensitiveValue); return err },
		func() error { _, err := f.h.Set(ctx, testActor, "a.url", sensitiveValue+"/2"); return err },
		func() error { _, err := f.h.Clear(ctx, testActor, "a.url"); return err },
		func() error { _, err := f.h.Clear(ctx, testActor, "s.token"); return err },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("Wrong match. step %d expect: ok, got: %v", i, err)
		}
	}

	hs := f.store.Histories()
	if len(hs) != len(steps) {
		t.Fatalf("Wrong match. expect: %d history rows, got: %d", len(steps), len(hs))
	}
	for i, h := range hs {
		if !h.IsSecret {
			t.Errorf("Wrong match. history %d must be marked secret", i)
		}
		if h.OldValueJSON != nil || h.NewValueJSON != nil {
			t.Errorf("Wrong match. history %d carries a value column: old=%q new=%q", i, h.OldValueJSON, h.NewValueJSON)
		}
		text := fmt.Sprintf("%+v", *h)
		for _, v := range []string{secretValue[:16], "SECOND-SECRET-VAL", "internal.example.invalid"} {
			if strings.Contains(text, v) {
				t.Errorf("Wrong match. history %d leaks %q", i, v)
			}
		}
	}
	if hs[1].OldKeyVersion != 1 || hs[1].NewKeyVersion != 1 || hs[5].Action != "clear" || hs[5].OldKeyVersion != 1 {
		t.Errorf("Wrong match. key versions: %+v %+v", hs[1], hs[5])
	}
}

func Test_Clear(t *testing.T) {
	ctx := context.Background()

	t.Run("a param row is removed and the old value is kept in the history", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1), confighandler.WithSource("cli"))
		f.env["A_LIMIT"] = "9"
		if _, err := f.h.Set(ctx, testActor, "a.limit", "30"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		res, err := f.h.Clear(ctx, testActor, "a.limit")
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		// back to the environment value
		if res.Source != confighandler.SourceEnv {
			t.Errorf("Wrong match. expect: env, got: %s", res.Source)
		}
		if _, err := f.store.Get(ctx, "a.limit"); !errors.Is(err, confighandler.ErrNotFound) {
			t.Errorf("Wrong match. the row is still there")
		}
		hs := f.store.Histories()
		last := hs[len(hs)-1]
		if last.Action != "clear" || string(last.OldValueJSON) != "30" || last.NewValueJSON != nil || last.IsSecret || last.Source != "cli" {
			t.Errorf("Wrong match. history: %+v", last)
		}
	})

	t.Run("with no env value the key becomes none or default", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		if _, err := f.h.SetSecret(ctx, testActor, "s.token", "abc"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		res, err := f.h.Clear(ctx, testActor, "s.token")
		if err != nil || res.Source != confighandler.SourceNone || res.HasSecret {
			t.Errorf("Wrong match. got: %+v %v", res, err)
		}
	})

	t.Run("a missing row is a no-op without history", func(t *testing.T) {
		f := newFixture(t, testKeyList(1, 1))
		res, err := f.h.Clear(ctx, testActor, "s.token")
		if err != nil || res.Source != confighandler.SourceNone {
			t.Fatalf("Wrong match. got: %+v %v", res, err)
		}
		if len(f.store.Histories()) != 0 {
			t.Errorf("Wrong match. a no-op wrote a history row")
		}
	})
}

func Test_write_retryOnConflict(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		conflicts int
		prepare   func(f *fixture)
		call      func(f *fixture) error

		expectErr     error
		expectHistory int
		expectCalls   int // Get calls of the write
	}{
		{name: "set succeeds on the second attempt", conflicts: 1, call: func(f *fixture) error { _, err := f.h.Set(ctx, testActor, "a.limit", "5"); return err }, expectHistory: 1, expectCalls: 3},
		{name: "set succeeds on the third attempt", conflicts: 2, call: func(f *fixture) error { _, err := f.h.Set(ctx, testActor, "a.limit", "5"); return err }, expectHistory: 1, expectCalls: 4},
		{name: "set gives up after three attempts", conflicts: 3, call: func(f *fixture) error { _, err := f.h.Set(ctx, testActor, "a.limit", "5"); return err }, expectErr: confighandler.ErrConflict, expectHistory: 0, expectCalls: 3},
		{name: "secret set succeeds on the second attempt", conflicts: 1, call: func(f *fixture) error { _, err := f.h.SetSecret(ctx, testActor, "s.token", "abc"); return err }, expectHistory: 1, expectCalls: 3},
		{name: "clear succeeds on the second attempt", conflicts: 1, prepare: func(f *fixture) { f.putValueRow("a.limit", "int", "5") }, call: func(f *fixture) error { _, err := f.h.Clear(ctx, testActor, "a.limit"); return err }, expectHistory: 1, expectCalls: 3},
		{name: "clear gives up after three attempts", conflicts: 3, prepare: func(f *fixture) { f.putValueRow("a.limit", "int", "5") }, call: func(f *fixture) error { _, err := f.h.Clear(ctx, testActor, "a.limit"); return err }, expectErr: confighandler.ErrConflict, expectHistory: 0, expectCalls: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList(1, 1))
			if tt.prepare != nil {
				tt.prepare(f)
			}
			f.store.ConflictWrites = tt.conflicts

			err := tt.call(f)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if len(f.store.Histories()) != tt.expectHistory {
				t.Errorf("Wrong match. expect: %d history rows, got: %d", tt.expectHistory, len(f.store.Histories()))
			}
			if f.store.GetCalls != tt.expectCalls {
				t.Errorf("Wrong match. expect: %d Get calls, got: %d", tt.expectCalls, f.store.GetCalls)
			}
		})
	}
}

// the retry builds a fresh history row from the row it just read.
func Test_write_retryUsesTheFreshRow(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))
	f.putValueRow("a.limit", "int", "11")

	f.store.ConflictWrites = 1
	if _, err := f.h.Set(ctx, testActor, "a.limit", "12"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	hs := f.store.Histories()
	if len(hs) != 1 || string(hs[0].OldValueJSON) != "11" || string(hs[0].NewValueJSON) != "12" {
		t.Errorf("Wrong match. history: %+v", hs)
	}
	row, _ := f.store.Get(ctx, "a.limit")
	if row.Revision != 2 {
		t.Errorf("Wrong match. expect: revision 2, got: %d", row.Revision)
	}
}

func Test_defaultSource(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, testKeyList(1, 1))
	if _, err := f.h.Set(ctx, testActor, "a.limit", "5"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if hs := f.store.Histories(); hs[0].Source != "api" {
		t.Errorf("Wrong match. expect: api, got: %s", hs[0].Source)
	}
}

func Test_NewHandler_nilEnvAndCrypto(t *testing.T) {
	ctx := context.Background()
	h := confighandler.NewHandler(testRegistry(t), newFixture(t, "").store, nil, nil, testTable)

	if n, res, err := h.Int(ctx, "a.limit"); err != nil || n != 10 || res.Source != confighandler.SourceDefault {
		t.Errorf("Wrong match. got: %d %+v %v", n, res, err)
	}
	if _, err := h.SetSecret(ctx, testActor, "s.token", "abc"); !errors.Is(err, confighandler.ErrKeyNotConfigured) {
		t.Errorf("Wrong match. expect: ErrKeyNotConfigured, got: %v", err)
	}
}
