package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-tag-manager/internal/config"
)

// metricsEndpointSeq makes every run() call register a different metrics path, because
// initProm registers on http.DefaultServeMux, which panics on a duplicate pattern.
var metricsEndpointSeq atomic.Int64

var runEnvKeys = []string{
	"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS", "RABBITMQ_ADDRESS",
	"REDIS_ADDRESS", "REDIS_DATABASE", "REDIS_PASSWORD",
}

// isolateRunGlobals removes the host environment keys that viper would read, and resets
// viper and the shared config when the test ends.
func isolateRunGlobals(t *testing.T) {
	t.Helper()

	savedConfig := *config.Get()
	for _, k := range runEnvKeys {
		// t.Setenv registers the restore of the original value; the unset follows it.
		t.Setenv(k, "")
		_ = os.Unsetenv(k)
	}
	viper.Reset()

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = savedConfig
	})
}

func newRunCommand(dsn, endpoint string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("database_dsn", dsn, "")
	cmd.Flags().String("prometheus_endpoint", endpoint, "")
	cmd.Flags().String("prometheus_listen_address", "127.0.0.1:0", "")
	cmd.Flags().String("rabbitmq_address", "", "")
	cmd.Flags().String("redis_address", "", "")
	cmd.Flags().Int("redis_database", 1, "")
	cmd.Flags().String("redis_password", "", "")

	return cmd
}

func Test_run_configError(t *testing.T) {
	isolateRunGlobals(t)

	// a command without the config flags cannot be bound
	err := run(&cobra.Command{Use: "test"}, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "failed to initialize config: error binding database_dsn flag") {
		t.Errorf("Wrong error. got: %v", err)
	}
}

func Test_run_databaseUnreachable(t *testing.T) {
	isolateRunGlobals(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	dsn := fmt.Sprintf("tcp(%s)/test", ln.Addr().String())
	_ = ln.Close()

	endpoint := fmt.Sprintf("/m-%d", metricsEndpointSeq.Add(1))

	err = run(newRunCommand(dsn, endpoint), nil)
	if err == nil || !strings.Contains(err.Error(), "database ping error") {
		t.Errorf("Wrong error. got: %v", err)
	}

	// the metrics handler is registered on the configured endpoint before the database is opened
	_, pattern := http.DefaultServeMux.Handler(&http.Request{Method: http.MethodGet, URL: &url.URL{Path: endpoint}})
	if pattern != endpoint {
		t.Errorf("The metrics endpoint is not registered. expect: %s, got: %q", endpoint, pattern)
	}
}
