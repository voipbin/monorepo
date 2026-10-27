package main

import (
	"os"
	"strings"
	"testing"
)

func Test_main_help(t *testing.T) {
	savedArgs := os.Args
	t.Cleanup(func() { os.Args = savedArgs })
	os.Args = []string{"tag-control", "--help"}

	out := captureStdout(t, main)
	for _, expect := range []string{"Usage:", "tag-control", "Voipbin Tag Management CLI"} {
		if !strings.Contains(out, expect) {
			t.Errorf("The help output does not contain %q. got: %s", expect, out)
		}
	}
}
