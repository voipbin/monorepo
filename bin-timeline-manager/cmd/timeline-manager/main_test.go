package main

import (
	"testing"
)

func Test_runMySQLMigrations(t *testing.T) {
	tests := []struct {
		name string

		dsn string

		expectErr bool
	}{
		{
			name:      "malformed dsn returns an error",
			dsn:       "not-a-dsn",
			expectErr: true,
		},
		{
			name:      "unreachable database returns an error",
			dsn:       "user:pass@tcp(127.0.0.1:1)/test",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runMySQLMigrations(tt.dsn)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
