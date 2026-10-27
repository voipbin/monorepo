package main

import (
	"testing"

	"monorepo/bin-outdial-manager/internal/config"
)

// Test_createDBHandler_databaseUnreachable uses a unix socket that does not exist, so the
// connection fails at once without any database or network. The test fixes the contract that a
// failed start returns a nil handler and an error; the cache step is never reached.
func Test_createDBHandler_databaseUnreachable(t *testing.T) {
	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })
	cfg.DatabaseDSN = "test-user@unix(/nonexistent-test-dir/none.sock)/test_db"

	res, err := createDBHandler()
	if res != nil {
		t.Errorf("expected no handler, got: %v", res)
	}
	if err == nil {
		t.Errorf("expected the connection error, got nil")
	}
}
