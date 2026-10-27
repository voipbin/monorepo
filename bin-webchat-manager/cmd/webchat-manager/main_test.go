package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"monorepo/bin-webchat-manager/internal/config"
)

// Test_main_help runs main with the help flag, which returns without starting
// the daemon. The package has no shared reset helper, so everything the
// configuration bootstrap touches is saved and restored here.
func Test_main_help(t *testing.T) {
	formatter := logrus.StandardLogger().Formatter
	level := logrus.GetLevel()
	saved := *config.Get()
	origArgs := os.Args
	origStdout := os.Stdout

	viper.Reset()
	t.Cleanup(func() {
		viper.Reset()
		logrus.SetFormatter(formatter)
		logrus.SetLevel(level)
		*config.Get() = saved
		os.Args = origArgs
		os.Stdout = origStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	os.Args = []string{"webchat-manager", "--help"}
	os.Stdout = w
	main() // the help output returns without exiting the process
	os.Stdout = origStdout
	_ = w.Close()
	out := <-done

	for _, expect := range []string{"webchat-manager", "--database_dsn", "--redis_address", "--prometheus_endpoint"} {
		if !strings.Contains(out, expect) {
			t.Errorf("Wrong output. expect: containing %q, got: %s", expect, out)
		}
	}
}
