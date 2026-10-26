package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func Test_main_help(t *testing.T) {
	resetViper(t)

	origArgs := os.Args
	origStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = origArgs
		os.Stdout = origStdout
	})

	r, w, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("could not create a pipe: %v", errPipe)
	}
	outCh := make(chan string, 1)
	go func() {
		out, _ := io.ReadAll(r)
		outCh <- string(out)
	}()

	os.Args = []string{"hook-control", "--help"}
	os.Stdout = w
	main()
	os.Stdout = origStdout
	_ = w.Close()
	out := <-outCh

	for _, name := range []string{"hook-control", "send-email", "send-message", "send-conversation"} {
		if !strings.Contains(out, name) {
			t.Errorf("help output does not mention %q. got: %q", name, out)
		}
	}
}
