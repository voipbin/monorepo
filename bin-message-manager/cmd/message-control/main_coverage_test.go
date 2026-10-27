package main

import (
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"monorepo/bin-message-manager/internal/config"
)

// Test_main_help runs main with the help flag only. The help flag is handled before the persistent
// pre-run hook, so the config loader does not run. Any other argument can end the test binary through
// log.Fatalf, which hides the real failure.
func Test_main_help(t *testing.T) {
	resetControlState(t)

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"message-control", "--help"}

	out := captureStdout(t, main)

	for _, expect := range []string{"message-control [command]", "message", "migrate"} {
		if !strings.Contains(out, expect) {
			t.Errorf("Wrong match. expect: help output contains %q, got: %s", expect, out)
		}
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connects to the cache", func(t *testing.T) {
		resetControlState(t)

		mr := miniredis.RunT(t)
		config.Get().RedisAddress = mr.Addr()
		config.Get().RedisPassword = ""
		config.Get().RedisDatabase = 0

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Fatalf("Wrong match. expect: cache handler, got: nil")
		}
	})

	t.Run("cache is not reachable", func(t *testing.T) {
		resetControlState(t)

		// port 0 can never be connected to, so no external state is involved.
		config.Get().RedisAddress = "127.0.0.1:0"
		config.Get().RedisPassword = ""
		config.Get().RedisDatabase = 0

		res, err := initCache()
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if err == nil || !strings.HasPrefix(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong match. expect: could not connect to the cache, got: %v", err)
		}
	})
}
