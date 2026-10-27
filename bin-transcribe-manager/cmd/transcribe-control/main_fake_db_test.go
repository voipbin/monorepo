package main

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"monorepo/bin-transcribe-manager/internal/config"
)

// Test_initHandler_cacheError checks that initHandler stops at the cache step, before RabbitMQ, when
// the database is reachable but the cache is not.
func Test_initHandler_cacheError(t *testing.T) {
	saveConfig(t)
	config.Get().DatabaseDSN = "tester@tcp(" + startFakeMySQL(t) + ")/transcribe_test"
	config.Get().RedisAddress = closedAddr(t)

	res, err := initHandler()
	if err == nil {
		t.Fatalf("initHandler() expected an error")
	}
	if res != nil {
		t.Errorf("initHandler() expected a nil handler on error. got: %v", res)
	}
	if !strings.Contains(err.Error(), "could not initialize the cache") {
		t.Errorf("initHandler() error was not wrapped. err: %v", err)
	}
	if strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("initHandler() expected the database connection to succeed. err: %v", err)
	}
}

// Test_main_help runs main through the help command, which prints the usage and returns.
func Test_main_help(t *testing.T) {
	saveConfig(t)
	t.Cleanup(viper.Reset)

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"transcribe-control", "help"}

	out := captureStdout(t, main)

	for _, want := range []string{"Voipbin Transcribe Management CLI", "transcribe", "migrate"} {
		if !strings.Contains(out, want) {
			t.Errorf("Wrong match. expect the usage to contain %q, got: %q", want, out)
		}
	}
}
