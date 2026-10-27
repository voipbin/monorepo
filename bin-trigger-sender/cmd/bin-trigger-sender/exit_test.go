package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// main terminates the process through os.Exit on error, so these tests run it in a
// child process: the test binary re-executes itself with -test.run selecting only
// Test_main_exit_helper and a scenario variable. The child coverage is merged into
// the parent profile by go test. These tests must not call t.Parallel().

const (
	exitScenarioEnv = "BIN_TRIGGER_SENDER_EXIT_SCENARIO"

	// unparsable broker address: run fails while parsing it, with no network access.
	badBrokerAddr = "amqp://%zz"
)

var exitScenarios = map[string][]string{
	"missing_rabbit_addr": {"-queue", testQueue, "-uri", testURI, "-rabbit_addr", ""},
	"missing_queue":       {"-rabbit_addr", badBrokerAddr, "-uri", testURI},
	"missing_uri":         {"-rabbit_addr", badBrokerAddr, "-queue", testQueue},
	"invalid_data":        {"-rabbit_addr", badBrokerAddr, "-queue", testQueue, "-uri", testURI, "-data", "{"},
	"run_failure":         {"-rabbit_addr", badBrokerAddr, "-queue", testQueue, "-uri", testURI},
}

// Test_main_exit_helper is the child side. Without the scenario variable it does nothing.
func Test_main_exit_helper(t *testing.T) {
	args, ok := exitScenarios[os.Getenv(exitScenarioEnv)]
	if !ok {
		t.Skip("child process helper, runs only through Test_main_exit_codes")
	}

	runMain(t, args...)
}

func Test_main_exit_codes(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Could not get the test binary path. err: %v", err)
	}

	// an inherited scenario variable must not reach the child twice
	var baseEnv []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, exitScenarioEnv+"=") {
			baseEnv = append(baseEnv, e)
		}
	}

	tests := []struct {
		name         string
		wantContains string
		wantAbsent   string
	}{
		{"missing_rabbit_addr", "required: -rabbit_addr, -queue, -uri", "Failed:"},
		{"missing_queue", "required: -rabbit_addr, -queue, -uri", "Failed:"},
		{"missing_uri", "required: -rabbit_addr, -queue, -uri", "Failed:"},
		{"invalid_data", "invalid JSON in -data flag", "Failed:"},
		{"run_failure", "Failed: dial RabbitMQ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, exe, "-test.run=^Test_main_exit_helper$")
			cmd.Env = append(append([]string(nil), baseEnv...), exitScenarioEnv+"="+tt.name)
			cmd.WaitDelay = 5 * time.Second
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err := cmd.Run()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("Wrong match. expect: exit status 1, got: %v, stderr: %s", err, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantContains) {
				t.Errorf("Wrong match. expect stderr to contain: %q, got: %s", tt.wantContains, stderr.String())
			}
			if tt.wantAbsent != "" && strings.Contains(stderr.String(), tt.wantAbsent) {
				t.Errorf("Wrong match. expect stderr without: %q, got: %s", tt.wantAbsent, stderr.String())
			}
		})
	}
}
