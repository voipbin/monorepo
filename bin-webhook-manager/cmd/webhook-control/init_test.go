package main

import (
	"net"
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"monorepo/bin-webhook-manager/internal/config"
)

// setConfig replaces the process-wide config for one test and restores it afterwards.
func setConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	orig := *config.Get()
	*config.Get() = cfg
	t.Cleanup(func() { *config.Get() = orig })
}

// closedAddr returns a loopback address on which nothing listens, so a connection is refused at once.
func closedAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not reserve a loopback port. err: %v", err)
	}
	addr := l.Addr().String()
	if errClose := l.Close(); errClose != nil {
		t.Fatalf("Could not release the loopback port. err: %v", errClose)
	}

	return addr
}

func Test_initHandler_databaseUnreachable(t *testing.T) {
	setConfig(t, config.Config{DatabaseDSN: "tcp(" + closedAddr(t) + ")/test"})

	res, err := initHandler()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect error containing: could not connect to the database, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
}

func Test_initCache(t *testing.T) {
	t.Run("server unreachable", func(t *testing.T) {
		setConfig(t, config.Config{RedisAddress: closedAddr(t)})

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong match. expect error containing: could not connect to the cache, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil handler, got: %v", res)
		}
	})

	t.Run("server reachable", func(t *testing.T) {
		mr := miniredis.RunT(t)
		setConfig(t, config.Config{RedisAddress: mr.Addr()})

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Errorf("Wrong match. expect: a handler, got: nil")
		}
	})
}

// Test_runSend_initHandlerFails checks that valid input reaches the handler setup and that a setup
// failure is reported without sending anything. The commands are called directly, not through
// Execute, so the config set here is not replaced by the loading of the global config.
func Test_runSend_initHandlerFails(t *testing.T) {
	const validID = "5a0b3a3c-9d0b-11ee-8f4f-0b0e2b1c2d3e"

	tests := []struct {
		name string

		run func() error
		set map[string]string
	}{
		{
			name: "send-to-customer",
			run:  func() error { return runSendToCustomer(nil, nil) },
			set:  map[string]string{"customer-id": validID, "data": `{"type":"call_updated"}`, "data-type": "application/json"},
		},
		{
			name: "send-to-uri",
			run:  func() error { return runSendToURI(nil, nil) },
			set: map[string]string{
				"customer-id": validID, "uri": "https://example.com/hook", "method": "POST",
				"data": `{"type":"call_updated"}`, "data-type": "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			setConfig(t, config.Config{DatabaseDSN: "tcp(" + closedAddr(t) + ")/test"})
			for k, v := range tt.set {
				viper.Set(k, v)
			}

			var err error
			out := captureStdout(t, func() { err = tt.run() })

			if err == nil || !strings.Contains(err.Error(), "failed to initialize handlers") || !strings.Contains(err.Error(), "could not connect to the database") {
				t.Errorf("Wrong match. expect error containing: failed to initialize handlers, could not connect to the database, got: %v", err)
			}
			if out != "" {
				t.Errorf("Wrong match. expect: nothing printed, got: %q", out)
			}
		})
	}
}

// Test_main_help runs main with --help: cobra prints the usage and returns before any handler is built.
func Test_main_help(t *testing.T) {
	origArgs := os.Args
	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	resetViper(t)
	t.Cleanup(func() {
		os.Args = origArgs
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})

	os.Args = []string{"webhook-control", "--help"}
	out := captureStdout(t, main)

	for _, expect := range []string{"Voipbin Webhook Management CLI", "webhook", "--database_dsn"} {
		if !strings.Contains(out, expect) {
			t.Errorf("Wrong match. expect usage containing: %q, got: %q", expect, out)
		}
	}
}
