package dbhandler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-email-manager/internal/configkeys"
)

// openMySQLForConfigs opens a throwaway MariaDB or MySQL whose DSN is given in
// EMAIL_CONFIG_TEST_MYSQL_DSN. The tests are skipped without it: the migrations of the service
// are applied to that database before the run (see the service docs). Never point it at a
// shared or production database; the tests empty the two configs tables.
func openMySQLForConfigs(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("EMAIL_CONFIG_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("EMAIL_CONFIG_TEST_MYSQL_DSN is not set")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, table := range []string{"email_config_histories", "email_configs"} {
		if _, err := db.Exec("delete from " + table); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	return db
}

func Test_configStore_mysql_contract(t *testing.T) {
	db := openMySQLForConfigs(t)

	storetest.Contract(t,
		func(t *testing.T) confighandler.Store {
			for _, table := range []string{"email_config_histories", "email_configs"} {
				if _, err := db.Exec("delete from " + table); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			}
			return NewConfigStore(db)
		},
		func(t *testing.T, _ confighandler.Store) int {
			var n int
			if err := db.QueryRow("select count(*) from email_config_histories").Scan(&n); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			return n
		},
	)
}

// Test_configStore_mysql_constraints checks what only a real server enforces: the duplicate key
// number, the CHECK constraints and the transaction.
func Test_configStore_mysql_constraints(t *testing.T) {
	db := openMySQLForConfigs(t)
	ctx := context.Background()
	s := NewConfigStore(db)

	now := time.Date(2026, 5, 6, 7, 8, 9, 123456000, time.UTC)
	actor := uuid.Must(uuid.NewV4())
	hist := func(key string) *confighandler.History {
		return &confighandler.History{ID: uuid.Must(uuid.NewV4()), Key: key, Action: "set", IsSecret: true, ActorID: actor, Source: "cli", TMCreate: now, NewKeyVersion: 1}
	}
	secretRow := func(key string) *confighandler.Row {
		return &confighandler.Row{Key: key, ValueType: "secret", SecretCiphertext: []byte("0123456789012345678901234567890123456789"), KeyVersion: 1, Revision: 1, UpdatedBy: &actor, TMCreate: now, TMUpdate: now}
	}

	if err := s.Insert(ctx, secretRow("k"), hist("k")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := s.Insert(ctx, secretRow("k"), hist("k")); !errors.Is(err, confighandler.ErrConflict) {
		t.Errorf("Wrong match. expect: ErrConflict for a duplicate key (1062), got: %v", err)
	}

	got, err := s.Get(ctx, "k")
	if err != nil || !got.TMCreate.Equal(now) || !got.TMUpdate.Equal(now) || got.Revision != 1 || got.UpdatedBy == nil || *got.UpdatedBy != actor {
		t.Errorf("Wrong match. got: %+v %v", got, err)
	}

	// a secret row that also holds a plain value, and a plain row that holds a secret, are refused
	bad := secretRow("bad-both")
	bad.ValueJSON = []byte(`"x"`)
	if err := s.Insert(ctx, bad, hist("bad-both")); err == nil {
		t.Errorf("Wrong match. expect: the CHECK constraint to refuse a secret row with a plain value")
	}
	bad2 := &confighandler.Row{Key: "bad-plain", ValueType: "string", ValueJSON: []byte(`"x"`), SecretCiphertext: []byte("0123456789012345678901234567890123456789"), KeyVersion: 1, Revision: 1, TMCreate: now, TMUpdate: now}
	if err := s.Insert(ctx, bad2, hist("bad-plain")); err == nil {
		t.Errorf("Wrong match. expect: the CHECK constraint to refuse a plain row with a secret")
	}

	// a history row of a secret that carries a value is refused, and the row insert is undone
	h := hist("bad-hist")
	h.NewValueJSON = []byte(`"leak"`)
	if err := s.Insert(ctx, secretRow("bad-hist"), h); err == nil {
		t.Errorf("Wrong match. expect: the CHECK constraint to refuse a secret history row with a value")
	}
	if _, err := s.Get(ctx, "bad-hist"); !errors.Is(err, confighandler.ErrNotFound) {
		t.Errorf("Wrong match. the row stayed after the history insert failed: %v", err)
	}
}

// Test_configStore_mysql_endToEnd runs the real handler on the real store: write, rotate, read
// back, clear, and check that the stored bytes are not the plaintext and the history holds no value.
func Test_configStore_mysql_endToEnd(t *testing.T) {
	db := openMySQLForConfigs(t)
	ctx := context.Background()

	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(0x50 + i)
	}
	crypto, err := secretcrypto.New("1:"+base64.StdEncoding.EncodeToString(raw), secretcrypto.WithMaxPlaintext(2020))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	reg, err := configkeys.NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	env := func(name string) (string, bool) { return "", false }
	h := confighandler.NewHandler(reg, NewConfigStore(db), crypto, env, configkeys.Table, confighandler.WithSource("cli"))

	const value = "synthetic-credential-value"
	actor := uuid.Must(uuid.NewV4())

	if _, err := h.SetSecret(ctx, actor, configkeys.KeyMailgunAPIKey, value); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	secret, res, err := h.Secret(ctx, configkeys.KeyMailgunAPIKey)
	if err != nil || secret.Reveal() != value || res.Source != confighandler.SourceDB || res.Revision != 1 {
		t.Fatalf("Wrong match. got: %v %+v", err, res)
	}

	if _, err := h.SetSecret(ctx, actor, configkeys.KeyMailgunAPIKey, value+"-2"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	secret, res, _ = h.Secret(ctx, configkeys.KeyMailgunAPIKey)
	if secret.Reveal() != value+"-2" || res.Revision != 2 {
		t.Errorf("Wrong match. got: %+v", res)
	}

	var blob []byte
	if err := db.QueryRow("select secret_ciphertext from email_configs where config_key = ?", configkeys.KeyMailgunAPIKey).Scan(&blob); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if bytes.Contains(blob, []byte(value)) {
		t.Errorf("Wrong match. the stored bytes hold the plaintext")
	}

	if _, err := h.Clear(ctx, actor, configkeys.KeyMailgunAPIKey); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if _, res, _ := h.Secret(ctx, configkeys.KeyMailgunAPIKey); res.Source != confighandler.SourceNone {
		t.Errorf("Wrong match. expect: none after clear, got: %s", res.Source)
	}

	var histories, withValue int
	_ = db.QueryRow("select count(*), sum(old_value_json is not null or new_value_json is not null) from email_config_histories").Scan(&histories, &withValue)
	if histories != 3 || withValue != 0 {
		t.Errorf("Wrong match. expect: 3 history rows without a value, got: %d, %d", histories, withValue)
	}
}
