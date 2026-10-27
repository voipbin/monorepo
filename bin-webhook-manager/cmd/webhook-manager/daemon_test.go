package main

import (
	"io"
	"net"
	"os"
	"os/signal"
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

func Test_runDaemon_databaseUnreachable(t *testing.T) {
	setConfig(t, config.Config{DatabaseDSN: "tcp(" + closedAddr(t) + ")/test"})

	// drain a left-over done signal and unregister the handler runDaemon installs.
	select {
	case <-chDone:
	default:
	}
	t.Cleanup(func() {
		signal.Stop(chSigs)
		select {
		case <-chDone:
		default:
		}
	})

	err := runDaemon()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect error containing: could not connect to the database, got: %v", err)
	}
}

func Test_initCache(t *testing.T) {
	t.Run("server unreachable", func(t *testing.T) {
		setConfig(t, config.Config{RedisAddress: closedAddr(t)})

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "cache connect error") {
			t.Errorf("Wrong match. expect error containing: cache connect error, got: %v", err)
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

// Test_main_help runs main with --help: cobra prints the usage and returns before the daemon starts.
func Test_main_help(t *testing.T) {
	origArgs := os.Args
	origOut := os.Stdout
	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	viper.Reset()
	t.Cleanup(func() {
		os.Args = origArgs
		os.Stdout = origOut
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
		viper.Reset()
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Could not create a pipe. err: %v", err)
	}
	os.Stdout = w
	os.Args = []string{"webhook-manager", "--help"}

	main()

	os.Stdout = origOut
	if errClose := w.Close(); errClose != nil {
		t.Fatalf("Could not close the pipe. err: %v", errClose)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Could not read the pipe. err: %v", err)
	}

	for _, expect := range []string{"Voipbin Webhook Manager Daemon", "--database_dsn", "--rabbitmq_address", "--redis_address"} {
		if !strings.Contains(string(out), expect) {
			t.Errorf("Wrong match. expect usage containing: %q, got: %q", expect, string(out))
		}
	}
}
