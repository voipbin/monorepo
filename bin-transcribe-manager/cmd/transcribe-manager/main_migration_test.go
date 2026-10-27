package main

import (
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"monorepo/bin-transcribe-manager/internal/config"
)

// completionDirective matches the last line of cobra's completion output, for example ":0".
var completionDirective = regexp.MustCompile(`(^|\n):\d+$`)

// Test_runDaemon_migrationError checks that the daemon stops with the wrapped migration error when
// the database accepts the connection but the migrations cannot be applied, and that it does so
// before the cache or the broker.
func Test_runDaemon_migrationError(t *testing.T) {
	saveConfig(t)
	config.Get().PrometheusEndpoint = ""
	config.Get().PrometheusListenAddress = ""
	config.Get().DatabaseDSN = "tester@tcp(" + startFakeMySQL(t) + ")/transcribe_test"
	config.Get().RedisAddress = closedAddr(t)

	// runDaemon starts the signal goroutine; end it in cleanup so it does not leak into other tests.
	t.Cleanup(func() {
		chSigs <- os.Interrupt
		select {
		case <-chDone:
		case <-time.After(5 * time.Second):
			t.Error("the signal goroutine did not report done")
		}
		signal.Stop(chSigs)
	})

	err := runDaemon()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not apply the database migrations") {
		t.Errorf("Wrong match. expect: wrapped migration error, got: %v", err)
	}
	if strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: the connection to succeed, got: %v", err)
	}
}

// Test_main_completion runs main through the hidden completion command of cobra, which returns
// without starting the daemon.
func Test_main_completion(t *testing.T) {
	saveConfig(t)
	t.Cleanup(viper.Reset)

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"transcribe-manager", "__complete", ""}

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	main()

	os.Stdout = origStdout
	_ = w.Close()
	out, errRead := io.ReadAll(r)
	if errRead != nil {
		t.Fatalf("could not read the captured output. err: %v", errRead)
	}

	// the completion output ends with the directive line
	if !completionDirective.MatchString(strings.TrimSpace(string(out))) {
		t.Errorf("Wrong match. expect: the completion directive line, got: %q", out)
	}
}
