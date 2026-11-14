package main

import (
	_ "github.com/mattn/go-sqlite3"

	"bytes"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/internal/transcribeconfig"
)

const testActor = "11111111-2222-4333-8444-555555555555"

func testMasterKeys() string {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(0x40 + i)
	}

	return "1:" + base64.StdEncoding.EncodeToString(raw)
}

// useConfigHarness replaces the handler, the stdin and the stdout of the config commands.
func useConfigHarness(t *testing.T, cfg *config.Config, stdin string) (*storetest.Memory, *bytes.Buffer) {
	t.Helper()

	store := storetest.NewMemory()
	h, err := transcribeconfig.New(cfg, store, confighandler.WithSource(historySourceCLI))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	out := &bytes.Buffer{}
	savedOpen, savedIn, savedOut := openConfigHandler, configIn, configOut
	openConfigHandler = func() (confighandler.Handler, error) { return h, nil }
	configIn = strings.NewReader(stdin)
	configOut = out
	t.Cleanup(func() { openConfigHandler, configIn, configOut = savedOpen, savedIn, savedOut })

	return store, out
}

func runConfigCommand(args ...string) error {
	cmd := cmdConfig()
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	return cmd.Execute()
}

func Test_config_set_list_clear(t *testing.T) {
	const secret = "synthetic-credential-value"

	// the variable that the service reads is set; the list of the stored settings does not show it
	t.Setenv("AWS_SECRET_KEY", "env-value")
	store, out := useConfigHarness(t, &config.Config{ConfigEncryptionKeys: testMasterKeys()}, secret+"\n")

	if err := runConfigCommand("set", "--key", "aws.secret_key", "--actor-id", testActor); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if strings.Contains(out.String(), secret) || !strings.Contains(out.String(), `"source": "db"`) {
		t.Errorf("Wrong match. set output: %s", out.String())
	}

	row, err := store.Get(nil, "aws.secret_key") //nolint:staticcheck // the in-memory store ignores the context
	if err != nil || bytes.Contains(row.SecretCiphertext, []byte(secret)) {
		t.Errorf("Wrong match. the stored row holds the plaintext or is missing: %v", err)
	}

	out.Reset()
	if err := runConfigCommand("list"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if strings.Contains(out.String(), secret) || strings.Contains(out.String(), "env-value") ||
		!strings.Contains(out.String(), testActor) {
		t.Errorf("Wrong match. list output: %s", out.String())
	}
	if strings.Contains(out.String(), `"env_shadowed": true`) || !strings.Contains(out.String(), `"env_shadowed": false`) ||
		strings.Contains(out.String(), "env_name") {
		t.Errorf("Wrong match. expect: env_shadowed false and no env_name, list output: %s", out.String())
	}

	out.Reset()
	if err := runConfigCommand("clear", "--key", "aws.secret_key", "--actor-id", testActor); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !strings.Contains(out.String(), `"source": "none"`) || strings.Contains(out.String(), `"source": "env"`) {
		t.Errorf("Wrong match. expect: source none after the clear, got: %s", out.String())
	}
}

// After a set or a clear, one line on stderr names the variable that is read and says that the stored
// value is not used. The output on stdout stays the JSON only.
func Test_config_noticeOnStderr(t *testing.T) {
	tests := []struct {
		name string
		args []string
		key  string
		env  string
	}{
		{name: "set access key", args: []string{"set", "--key", "aws.access_key", "--actor-id", testActor}, key: "aws.access_key", env: "AWS_ACCESS_KEY"},
		{name: "set secret key", args: []string{"set", "--key", "aws.secret_key", "--actor-id", testActor}, key: "aws.secret_key", env: "AWS_SECRET_KEY"},
		{name: "clear access key", args: []string{"clear", "--key", "aws.access_key", "--actor-id", testActor}, key: "aws.access_key", env: "AWS_ACCESS_KEY"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, out := useConfigHarness(t, &config.Config{ConfigEncryptionKeys: testMasterKeys()}, "synthetic-credential-value\n")
			errOut := &bytes.Buffer{}
			saved := configErr
			configErr = errOut
			t.Cleanup(func() { configErr = saved })

			if err := runConfigCommand(tt.args...); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			got := errOut.String()
			if strings.Count(got, "\n") != 1 || !strings.Contains(got, tt.env) || !strings.Contains(got, "not used") || strings.Contains(got, "synthetic-credential-value") {
				t.Errorf("Wrong match. expect: one line that names %s, got: %q", tt.env, got)
			}
			if strings.Contains(out.String(), tt.env) {
				t.Errorf("Wrong match. the notice must not be on stdout: %s", out.String())
			}
		})
	}

	t.Run("a failed set prints no notice", func(t *testing.T) {
		useConfigHarness(t, &config.Config{}, "v\n")
		errOut := &bytes.Buffer{}
		saved := configErr
		configErr = errOut
		t.Cleanup(func() { configErr = saved })

		if err := runConfigCommand("set", "--key", "aws.access_key", "--actor-id", testActor); err == nil {
			t.Fatalf("Wrong match. expect: an error")
		}
		if errOut.Len() != 0 {
			t.Errorf("Wrong match. expect: no notice, got: %q", errOut.String())
		}
	})
}

