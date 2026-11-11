package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
)

func resetConfigTables(t *testing.T) {
	t.Helper()

	for _, table := range []string{"transcribe_config_histories", "transcribe_configs"} {
		if _, err := dbTest.Exec("delete from " + table); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func countHistories(t *testing.T) int {
	t.Helper()

	var n int
	if err := dbTest.QueryRow("select count(*) from transcribe_config_histories").Scan(&n); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return n
}

// Test_configStore_contract runs the contract that every confighandler.Store must pass, the same
// one that the in-memory store of the library passes.
func Test_configStore_contract(t *testing.T) {
	storetest.Contract(t,
		func(t *testing.T) confighandler.Store {
			resetConfigTables(t)
			return NewConfigStore(dbTest)
		},
		func(t *testing.T, _ confighandler.Store) int { return countHistories(t) },
	)
}

func Test_configStore_list(t *testing.T) {
	resetConfigTables(t)
	ctx := context.Background()
	s := NewConfigStore(dbTest)

	now := time.Date(2026, 5, 6, 7, 8, 9, 123456000, time.UTC)
	actor := uuid.Must(uuid.NewV4())
	for _, key := range []string{"b.key", "a.key", "c.key"} {
		row := &confighandler.Row{Key: key, ValueType: "string", ValueJSON: []byte(`"x"`), Revision: 1, UpdatedBy: &actor, TMCreate: now, TMUpdate: now}
		hist := &confighandler.History{ID: uuid.Must(uuid.NewV4()), Key: key, Action: "set", ActorID: actor, Source: "cli", TMCreate: now, NewValueJSON: []byte(`"x"`)}
		if err := s.Insert(ctx, row, hist); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	rows, err := s.List(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	var keys []string
	for _, r := range rows {
		keys = append(keys, r.Key)
		if !r.TMCreate.Equal(now) {
			t.Errorf("Wrong match. expect: %v, got: %v", now, r.TMCreate)
		}
	}
	if strings.Join(keys, ",") != "a.key,b.key,c.key" {
		t.Errorf("Wrong match. expect: ordered keys, got: %v", keys)
	}
}

func Test_configStore_updateKeepsCreateTimeAndWritesRevision(t *testing.T) {
	resetConfigTables(t)
	ctx := context.Background()
	s := NewConfigStore(dbTest)

	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := created.Add(time.Hour)
	actor := uuid.Must(uuid.NewV4())

	row := &confighandler.Row{Key: "k", ValueType: "secret", SecretCiphertext: []byte("0123456789012345678901234567890123456789"), KeyVersion: 1, Revision: 1, UpdatedBy: &actor, TMCreate: created, TMUpdate: created}
	hist := &confighandler.History{ID: uuid.Must(uuid.NewV4()), Key: "k", Action: "set", IsSecret: true, ActorID: actor, Source: "cli", TMCreate: created, NewKeyVersion: 1}
	if err := s.Insert(ctx, row, hist); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	upd := &confighandler.Row{Key: "k", ValueType: "secret", SecretCiphertext: []byte("abcdefghijklmnopqrstuvwxyz0123456789"), KeyVersion: 2, UpdatedBy: &actor, TMCreate: later, TMUpdate: later}
	hist2 := &confighandler.History{ID: uuid.Must(uuid.NewV4()), Key: "k", Action: "set", IsSecret: true, ActorID: actor, Source: "cli", TMCreate: later, OldKeyVersion: 1, NewKeyVersion: 2}
	if err := s.Update(ctx, upd, 1, hist2); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	got, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Revision != 2 || got.KeyVersion != 2 || !got.TMCreate.Equal(created) || !got.TMUpdate.Equal(later) || string(got.SecretCiphertext) != string(upd.SecretCiphertext) {
		t.Errorf("Wrong match. got: %+v", got)
	}

	// the history of a secret has no value columns.
	var oldV, newV sql.NullString
	var isSecret bool
	if err := dbTest.QueryRow("select old_value_json, new_value_json, is_secret from transcribe_config_histories where id = ?", hist2.ID.Bytes()).Scan(&oldV, &newV, &isSecret); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if oldV.Valid || newV.Valid || !isSecret {
		t.Errorf("Wrong match. the secret history row holds a value: %v %v %v", oldV, newV, isSecret)
	}
}

// Test_configStore_failedHistoryRollsBack proves that the row change and the history are one
// transaction: when the history row can not be written, the row change is undone.
func Test_configStore_failedHistoryRollsBack(t *testing.T) {
	resetConfigTables(t)
	ctx := context.Background()
	s := NewConfigStore(dbTest)

	now := time.Now().UTC()
	actor := uuid.Must(uuid.NewV4())
	histID := uuid.Must(uuid.NewV4())

	row := func(key string) *confighandler.Row {
		return &confighandler.Row{Key: key, ValueType: "string", ValueJSON: []byte(`"x"`), Revision: 1, UpdatedBy: &actor, TMCreate: now, TMUpdate: now}
	}
	hist := func(key string) *confighandler.History {
		return &confighandler.History{ID: histID, Key: key, Action: "set", ActorID: actor, Source: "cli", TMCreate: now}
	}

	if err := s.Insert(ctx, row("first"), hist("first")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the same history id again: the history insert fails after the row insert.
	err := s.Insert(ctx, row("second"), hist("second"))
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: %v", err)
	}
	if _, errGet := s.Get(ctx, "second"); !errors.Is(errGet, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. the row stayed after a failed history insert: %v", errGet)
	}

	// an update whose history fails is undone as well
	upd := row("first")
	upd.ValueJSON = []byte(`"changed"`)
	if err := s.Update(ctx, upd, 1, hist("first")); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	got, _ := s.Get(ctx, "first")
	if got.Revision != 1 || string(got.ValueJSON) != `"x"` {
		t.Errorf("Wrong match. the update stayed after a failed history insert: %+v", got)
	}

	if err := s.Delete(ctx, "first", 1, hist("first")); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if _, errGet := s.Get(ctx, "first"); errGet != nil {
		t.Errorf("Wrong match. the delete stayed after a failed history insert: %v", errGet)
	}

	// a nil history is refused
	if err := s.Insert(ctx, row("third"), nil); err == nil {
		t.Errorf("Wrong match. expect: error for a missing history row")
	}
	if _, errGet := s.Get(ctx, "third"); !errors.Is(errGet, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. the row stayed without a history row")
	}
}

func Test_configStore_errors(t *testing.T) {
	ctx := context.Background()

	closed, err := sql.Open("sqlite3", "file:closed?mode=memory")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	_ = closed.Close()
	s := NewConfigStore(closed)

	if _, err := s.Get(ctx, "k"); err == nil || errors.Is(err, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. a database error must not be not found, got: %v", err)
	}
	if _, err := s.List(ctx); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}

	now := time.Now()
	actor := uuid.Must(uuid.NewV4())
	row := &confighandler.Row{Key: "k", ValueType: "string", ValueJSON: []byte(`"x"`), Revision: 1, TMCreate: now, TMUpdate: now}
	hist := &confighandler.History{ID: uuid.Must(uuid.NewV4()), Key: "k", Action: "set", ActorID: actor, Source: "cli", TMCreate: now}
	if err := s.Insert(ctx, row, hist); err == nil || errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: a database error, got: %v", err)
	}
	if err := s.Update(ctx, row, 1, hist); err == nil || errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: a database error, got: %v", err)
	}
	if err := s.Delete(ctx, "k", 1, hist); err == nil || errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: a database error, got: %v", err)
	}
}

