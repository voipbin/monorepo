package main

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-tag-manager/internal/config"
)

var hostEnvKeys = []string{
	"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS", "RABBITMQ_ADDRESS",
	"REDIS_ADDRESS", "REDIS_DATABASE", "REDIS_PASSWORD",
	// keys that the sub-commands read through viper, which reads the host environment for any key
	"NAME", "ID", "TOKEN", "LIMIT", "DETAIL", "CUSTOMER_ID",
}

// isolateGlobals resets viper, removes host environment keys and restores the shared config.
func isolateGlobals(t *testing.T) {
	t.Helper()

	savedConfig := *config.Get()
	type savedEnv struct {
		key   string
		value string
		found bool
	}
	var saved []savedEnv
	for _, k := range hostEnvKeys {
		v, ok := os.LookupEnv(k)
		saved = append(saved, savedEnv{key: k, value: v, found: ok})
		_ = os.Unsetenv(k)
	}
	viper.Reset()

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = savedConfig
		for _, s := range saved {
			if s.found {
				_ = os.Setenv(s.key, s.value)
			} else {
				_ = os.Unsetenv(s.key)
			}
		}
	})
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	f()

	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectID  string
		expectErr string
	}{
		{
			name:      "empty value",
			value:     "",
			expectErr: "Tag ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectErr: "invalid format for Tag ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:     "valid uuid",
			value:    "11111111-2222-3333-4444-555555555555",
			expectID: "11111111-2222-3333-4444-555555555555",
		},
		{
			name:      "nil uuid is rejected",
			value:     "00000000-0000-0000-0000-000000000000",
			expectErr: "invalid format for Tag ID: '00000000-0000-0000-0000-000000000000' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGlobals(t)
			viper.Set("id", tt.value)

			res, err := resolveUUID("id", "Tag ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong error. expect: %s, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.String() != tt.expectID {
				t.Errorf("Wrong id. expect: %s, got: %s", tt.expectID, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indents with two spaces", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(map[string]any{"name": "tag-a", "count": 2})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expect := "{\n  \"count\": 2,\n  \"name\": \"tag-a\"\n}\n"
		if out != expect {
			t.Errorf("Wrong output. expect: %q, got: %q", expect, out)
		}
	})

	t.Run("unmarshalable value", func(t *testing.T) {
		var err error
		out := captureStdout(t, func() {
			err = printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong error. got: %v", err)
		}
		if out != "" {
			t.Errorf("nothing must be printed on error, got: %q", out)
		}
	})
}

func Test_initCommand_structure(t *testing.T) {
	isolateGlobals(t)

	root := initCommand()

	if root.Use != "tag-control" {
		t.Errorf("Wrong root use. got: %s", root.Use)
	}

	var rootNames []string
	for _, c := range root.Commands() {
		rootNames = append(rootNames, c.Name())
	}
	for _, expect := range []string{"tag", "migrate"} {
		if !contains(rootNames, expect) {
			t.Errorf("root command %s is missing. got: %v", expect, rootNames)
		}
	}

	var tagCmd, migrateCmd *cobra.Command
	for _, c := range root.Commands() {
		switch c.Name() {
		case "tag":
			tagCmd = c
		case "migrate":
			migrateCmd = c
		}
	}

	var tagNames []string
	for _, c := range tagCmd.Commands() {
		tagNames = append(tagNames, c.Name())
	}
	for _, expect := range []string{"create", "get", "list", "update", "delete"} {
		if !contains(tagNames, expect) {
			t.Errorf("tag sub-command %s is missing. got: %v", expect, tagNames)
		}
	}

	var migrateNames []string
	for _, c := range migrateCmd.Commands() {
		migrateNames = append(migrateNames, c.Name())
	}
	for _, expect := range []string{"status", "up", "down", "force"} {
		if !contains(migrateNames, expect) {
			t.Errorf("migrate sub-command %s is missing. got: %v", expect, migrateNames)
		}
	}

	flags := map[string]map[string]string{
		"create": {"customer-id": "", "name": "", "detail": ""},
		"get":    {"id": ""},
		"list":   {"limit": "100", "token": "", "customer-id": ""},
		"update": {"id": "", "name": "", "detail": ""},
		"delete": {"id": ""},
	}
	for _, c := range tagCmd.Commands() {
		expects, ok := flags[c.Name()]
		if !ok {
			continue
		}
		for flagName, def := range expects {
			f := c.Flags().Lookup(flagName)
			if f == nil {
				t.Errorf("%s: flag %s is missing", c.Name(), flagName)
				continue
			}
			if f.DefValue != def {
				t.Errorf("%s: wrong default for %s. expect: %q, got: %q", c.Name(), flagName, def, f.DefValue)
			}
		}
		if c.RunE == nil {
			t.Errorf("%s: RunE is not set", c.Name())
		}
	}

	// the infrastructure flags come from config.Bootstrap
	for _, name := range []string{"database_dsn", "redis_address", "rabbitmq_address"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("persistent flag %s is missing", name)
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func Test_runCreate_validation(t *testing.T) {
	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{
			name:      "customer id is missing",
			args:      []string{"tag", "create", "--name=tag-a"},
			expectErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:      "customer id is not a uuid",
			args:      []string{"tag", "create", "--customer-id=abc", "--name=tag-a"},
			expectErr: "failed to resolve customer ID: invalid format for Customer ID: 'abc' is not a valid UUID",
		},
		{
			name:      "name is missing",
			args:      []string{"tag", "create", "--customer-id=11111111-2222-3333-4444-555555555555"},
			expectErr: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGlobals(t)

			root := initCommand()
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(tt.args)

			err := root.Execute()
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong error. expect: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

// fakeRedis answers PING with PONG and records the commands. It is the smallest server that
// satisfies cachehandler.Connect.
type fakeRedis struct {
	ln net.Listener

	mu    sync.Mutex
	conns []net.Conn
	wg    sync.WaitGroup
}

func newFakeRedis(t *testing.T) *fakeRedis {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	f := &fakeRedis{ln: ln}
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			f.mu.Lock()
			f.conns = append(f.conns, conn)
			f.mu.Unlock()
			f.wg.Add(1)
			go func() {
				defer f.wg.Done()
				defer func() { _ = conn.Close() }()
				r := bufio.NewReader(conn)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					// the only command sent by Connect is "*1\r\n$4\r\nPING\r\n"
					if strings.HasPrefix(line, "*") {
						for i := 0; i < 2; i++ {
							if _, err := r.ReadString('\n'); err != nil {
								return
							}
						}
						if _, err := conn.Write([]byte("+PONG\r\n")); err != nil {
							return
						}
					}
				}
			}()
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		f.mu.Lock()
		for _, c := range f.conns {
			_ = c.Close()
		}
		f.mu.Unlock()
		f.wg.Wait()
	})
	return f
}

// injectConfig loads the given redis address into the shared config through config.InitConfig,
// which is the only exported setter (LoadGlobalConfig is guarded by a sync.Once).
func injectConfig(t *testing.T, redisAddress string) {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	for _, name := range []string{"database_dsn", "prometheus_endpoint", "prometheus_listen_address", "rabbitmq_address", "redis_password"} {
		cmd.Flags().String(name, "", "")
	}
	cmd.Flags().String("redis_address", redisAddress, "")
	cmd.Flags().Int("redis_database", 0, "")

	if err := config.InitConfig(cmd); err != nil {
		t.Fatalf("InitConfig returned an error: %v", err)
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connects to the configured server", func(t *testing.T) {
		isolateGlobals(t)
		srv := newFakeRedis(t)
		injectConfig(t, srv.ln.Addr().String())

		res, err := initCache()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Errorf("expected a cache handler")
		}
	})

	t.Run("unreachable server", func(t *testing.T) {
		isolateGlobals(t)

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("could not listen: %v", err)
		}
		addr := ln.Addr().String()
		_ = ln.Close()
		injectConfig(t, addr)

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong error. got: %v", err)
		}
		if res != nil {
			t.Errorf("expected a nil handler on failure, got: %v", res)
		}
	})
}
