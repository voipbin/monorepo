package main

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"monorepo/bin-rag-manager/internal/config"
)

// testBadDSN is rejected by net/url (a space in the password), and that parse error echoes the
// password back unmasked.
const testBadDSN = "postgres://dbuser:secret pass@db.internal:5432/rag"

func Test_redactDSNError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect string
	}{
		{
			name:   "nil",
			err:    nil,
			expect: "",
		},
		{
			name:   "invalid userinfo keeps the space",
			err:    errors.New(`parse "postgres://u:pa ss@h/db": net/url: invalid userinfo`),
			expect: `parse "postgres://***@h/db": net/url: invalid userinfo`,
		},
		{
			name:   "userinfo with an escaped quote from a real url.Parse error",
			err:    testURLParseError("postgres://u:pa\"ss@h/db"),
			expect: `parse "postgres://***@h/db": net/url: invalid userinfo`,
		},
		{
			name:   "no userinfo",
			err:    errors.New("dial tcp 127.0.0.1:1: connect: connection refused"),
			expect: "dial tcp 127.0.0.1:1: connect: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactDSNError(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", got)
				}
				return
			}
			if got.Error() != tt.expect {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expect, got.Error())
			}
		})
	}
}

func Test_openPostgres_doesNotLeakPassword(t *testing.T) {
	db, err := openPostgres(testBadDSN)
	if err == nil {
		_ = db.Close()
		t.Fatal("Expected an error for the malformed DSN")
	}

	for _, secret := range []string{"secret pass", "dbuser"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("openPostgres() error has %q. err: %v", secret, err)
		}
	}
}

func Test_runMigrations_doesNotLeakPassword(t *testing.T) {
	// runMigrations opens "./migrations" first, so give it an empty directory.
	t.Chdir(t.TempDir())
	if err := os.Mkdir("migrations", 0o700); err != nil {
		t.Fatalf("Could not create the migrations directory. err: %v", err)
	}

	err := runMigrations(config.Config{PostgreSQLDSN: testBadDSN})
	if err == nil {
		t.Fatal("Expected an error for the malformed DSN")
	}

	for _, secret := range []string{"secret pass", "dbuser"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("runMigrations() error has %q. err: %v", secret, err)
		}
	}
}

// testURLParseError returns the error of url.Parse for a malformed URL. The URL goes through a
// parameter so that static analysis does not flag the deliberately invalid literal.
func testURLParseError(raw string) error {
	_, err := url.Parse(raw)
	return err
}
