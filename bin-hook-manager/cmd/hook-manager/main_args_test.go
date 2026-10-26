package main

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func Test_main_help(t *testing.T) {
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

	os.Args = []string{"hook-manager", "--help"}
	os.Stdout = w
	main()
	os.Stdout = origStdout
	_ = w.Close()
	out := <-outCh

	if !strings.Contains(out, "Hook Manager handles webhook receivers") {
		t.Errorf("help output does not show the command description. got: %q", out)
	}
	if !strings.Contains(out, "hook-manager [flags]") {
		t.Errorf("help output does not show the usage line. got: %q", out)
	}
	for _, flag := range []string{"--database_dsn", "--rabbitmq_address", "--ssl_cert_base64"} {
		if !strings.Contains(out, flag) {
			t.Errorf("help output does not mention %q. got: %q", flag, out)
		}
	}
}

func Test_writeBase64_invalidBase64(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "out.pem")

	std := logrus.StandardLogger()
	prevExit := std.ExitFunc
	var exitCodes []int
	std.ExitFunc = func(code int) { exitCodes = append(exitCodes, code) }
	t.Cleanup(func() { std.ExitFunc = prevExit })

	// the input is not valid base64 (see base64.StdEncoding alphabet)
	if _, errDecode := base64.StdEncoding.DecodeString("!!not-base64!!"); errDecode == nil {
		t.Fatalf("the test input must be invalid base64")
	}

	err := writeBase64(filename, "!!not-base64!!")
	if err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}

	if len(exitCodes) != 1 || exitCodes[0] != 1 {
		t.Fatalf("Wrong match. expect exit codes: [1], got: %v", exitCodes)
	}

	got, errRead := os.ReadFile(filename)
	if errRead != nil {
		t.Fatalf("the file was not created: %v", errRead)
	}
	if len(got) != 0 {
		t.Errorf("nothing must be written for invalid input. got: %q", string(got))
	}
}
