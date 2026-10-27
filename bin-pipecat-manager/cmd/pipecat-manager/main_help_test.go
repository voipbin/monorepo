package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func Test_main_help(t *testing.T) {
	orig := os.Args
	origStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = orig
		os.Stdout = origStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	os.Stdout = w
	os.Args = []string{"pipecat-manager", "--help"}

	// the help flag ends the command before the daemon starts, so main returns normally.
	main()

	os.Stdout = origStdout
	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the output. err: %v", err)
	}

	if !strings.Contains(string(out), "Voipbin Pipecat Manager Daemon") {
		t.Errorf("Wrong output. expect: the command description, got: %q", string(out))
	}
}
