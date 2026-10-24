package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func Test_rootCmdFlags(t *testing.T) {
	tests := []struct {
		name string

		expectDefault string
	}{
		{name: "database_dsn", expectDefault: "testid:testpassword@tcp(127.0.0.1:3306)/test"},
		{name: "prometheus_endpoint", expectDefault: "/metrics"},
		{name: "prometheus_listen_address", expectDefault: ":2112"},
		{name: "ssl_privkey_base64", expectDefault: ""},
		{name: "ssl_cert_base64", expectDefault: ""},
		{name: "paddle_webhook_secret_key", expectDefault: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := rootCmd.Flags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("flag %s is not defined", tt.name)
			}
			if f.DefValue != tt.expectDefault {
				t.Errorf("Wrong default. expect: %q, got: %q", tt.expectDefault, f.DefValue)
			}
		})
	}

	rabbit := rootCmd.Flags().Lookup("rabbitmq_address")
	if rabbit == nil || !strings.HasPrefix(rabbit.DefValue, "amqp://") || !strings.HasSuffix(rabbit.DefValue, "@localhost:5672") {
		t.Errorf("rabbitmq_address flag must default to a local amqp address")
	}

	if rootCmd.Use != "hook-manager" {
		t.Errorf("Wrong match. expect: hook-manager, got: %s", rootCmd.Use)
	}
}

func Test_writeBase64(t *testing.T) {
	tests := []struct {
		name string

		data string

		expectContent string
	}{
		{name: "decoded content is written", data: base64.StdEncoding.EncodeToString([]byte("fake-pem-content\nline2\n")), expectContent: "fake-pem-content\nline2\n"},
		{name: "empty data creates an empty file", data: "", expectContent: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "out.pem")

			if err := writeBase64(filename, tt.data); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			got, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("the file was not created: %v", err)
			}
			if string(got) != tt.expectContent {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectContent, string(got))
			}
		})
	}

	t.Run("file can not be created", func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), "missing-dir", "out.pem")

		if err := writeBase64(filename, "ZmFrZQ=="); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_writeBase64_logs(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("neutral-payload-text"))
	filename := filepath.Join(t.TempDir(), "missing-dir", "out.pem")

	std := logrus.StandardLogger()
	prevHooks := std.ReplaceHooks(make(logrus.LevelHooks))
	prevLevel := std.GetLevel()
	std.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		std.ReplaceHooks(prevHooks)
		std.SetLevel(prevLevel)
	})
	hook := logrustest.NewLocal(std)

	if err := writeBase64(filename, payload); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	entries := hook.AllEntries()
	if len(entries) < 1 {
		t.Fatalf("Wrong match. expect: at least 1 log entry, got: %d", len(entries))
	}

	for _, e := range entries {
		if strings.Contains(e.Message, payload) {
			t.Errorf("the log message contains the input data. message: %s", e.Message)
		}
		if _, ok := e.Data["data"]; ok {
			t.Errorf("the log entry has a data field")
		}
		for k, v := range e.Data {
			if strings.Contains(fmt.Sprint(v), payload) {
				t.Errorf("the log field contains the input data. field: %s", k)
			}
		}
	}
}

var (
	initPromOnce sync.Once
	initPromAddr string
)

func Test_initProm(t *testing.T) {
	// initProm registers on http.DefaultServeMux and starts a listener that can
	// not be stopped, so it must run once per test process (-count=N safe).
	initPromOnce.Do(func() {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("could not reserve a loopback port: %v", err)
		}
		initPromAddr = l.Addr().String()
		_ = l.Close()

		initProm("/metrics", initPromAddr)
	})
	addr := initPromAddr

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(8 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		res, errGet := client.Get("http://" + addr + "/metrics")
		if errGet != nil {
			lastErr = errGet
			time.Sleep(100 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("Wrong status. expect: 200, got: %d", res.StatusCode)
		}
		if !strings.Contains(string(body), "go_goroutines") {
			t.Errorf("metrics body does not look like prometheus output")
		}
		return
	}
	t.Fatalf("the prometheus endpoint did not come up in time. last error: %v", lastErr)
}
