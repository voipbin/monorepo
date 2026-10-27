package main

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// main builds the command tree from the process arguments and runs it. With --help it prints the
// usage and returns, so it can run in a test without a database.
func Test_main_help(t *testing.T) {
	// environment variables that viper.AutomaticEnv would otherwise map onto the flags
	for _, key := range []string{"TYPE", "NAME", "DETAIL", "ID", "TOKEN", "CHAT_ID", "OWNER_TYPE", "OWNER_ID", "EMOJI", "TEXT", "SIZE"} {
		t.Setenv(key, "")
	}

	viper.Reset()
	t.Cleanup(viper.Reset)

	origArgs := os.Args
	os.Args = []string{"talk-control", "--help"}
	t.Cleanup(func() { os.Args = origArgs })

	out := captureStdout(t, main)

	for _, want := range []string{"talk-control", "chat", "message", "participant", "reaction", "migrate"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output does not contain %q: %s", want, out)
		}
	}
}