func Test_config_errors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
		cfg   *config.Config

		expectErr string
	}{
		{name: "set without key", args: []string{"set", "--actor-id", testActor}, cfg: &config.Config{}, expectErr: "key is required"},
		{name: "set without actor", args: []string{"set", "--key", "aws.secret_key"}, cfg: &config.Config{}, expectErr: "Actor ID is required"},
		{name: "set bad actor", args: []string{"set", "--key", "aws.secret_key", "--actor-id", "nope"}, cfg: &config.Config{}, expectErr: "not a valid UUID"},
		{name: "set unknown key", args: []string{"set", "--key", "no.such", "--actor-id", testActor}, cfg: &config.Config{}, expectErr: "unknown config key"},
		{name: "set without master key", args: []string{"set", "--key", "aws.secret_key", "--actor-id", testActor}, stdin: "v\n", cfg: &config.Config{}, expectErr: "failed to store the value"},
		{name: "set too long on stdin", args: []string{"set", "--key", "aws.secret_key", "--actor-id", testActor}, stdin: strings.Repeat("a", maxStdinBytes+1), cfg: &config.Config{ConfigEncryptionKeys: testMasterKeys()}, expectErr: "too long"},
		{name: "set too long for the key", args: []string{"set", "--key", "aws.secret_key", "--actor-id", testActor}, stdin: strings.Repeat("a", 513), cfg: &config.Config{ConfigEncryptionKeys: testMasterKeys()}, expectErr: "failed to store the value"},
		{name: "set empty", args: []string{"set", "--key", "aws.secret_key", "--actor-id", testActor}, stdin: "\n", cfg: &config.Config{ConfigEncryptionKeys: testMasterKeys()}, expectErr: "failed to store the value"},
		{name: "clear without key", args: []string{"clear", "--actor-id", testActor}, cfg: &config.Config{}, expectErr: "key is required"},
		{name: "clear without actor", args: []string{"clear", "--key", "aws.secret_key"}, cfg: &config.Config{}, expectErr: "Actor ID is required"},
		{name: "clear unknown key", args: []string{"clear", "--key", "no.such", "--actor-id", testActor}, cfg: &config.Config{}, expectErr: "failed to clear the value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, out := useConfigHarness(t, tt.cfg, tt.stdin)

			err := runConfigCommand(tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
			if out.Len() != 0 {
				t.Errorf("Wrong match. expect: no output, got: %s", out.String())
			}
			if err != nil && tt.stdin != "" && strings.Contains(err.Error(), strings.TrimSpace(tt.stdin)) && len(strings.TrimSpace(tt.stdin)) > 1 {
				t.Errorf("Wrong match. the error holds the value")
			}
		})
	}
}

func Test_config_openHandlerFails(t *testing.T) {
	useConfigHarness(t, &config.Config{}, "")
	openConfigHandler = func() (confighandler.Handler, error) { return nil, errors.New("database is down") }

	for _, args := range [][]string{
		{"list"},
		{"set", "--key", "aws.secret_key", "--actor-id", testActor},
		{"clear", "--key", "aws.secret_key", "--actor-id", testActor},
	} {
		if err := runConfigCommand(args...); err == nil || !strings.Contains(err.Error(), "failed to initialize handlers") {
			t.Errorf("Wrong match. %v expect: initialize error, got: %v", args, err)
		}
	}
}

