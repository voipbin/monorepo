package main

import (
	"os"
	"strings"
	"testing"
)

func Test_main_help(t *testing.T) {
	orig := os.Args
	t.Cleanup(func() { os.Args = orig })
	os.Args = []string{"pipecat-control", "--help"}

	// the help flag ends the command before any connection is opened, so main returns normally.
	out := captureStdout(t, main)

	if !strings.Contains(out, "Voipbin Pipecat Management CLI") {
		t.Errorf("Wrong output. expect: the command description, got: %q", out)
	}
	for _, sub := range []string{"pipecatcall", "migrate"} {
		if !strings.Contains(out, sub) {
			t.Errorf("Wrong output. expect: the %q command listed, got: %q", sub, out)
		}
	}
}
