package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"monorepo/bin-campaign-manager/internal/config"
)

// badDSN is rejected while it is parsed, so no connection is attempted.
const badDSN = "%%%bad"

func Test_initHandler_invalidDSN(t *testing.T) {
	isolateGlobals(t)
	*config.Get() = config.Config{DatabaseDSN: badDSN}

	res, err := initHandler()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: the database error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: no handler, got: %v", res)
	}
}

func Test_initCache(t *testing.T) {
	isolateGlobals(t)
	mr := miniredis.RunT(t)
	*config.Get() = config.Config{RedisAddress: mr.Addr()}

	t.Run("reachable cache", func(t *testing.T) {
		res, err := initCache()
		if err != nil || res == nil {
			t.Errorf("Wrong match. expect: a connected cache, got: %v, %v", res, err)
		}
	})

	t.Run("closed cache", func(t *testing.T) {
		mr.Close()

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong match. expect: the cache error, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: no cache, got: %v", res)
		}
	})
}

// Test_subcommands_stopAtInitialization runs every campaign sub-command with valid arguments
// against an invalid database address. Each one must stop with the initialization error.
func Test_subcommands_stopAtInitialization(t *testing.T) {
	const id = "0ca2ba7c-0000-4000-8000-000000000001"

	tests := []struct {
		name string
		args []string
	}{
		{"create", []string{"campaign", "create", "--customer-id=" + id, "--name=campaign-a", "--outplan-id=" + id, "--outdial-id=" + id}},
		{"get", []string{"campaign", "get", "--id=" + id}},
		{"list", []string{"campaign", "list", "--customer-id=" + id}},
		{"update-basic-info", []string{"campaign", "update-basic-info", "--id=" + id, "--name=campaign-b"}},
		{"update-status", []string{"campaign", "update-status", "--id=" + id, "--status=run"}},
		{"delete", []string{"campaign", "delete", "--id=" + id}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGlobals(t)
			*config.Get() = config.Config{DatabaseDSN: badDSN}

			root := initCommand()
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(tt.args)

			err := root.Execute()
			expect := "failed to initialize handlers: could not connect to the database"
			if err == nil || !strings.HasPrefix(err.Error(), expect) {
				t.Errorf("Wrong match. expect prefix: %s, got: %v", expect, err)
			}
		})
	}
}

func Test_main_help(t *testing.T) {
	isolateGlobals(t)

	savedArgs := os.Args
	t.Cleanup(func() { os.Args = savedArgs })
	os.Args = []string{"campaign-control", "--help"}

	out := captureStdout(t, main)
	if !strings.Contains(out, "Voipbin Campaign Management CLI") {
		t.Errorf("Wrong match. expect: the help text, got: %q", out)
	}
}
