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

	"monorepo/bin-pipecat-manager/internal/config"
	"monorepo/bin-pipecat-manager/internal/pipecatconfig"
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
	h, err := pipecatconfig.New(cfg, store, confighandler.WithSource(historySourceCLI))
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

	// the variable of the key is set: the stored value wins and the list says the variable is shadowed
	t.Setenv("DEEPGRAM_API_KEY", "env-value")
	store, out := useConfigHarness(t, &config.Config{ConfigEncryptionKeys: testMasterKeys()}, secret+"\n")

	if err := runConfigCommand("set", "--key", "deepgram.api_key", "--actor-id", testActor); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if strings.Contains(out.String(), secret) || !strings.Contains(out.String(), `"source": "db"`) {
		t.Errorf("Wrong match. set output: %s", out.String())
	}

	row, err := store.Get(nil, "deepgram.api_key") //nolint:staticcheck // the in-memory store ignores the context
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
	if !strings.Contains(out.String(), `"env_shadowed": true`) || strings.Contains(out.String(), "env_name") {
		t.Errorf("Wrong match. expect: env_shadowed true and no env_name, list output: %s", out.String())
	}

	out.Reset()
	if err := runConfigCommand("clear", "--key", "deepgram.api_key", "--actor-id", testActor); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !strings.Contains(out.String(), `"source": "env"`) || strings.Contains(out.String(), "env-value") {
		t.Errorf("Wrong match. expect: source env after the clear (the variable applies again), got: %s", out.String())
	}
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
		{name: "set without actor", args: []string{"set", "--key", "deepgram.api_key"}, cfg: &config.Config{}, expectErr: "Actor ID is required"},
		{name: "set bad actor", args: []string{"set", "--key", "deepgram.api_key", "--actor-id", "nope"}, cfg: &config.Config{}, expectErr: "not a valid UUID"},
		{name: "set unknown key", args: []string{"set", "--key", "no.such", "--actor-id", testActor}, cfg: &config.Config{}, expectErr: "unknown config key"},
		{name: "set without master key", args: []string{"set", "--key", "deepgram.api_key", "--actor-id", testActor}, stdin: "v\n", cfg: &config.Config{}, expectErr: "failed to store the value"},
		{name: "set too long on stdin", args: []string{"set", "--key", "deepgram.api_key", "--actor-id", testActor}, stdin: strings.Repeat("a", maxStdinBytes+1), cfg: &config.Config{ConfigEncryptionKeys: testMasterKeys()}, expectErr: "too long"},
		{name: "set too long for the key", args: []string{"set", "--key", "deepgram.api_key", "--actor-id", testActor}, stdin: strings.Repeat("a", 513), cfg: &config.Config{ConfigEncryptionKeys: testMasterKeys()}, expectErr: "failed to store the value"},
		{name: "set empty", args: []string{"set", "--key", "deepgram.api_key", "--actor-id", testActor}, stdin: "\n", cfg: &config.Config{ConfigEncryptionKeys: testMasterKeys()}, expectErr: "failed to store the value"},
		{name: "clear without key", args: []string{"clear", "--actor-id", testActor}, cfg: &config.Config{}, expectErr: "key is required"},
		{name: "clear without actor", args: []string{"clear", "--key", "deepgram.api_key"}, cfg: &config.Config{}, expectErr: "Actor ID is required"},
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
		{"set", "--key", "deepgram.api_key", "--actor-id", testActor},
		{"clear", "--key", "deepgram.api_key", "--actor-id", testActor},
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
	t.Setenv("OPENAI_API_KEY", "env-access")
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

	// the environment of this process is a source (os.LookupEnv): the key whose variable is set
	// shows env, the others none, and no value is printed.
	if !strings.Contains(out.String(), `"key": "openai.api_key"`) || !strings.Contains(out.String(), `"key": "deepgram.api_key"`) ||
		!strings.Contains(out.String(), `"source": "none"`) || strings.Count(out.String(), `"source": "env"`) != 1 || strings.Contains(out.String(), "env-access") {
		t.Errorf("Wrong match. list output: %s", out.String())
	}
}

// The master key list is never a flag of any command.
func Test_config_noMasterKeyFlag(t *testing.T) {
	root := initCommand()
	for _, name := range []string{"pipecat_config_encryption_keys", "config_encryption_keys", "master-key", "encryption-keys"} {
		if root.PersistentFlags().Lookup(name) != nil || root.Flags().Lookup(name) != nil {
			t.Errorf("Wrong match. a flag %q exists for the master key list", name)
		}
	}
}