func Test_nullValues(t *testing.T) {
	if nullString(nil) != nil || nullString([]byte{}) != nil || nullBytes(nil) != nil || nullBytes([]byte{}) != nil {
		t.Errorf("Wrong match. expect: NULL for nil and empty input")
	}
	if nullString([]byte(`"x"`)) != `"x"` {
		t.Errorf("Wrong match. expect: the JSON as text")
	}
	if b, ok := nullBytes([]byte{1, 2}).([]byte); !ok || len(b) != 2 {
		t.Errorf("Wrong match. expect: the bytes")
	}
}

func Test_parseTime(t *testing.T) {
	want := time.Date(2026, 5, 6, 7, 8, 9, 123456000, time.UTC)

	tests := []struct {
		name  string
		value any

		expectErr bool
	}{
		{name: "time", value: want},
		{name: "mysql text", value: []byte("2026-05-06 07:08:09.123456")},
		{name: "mysql text without fraction", value: "2026-05-06 07:08:09", expectErr: false},
		{name: "sqlite text with zone", value: "2026-05-06 07:08:09.123456+00:00"},
		{name: "rfc3339", value: "2026-05-06T07:08:09.123456Z"},
		{name: "unknown text", value: "yesterday", expectErr: true},
		{name: "unexpected type", value: 12, expectErr: true},
		{name: "nil", value: nil, expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTime(tt.value)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.name != "mysql text without fraction" && !got.Equal(want) {
				t.Errorf("Wrong match. expect: %v, got: %v", want, got)
			}
		})
	}
}
