package main

import (
	"strings"
	"testing"

	"monorepo/bin-rag-manager/internal/config"
)

func Test_validatePostgreSQLDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string

		expectErr bool
	}{
		{"postgres url", "postgres://user:SENTINEL@db.example:5432/db?sslmode=disable", false},
		{"postgresql url", "postgresql://user:SENTINEL@db.example/db", false},
		{"bad port", "postgres://user:SENTINEL@db.example:99x/db", true},
		// SENTINEL%zz is the only reason this DSN does not parse. Do not tidy the value.
		{"bad escape in password", "postgres://user:SENTINEL%zz@db.example/db", true},
		{"keyword style", "host=db.example password=SENTINEL:x", true},
		{"opaque", "postgres:user:SENTINEL@db.example/db", true},
		{"upper case scheme", "POSTGRES://user:SENTINEL@db.example/db", true},
		{"leading space", " postgres://user:SENTINEL@db.example/db", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostgreSQLDSN(tt.dsn)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if err != nil && strings.Contains(err.Error(), "SENTINEL") {
				t.Errorf("error contains the credential: %v", err)
			}
		})
	}
}

// Test_runMigrations_dsn_no_credentials checks the error that run() returns and logs. The test
// moves to the service root because runMigrations opens file://./migrations relative to the
// working directory and golang-migrate opens that source before the database URL: without the
// directory the error never contains the DSN, even without the validation.
func Test_runMigrations_dsn_no_credentials(t *testing.T) {
	t.Chdir("../..")

	tests := []struct {
		name string
		dsn  string
	}{
		{"bad port", "postgres://user:SENTINEL@db.example:99x/db"},
		{"bad escape in password", "postgres://user:SENTINEL%zz@db.example/db"},
		{"keyword style", "host=db.example password=SENTINEL:x"},
		{"opaque", "postgres:user:SENTINEL@db.example/db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runMigrations(config.Config{PostgreSQLDSN: tt.dsn})
			if err == nil {
				t.Fatalf("expected an error")
			}
			if strings.Contains(err.Error(), "SENTINEL") {
				t.Errorf("error contains the credential: %v", err)
			}
		})
	}
}
