package main

import (
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-transfer-manager/internal/config"
)

const testUnreachableDSN = "tester@unix(/nonexistent/transfer-control-test.sock)/test"

// setConfig replaces the global config for a test and restores it afterwards.
func setConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	saved := *config.Get()
	t.Cleanup(func() { *config.Get() = saved })

	*config.Get() = cfg
}

// resetViper clears the global viper before a test and again afterwards.
func resetViper(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)
}

func Test_resolveString(t *testing.T) {
	tests := []struct {
		name string

		setValue string
		setKey   bool

		expectRes string
		expectErr string
	}{
		{
			name:      "set",
			setKey:    true,
			setValue:  "attended",
			expectRes: "attended",
		},
		{
			name:      "empty value",
			setKey:    true,
			setValue:  "",
			expectErr: "Transfer type is required",
		},
		{
			name:      "key not set",
			expectErr: "Transfer type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			if tt.setKey {
				viper.Set("transfer-type", tt.setValue)
			}

			res, err := resolveString("transfer-type", "Transfer type")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Fatalf("expected error %q, got: %v", tt.expectErr, err)
				}
				if res != "" {
					t.Errorf("expected empty result on error, got: %q", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("expected: %q, got: %q", tt.expectRes, res)
			}
		})
	}
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name string

		value string

		expectRes uuid.UUID
		expectErr string
	}{
		{
			name:      "valid",
			value:     "5a6d1c2e-0001-11ee-9a11-000000000001",
			expectRes: uuid.FromStringOrNil("5a6d1c2e-0001-11ee-9a11-000000000001"),
		},
		{
			name:      "empty",
			value:     "",
			expectRes: uuid.Nil,
			expectErr: "Groupcall ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Groupcall ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:      "nil uuid string is rejected as invalid",
			value:     "00000000-0000-0000-0000-000000000000",
			expectRes: uuid.Nil,
			expectErr: "invalid format for Groupcall ID: '00000000-0000-0000-0000-000000000000' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("groupcall-id", tt.value)

			res, err := resolveUUID("groupcall-id", "Groupcall ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Fatalf("expected error %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("expected: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}

	saved := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = saved })

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	f()

	_ = w.Close()
	os.Stdout = saved

	return <-done
}

func Test_printJSON(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name string

		value any

		expectOut string
		expectErr string
	}{
		{
			name:      "struct is printed with two space indent",
			value:     &sample{Name: "alpha", Count: 2},
			expectOut: "{\n  \"name\": \"alpha\",\n  \"count\": 2\n}\n",
		},
		{
			name:      "unsupported value",
			value:     make(chan int),
			expectOut: "",
			expectErr: "failed to marshal JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			out := captureStdout(t, func() {
				err = printJSON(tt.value)
			})

			if tt.expectErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectErr) {
					t.Fatalf("expected error prefix %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if out != tt.expectOut {
				t.Errorf("expected output: %q, got: %q", tt.expectOut, out)
			}
		})
	}
}

func Test_cmdFlags(t *testing.T) {
	tests := []struct {
		name string

		newCmd func() *cobra.Command

		expectUse   string
		expectFlags []string
	}{
		{
			name:        "service-start",
			newCmd:      cmdServiceStart,
			expectUse:   "service-start",
			expectFlags: []string{"transfer-type", "transferer-call-id", "transferee-addresses"},
		},
		{
			name:        "get-by-groupcall",
			newCmd:      cmdGetByGroupcall,
			expectUse:   "get-by-groupcall",
			expectFlags: []string{"groupcall-id"},
		},
		{
			name:        "get-by-call",
			newCmd:      cmdGetByCall,
			expectUse:   "get-by-call",
			expectFlags: []string{"call-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.newCmd()
			if cmd.Use != tt.expectUse {
				t.Errorf("expected use: %q, got: %q", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("expected a run function on %q", tt.expectUse)
			}

			usageLines := strings.Split(strings.TrimSpace(cmd.Flags().FlagUsages()), "\n")
			if len(usageLines) != len(tt.expectFlags) {
				t.Errorf("expected %d flags, got: %d", len(tt.expectFlags), len(usageLines))
			}

			for _, name := range tt.expectFlags {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("flag %q is not defined", name)
					continue
				}
				if f.DefValue != "" {
					t.Errorf("flag %q: expected an empty default, got: %q", name, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	// Bootstrap switches the global logrus formatter and level.
	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
	})
	resetViper(t)

	root := initCommand()
	if root.Use != "transfer-control" {
		t.Errorf("expected root use: %q, got: %q", "transfer-control", root.Use)
	}
	if root.PersistentPreRunE == nil {
		t.Errorf("expected a pre-run hook on the root command")
	}

	// the infrastructure flags of the config package are bound on the root
	for _, name := range []string{
		"rabbitmq_address",
		"database_dsn",
		"redis_address",
		"redis_database",
	} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("persistent flag %q is not defined", name)
		}
	}

	sub, _, err := root.Find([]string{"transfer"})
	if err != nil || sub == nil || sub.Use != "transfer" {
		t.Fatalf("expected the transfer command, got: %v, err: %v", sub, err)
	}

	names := []string{}
	for _, c := range sub.Commands() {
		names = append(names, c.Use)
	}
	expectNames := map[string]bool{"service-start": true, "get-by-groupcall": true, "get-by-call": true}
	if len(names) != len(expectNames) {
		t.Fatalf("expected %d sub commands, got: %v", len(expectNames), names)
	}
	for _, n := range names {
		if !expectNames[n] {
			t.Errorf("unexpected sub command: %q", n)
		}
	}
}

func Test_run_initHandlerFailure(t *testing.T) {
	tests := []struct {
		name string

		run func(cmd *cobra.Command, args []string) error
	}{
		{name: "service-start", run: runServiceStart},
		{name: "get-by-groupcall", run: runGetByGroupcall},
		{name: "get-by-call", run: runGetByCall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setConfig(t, config.Config{DatabaseDSN: testUnreachableDSN})

			err := tt.run(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}

			msg := err.Error()
			for _, part := range []string{
				"failed to initialize handlers",
				"could not connect to the database",
				"database ping error",
			} {
				if !strings.Contains(msg, part) {
					t.Errorf("expected %q in the error, got: %s", part, msg)
				}
			}
			if !errors.Is(err, syscall.ENOENT) {
				t.Errorf("expected the cause to be a missing socket, got: %v", err)
			}
		})
	}
}

func Test_initHandler_databaseFailure(t *testing.T) {
	setConfig(t, config.Config{DatabaseDSN: testUnreachableDSN})

	res, err := initHandler()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if res != nil {
		t.Errorf("expected no handler on error, got: %v", res)
	}
	if !strings.HasPrefix(err.Error(), "could not connect to the database") {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		mr := miniredis.RunT(t)
		setConfig(t, config.Config{RedisAddress: mr.Addr()})

		res, err := initCache()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatalf("expected a cache handler, got nil")
		}
		if mr.TotalConnectionCount() < 1 {
			t.Errorf("expected the cache server to be connected, count: %d", mr.TotalConnectionCount())
		}
	})

	t.Run("server is down", func(t *testing.T) {
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()
		setConfig(t, config.Config{RedisAddress: addr})

		res, err := initCache()
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}
		if res != nil {
			t.Errorf("expected no handler on error, got: %v", res)
		}
		if !strings.HasPrefix(err.Error(), "could not connect to the cache") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