func Test_config_listStoreFails(t *testing.T) {
	store, _ := useConfigHarness(t, &config.Config{}, "")
	store.ListErr = errors.New("database is down")

	if err := runConfigCommand("list"); err == nil || !strings.Contains(err.Error(), "failed to list") {
		t.Errorf("Wrong match. expect: list error, got: %v", err)
	}
}

func Test_readValue(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		expect    string
		expectErr bool
	}{
		{name: "plain", in: "abc", expect: "abc"},
		{name: "one newline", in: "abc\n", expect: "abc"},
		{name: "crlf", in: "abc\r\n", expect: "abc"},
		{name: "two newlines are refused", in: "abc\n\n", expectErr: true},
		{name: "inner newline is refused", in: "ab\ncd\n", expectErr: true},
		{name: "inner carriage return is refused", in: "ab\rcd", expectErr: true},
		{name: "exactly the limit", in: strings.Repeat("a", maxStdinBytes), expect: strings.Repeat("a", maxStdinBytes)},
		{name: "one over the limit", in: strings.Repeat("a", maxStdinBytes+1), expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readValue(strings.NewReader(tt.in))
			if tt.expectErr != (err != nil) || got != tt.expect {
				t.Errorf("Wrong match. expect: %q %v, got: %q %v", tt.expect, tt.expectErr, got, err)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func Test_readValue_readError(t *testing.T) {
	if _, err := readValue(failingReader{}); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_toOutput_neverHoldsAValue(t *testing.T) {
	actor := uuid.Must(uuid.NewV4())
	out := toOutput(&confighandler.Resolved{Key: "k", Kind: confighandler.KindSecret, Type: confighandler.TypeString, Source: confighandler.SourceDB, HasSecret: true, UpdatedBy: &actor, Revision: 3, KeyVersion: 1})
	if out.UpdatedBy == nil || *out.UpdatedBy != actor.String() || out.Revision != 3 || !out.HasSecret {
		t.Errorf("Wrong match. got: %+v", out)
	}
}

func Test_initConfigHandler(t *testing.T) {
	cfg := config.Get()
	saved := *cfg
	t.Cleanup(func() { *cfg = saved })
	cfg.DatabaseDSN = "synthetic-dsn"

	restore := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) {
		return sql.Open("sqlite3", ":memory:")
	})
	defer restore()

	h, err := initConfigHandler()
	if err != nil || h == nil {
		t.Errorf("Wrong match. expect: ok, got: %v %v", h, err)
	}

	restoreFail := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return nil, errors.New("connect refused") })
	defer restoreFail()
	if _, err := initConfigHandler(); err == nil || !strings.Contains(err.Error(), "could not connect") {
		t.Errorf("Wrong match. expect: connect error, got: %v", err)
	}
}

// The config group is reachable from the real root command and needs no Redis and no RabbitMQ: the
// root PersistentPreRunE only binds the flags and loads the configuration, and the commands open the
// database through openConfigHandler (replaced here by the harness).
func Test_config_throughTheRootCommand(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY", "env-access")
	_, out := useConfigHarness(t, &config.Config{}, "")

	root := initCommand()
	root.SetArgs([]string{"config", "list"})
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if !strings.Contains(out.String(), `"key": "aws.access_key"`) || !strings.Contains(out.String(), `"key": "aws.secret_key"`) ||
		!strings.Contains(out.String(), `"source": "none"`) || strings.Contains(out.String(), `"source": "env"`) || strings.Contains(out.String(), "env-access") {
		t.Errorf("Wrong match. list output: %s", out.String())
	}
}

// The master key list is never a flag of any command.
func Test_config_noMasterKeyFlag(t *testing.T) {
	root := initCommand()
	for _, name := range []string{"transcribe_config_encryption_keys", "config_encryption_keys", "master-key", "encryption-keys"} {
		if root.PersistentFlags().Lookup(name) != nil || root.Flags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists for the master key list", name)
		}
	}
}
