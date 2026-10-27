package main

import (
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"monorepo/bin-schedule-manager/internal/config"
)

func Test_main_help(t *testing.T) {
	// restores viper, the logrus settings and the process config afterwards
	useTestConfig(t, invalidDSNConfig())

	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"schedule-control", "--help"}

	out := captureStdout(t, main)

	for _, expect := range []string{"Voipbin Schedule Management CLI", "database_dsn", "schedule", "execution", "migrate"} {
		if !strings.Contains(out, expect) {
			t.Errorf("Wrong match. expect: the usage contains %q, got: %q", expect, out)
		}
	}
}

// Test_initCache_unreachable connects to the address of a server that was just
// stopped. Client retries make a failed connect slow, so this is the only
// failure case.
func Test_initCache_unreachable(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	useTestConfig(t, config.Config{RedisAddress: addr})

	cache, err := initCache()
	if err == nil || cache != nil {
		t.Fatalf("Wrong match. expect: error and nil cache, got: %v, %v", cache, err)
	}
	if !strings.Contains(err.Error(), "could not connect to the cache") {
		t.Errorf("Wrong match. expect: the cache connect error, got: %v", err)
	}
}
