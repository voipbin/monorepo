package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-number-manager/internal/config"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

// canary is a fake secret. It must appear nowhere but in the stdin of the test and in the
// decrypted value of the stored row.
const canary = "CANARY-fake-provider-secret-0001"

const providerTestActor = "11111111-2222-4333-8444-555555555555"

func providerTestKeys() string {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(0x40 + i)
	}

	return "1:" + base64.StdEncoding.EncodeToString(raw)
}

type providerRun struct {
	stdout string
	stderr string
	logs   string
	err    error
}

// all returns everything the command showed, including the error.
func (r providerRun) all() string {
	s := r.stdout + r.stderr + r.logs
	if r.err != nil {
		s += r.err.Error()
	}
	return s
}

// execProvider runs the real command tree (the root pre-run hook included) with the handler and
// the stdin replaced. The logs are captured at the level the CLI sets (debug).
func execProvider(t *testing.T, open func() (providerhandler.ProviderHandler, error), stdin string, args ...string) providerRun {
	t.Helper()

	resetViper(t)
	restoreLog(t)

	savedOpen, savedIn := openProviderHandler, providerIn
	openProviderHandler = open
	providerIn = strings.NewReader(stdin)
	t.Cleanup(func() { openProviderHandler, providerIn = savedOpen, savedIn })

	savedOut := logrus.StandardLogger().Out
	logs := &bytes.Buffer{}
	logrus.SetOutput(logs)
	t.Cleanup(func() { logrus.SetOutput(savedOut) })

	cmd := initCommand()
	stderr := &bytes.Buffer{}
	cmd.SetOut(stderr)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	var res providerRun
	res.stdout = captureStdout(t, func() { res.err = cmd.Execute() })
	res.stderr = stderr.String()
	res.logs = logs.String()

	return res
}

func openMock(h providerhandler.ProviderHandler) func() (providerhandler.ProviderHandler, error) {
	return func() (providerhandler.ProviderHandler, error) { return h, nil }
}

func Test_providerCommandTree(t *testing.T) {
	resetViper(t)
	restoreLog(t)

	root := initCommand()

	find := func(path ...string) *cobra.Command {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Fatalf("Wrong match. command %v is missing: %v", path, err)
		}
		return c
	}
	names := func(c *cobra.Command) []string {
		res := []string{}
		for _, s := range c.Commands() {
			res = append(res, s.Name())
		}
		sort.Strings(res)
		return res
	}

	if got := names(find("provider")); !reflect.DeepEqual(got, []string{"list", "secret", "update"}) {
		t.Errorf("Wrong match. provider subcommands: %v", got)
	}
	if got := names(find("provider", "secret")); !reflect.DeepEqual(got, []string{"delete", "set"}) {
		t.Errorf("Wrong match. provider secret subcommands: %v", got)
	}

	flagSet := func(c *cobra.Command) []string {
		res := []string{}
		c.Flags().VisitAll(func(f *pflag.Flag) { res = append(res, f.Name) })
		sort.Strings(res)
		return res
	}

	// the secret is never a flag: the exact set is pinned so that an added flag is caught
	tests := []struct {
		path   []string
		expect []string
	}{
		{[]string{"provider", "list"}, []string{}},
		{[]string{"provider", "update"}, []string{"actor-id", "config", "enabled", "name", "priority"}},
		{[]string{"provider", "secret", "set"}, []string{"actor-id", "name"}},
		{[]string{"provider", "secret", "delete"}, []string{"actor-id", "name"}},
	}
	for _, tt := range tests {
		if got := flagSet(find(tt.path...)); !reflect.DeepEqual(got, tt.expect) {
			t.Errorf("Wrong match. %v flags expect: %v, got: %v", tt.path, tt.expect, got)
		}
	}

	if f := find("provider", "update").Flags().Lookup("config"); f.Value.Type() != "stringArray" {
		t.Errorf("Wrong match. config flag type expect: stringArray, got: %s", f.Value.Type())
	}
}

func Test_readSecret(t *testing.T) {
	tests := []struct {
		name string

		in string

		expect    string
		expectErr string
	}{
		{name: "plain", in: "abc", expect: "abc"},
		{name: "one trailing newline", in: "abc\n", expect: "abc"},
		{name: "one trailing crlf", in: "abc\r\n", expect: "abc"},
		{name: "empty", in: "", expect: ""},
		{name: "two lines", in: "abc\ndef\n", expectErr: "more than one line"},
		{name: "two trailing newlines", in: "abc\n\n", expectErr: "more than one line"},
		{name: "too long", in: strings.Repeat("a", maxSecretStdinBytes+1), expectErr: "too long"},
		{name: "longest", in: strings.Repeat("a", maxSecretStdinBytes), expect: strings.Repeat("a", maxSecretStdinBytes)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readSecret(strings.NewReader(tt.in))
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil || got != tt.expect {
				t.Errorf("Wrong match. expect: %q, got: %q, err: %v", tt.expect, got, err)
			}
		})
	}
}

func Test_readSecret_files(t *testing.T) {
	// a character device (a terminal, /dev/null) is refused
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = devNull.Close() }()
	if _, err := readSecret(devNull); err == nil || !strings.Contains(err.Error(), "stdin must be a pipe or a file") {
		t.Errorf("Wrong match. expect the character device to be refused, got: %v", err)
	}

	// a pipe is accepted
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("from-pipe\n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	if got, err := readSecret(r); err != nil || got != "from-pipe" {
		t.Errorf("Wrong match. pipe expect: from-pipe, got: %q, err: %v", got, err)
	}
	_ = r.Close()

	// a file is accepted
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if got, err := readSecret(f); err != nil || got != "from-file" {
		t.Errorf("Wrong match. file expect: from-file, got: %q, err: %v", got, err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failure") }

func Test_readSecret_readError(t *testing.T) {
	if _, err := readSecret(failingReader{}); err == nil || !strings.Contains(err.Error(), "could not read the secret from stdin") {
		t.Errorf("Wrong match. got: %v", err)
	}
	_ = io.EOF
}

func Test_parseConfigEntries(t *testing.T) {
	tests := []struct {
		name string

		in []string

		expect    map[string]string
		expectErr string
	}{
		{name: "none", in: nil, expect: map[string]string{}},
		{name: "one", in: []string{"connection_id=abc"}, expect: map[string]string{"connection_id": "abc"}},
		{name: "empty value removes", in: []string{"profile_id="}, expect: map[string]string{"profile_id": ""}},
		{name: "value with equal sign and comma", in: []string{"k=a=b,c"}, expect: map[string]string{"k": "a=b,c"}},
		{name: "no equal sign", in: []string{canary}, expectErr: "invalid --config entry"},
		{name: "empty key", in: []string{"=" + canary}, expectErr: "invalid --config entry"},
		{name: "duplicate", in: []string{"a=1", "a=" + canary}, expectErr: "duplicate --config key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfigEntries(tt.in)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				if strings.Contains(err.Error(), canary) {
					t.Errorf("Wrong match. the error echoes the entry: %v", err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v, err: %v", tt.expect, got, err)
			}
		})
	}
}

func Test_provider_calls(t *testing.T) {
	actor := providerTestActor
	enabled := false
	priority := 5

	tests := []struct {
		name string

		stdin string
		args  []string

		expectCall func(m *providerhandler.MockProviderHandler)
	}{
		{
			name: "list",
			args: []string{"provider", "list"},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().List(gomock.Any()).Return([]*provider.Info{{ProviderName: number.ProviderNameTelnyx}}, nil)
			},
		},
		{
			name: "update all settings",
			args: []string{"provider", "update", "--name", "telnyx", "--actor-id", actor, "--enabled=false", "--priority", "5", "--config", "connection_id=c1", "--config", "profile_id="},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Update(gomock.Any(), number.ProviderNameTelnyx, gomock.Any(), &enabled, &priority, map[string]string{"connection_id": "c1", "profile_id": ""}).
					Return(&provider.Info{ProviderName: number.ProviderNameTelnyx}, nil)
			},
		},
		{
			name: "update only config passes nil for the unset flags",
			args: []string{"provider", "update", "--name", "twilio", "--actor-id", actor, "--config", "sid=ACfake"},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Update(gomock.Any(), number.ProviderNameTwilio, gomock.Any(), gomock.Nil(), gomock.Nil(), map[string]string{"sid": "ACfake"}).
					Return(&provider.Info{ProviderName: number.ProviderNameTwilio}, nil)
			},
		},
		{
			name:  "secret set passes the value of stdin",
			stdin: canary + "\n",
			args:  []string{"provider", "secret", "set", "--name", "telnyx", "--actor-id", actor},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().UpdateSecret(gomock.Any(), number.ProviderNameTelnyx, gomock.Any(), canary).
					Return(&provider.Info{ProviderName: number.ProviderNameTelnyx, HasSecret: true}, nil)
			},
		},
		{
			name: "secret delete",
			args: []string{"provider", "secret", "delete", "--name", "twilio", "--actor-id", actor},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().DeleteSecret(gomock.Any(), number.ProviderNameTwilio, gomock.Any()).
					Return(&provider.Info{ProviderName: number.ProviderNameTwilio}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := providerhandler.NewMockProviderHandler(mc)
			tt.expectCall(m)

			res := execProvider(t, openMock(m), tt.stdin, tt.args...)
			if res.err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", res.err)
			}
			if strings.Contains(res.stdout+res.stderr+res.logs, canary) {
				t.Errorf("Wrong match. the secret is in the output: %s", res.all())
			}
		})
	}
}

func Test_provider_actorIsPassed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	m := providerhandler.NewMockProviderHandler(mc)
	m.EXPECT().DeleteSecret(gomock.Any(), number.ProviderNameTelnyx, gomock.Any()).DoAndReturn(
		func(_ any, _ number.ProviderName, actor any) (*provider.Info, error) {
			if fmt.Sprint(actor) != providerTestActor {
				t.Errorf("Wrong match. expect actor: %s, got: %v", providerTestActor, actor)
			}
			return &provider.Info{}, nil
		})

	res := execProvider(t, openMock(m), "", "provider", "secret", "delete", "--name", "telnyx", "--actor-id", providerTestActor)
	if res.err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", res.err)
	}
}

// Test_provider_rejected runs the inputs that must fail before the handler is used. The mock has no
// expectation, so a call to it fails the test. No failure message may echo the given value.
func Test_provider_rejected(t *testing.T) {
	tests := []struct {
		name string

		env   map[string]string
		stdin string
		args  []string

		expectErr string
	}{
		{name: "update without name", args: []string{"provider", "update", "--actor-id", providerTestActor, "--priority", "1"}, expectErr: "name is required"},
		{name: "update with unknown name", args: []string{"provider", "update", "--name", canary, "--actor-id", providerTestActor, "--priority", "1"}, expectErr: "unknown provider name"},
		{name: "update without actor", args: []string{"provider", "update", "--name", "telnyx", "--priority", "1"}, expectErr: "actor-id is required"},
		{name: "update with a malformed actor", args: []string{"provider", "update", "--name", "telnyx", "--actor-id", canary, "--priority", "1"}, expectErr: "actor-id must be a valid UUID"},
		{name: "update with nothing to change", args: []string{"provider", "update", "--name", "telnyx", "--actor-id", providerTestActor}, expectErr: "at least one of"},
		{name: "update with a bad config entry", args: []string{"provider", "update", "--name", "telnyx", "--actor-id", providerTestActor, "--config", canary}, expectErr: "invalid --config entry"},
		{name: "update with a positional argument (bool flag with a space)", args: []string{"provider", "update", "--name", "telnyx", "--actor-id", providerTestActor, "--enabled", canary}, expectErr: "unexpected argument"},
		{name: "secret set with the secret as an argument", stdin: "x\n", args: []string{"provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor, canary}, expectErr: "unexpected argument"},
		{name: "secret set with two lines", stdin: canary + "\n" + canary + "\n", args: []string{"provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor}, expectErr: "more than one line"},
		{name: "secret set too long", stdin: strings.Repeat(canary, 200), args: []string{"provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor}, expectErr: "too long"},
		{name: "secret set without name", stdin: canary, args: []string{"provider", "secret", "set", "--actor-id", providerTestActor}, expectErr: "name is required"},
		{name: "secret delete with unknown name", args: []string{"provider", "secret", "delete", "--name", "messagebird", "--actor-id", providerTestActor}, expectErr: "unknown provider name"},
		{name: "list with an argument", args: []string{"provider", "list", canary}, expectErr: "unexpected argument"},
		// the environment must not fill a missing flag (the pre-run hook binds the flags to viper
		// with AutomaticEnv, so a viper based reader would take these)
		{name: "name from the environment is not used", env: map[string]string{"NAME": "twilio", "ENABLED": "true", "PRIORITY": "3", "CONFIG": "sid=ACfake"}, args: []string{"provider", "update", "--actor-id", providerTestActor}, expectErr: "name is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			mc := gomock.NewController(t)
			defer mc.Finish()
			m := providerhandler.NewMockProviderHandler(mc)

			res := execProvider(t, openMock(m), tt.stdin, tt.args...)
			if res.err == nil || !strings.Contains(res.err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, res.err)
			}
			if strings.Contains(res.all(), canary) {
				t.Errorf("Wrong match. a given value is echoed: %s", res.all())
			}
		})
	}
}

// Test_provider_groupWithArgument checks the commands that only group subcommands: a value after
// the group name must not be echoed ("unknown command" echoes it by default).
func Test_provider_groupWithArgument(t *testing.T) {
	for _, args := range [][]string{
		{"provider", canary},
		{"provider", "secret", canary},
	} {
		mc := gomock.NewController(t)
		m := providerhandler.NewMockProviderHandler(mc)

		res := execProvider(t, openMock(m), "", args...)
		if strings.Contains(res.all(), canary) {
			t.Errorf("Wrong match. args %v: a given value is echoed: %s", args, res.all())
		}
		mc.Finish()
	}
}

func Test_provider_openFailure(t *testing.T) {
	open := func() (providerhandler.ProviderHandler, error) { return nil, errors.New("cannot open") }

	for _, args := range [][]string{
		{"provider", "list"},
		{"provider", "update", "--name", "telnyx", "--actor-id", providerTestActor, "--priority", "1"},
		{"provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor},
		{"provider", "secret", "delete", "--name", "telnyx", "--actor-id", providerTestActor},
	} {
		res := execProvider(t, open, "value\n", args...)
		if res.err == nil || !strings.Contains(res.err.Error(), "failed to initialize handlers: cannot open") {
			t.Errorf("Wrong match. args %v: got: %v", args, res.err)
		}
	}
}

func Test_provider_handlerFailure(t *testing.T) {
	boom := errors.New("handler failure")

	tests := []struct {
		name string

		stdin string
		args  []string

		expectCall func(m *providerhandler.MockProviderHandler)
		expectErr  string
	}{
		{
			name: "list",
			args: []string{"provider", "list"},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().List(gomock.Any()).Return(nil, boom)
			},
			expectErr: "failed to list the providers",
		},
		{
			name: "update",
			args: []string{"provider", "update", "--name", "telnyx", "--actor-id", providerTestActor, "--priority", "1"},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom)
			},
			expectErr: "failed to update the provider",
		},
		{
			name:  "secret set",
			stdin: "value\n",
			args:  []string{"provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().UpdateSecret(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom)
			},
			expectErr: "failed to store the provider secret",
		},
		{
			name: "secret delete",
			args: []string{"provider", "secret", "delete", "--name", "telnyx", "--actor-id", providerTestActor},
			expectCall: func(m *providerhandler.MockProviderHandler) {
				m.EXPECT().DeleteSecret(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, boom)
			},
			expectErr: "failed to delete the provider secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := providerhandler.NewMockProviderHandler(mc)
			tt.expectCall(m)

			res := execProvider(t, openMock(m), tt.stdin, tt.args...)
			if res.err == nil || !strings.Contains(res.err.Error(), tt.expectErr) || !errors.Is(res.err, boom) {
				t.Errorf("Wrong match. expect error containing: %s wrapping the cause, got: %v", tt.expectErr, res.err)
			}
		})
	}
}

func Test_initProviderHandler_databaseUnreachable(t *testing.T) {
	const dsnPassword = "dsn-password-canary"
	setTestConfig(t, config.Config{DatabaseDSN: "svc:" + dsnPassword + "@tcp(127.0.0.1:1)/testdb?timeout=1s"})

	res, err := initProviderHandler()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect error containing: could not connect to the database, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
	if err != nil && strings.Contains(err.Error(), dsnPassword) {
		t.Errorf("Wrong match. the error holds the DSN password: %v", err)
	}
}

func Test_newProviderHandler(t *testing.T) {
	setTestConfig(t, config.Config{NumberSecretEncryptionKeys: "not-a-key-list"})

	res, err := newProviderHandler(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "could not parse the number secret encryption keys") {
		t.Errorf("Wrong match. got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
	if err != nil && strings.Contains(err.Error(), "not-a-key-list") {
		t.Errorf("Wrong match. the error holds the key text: %v", err)
	}

	setTestConfig(t, config.Config{NumberSecretEncryptionKeys: providerTestKeys()})
	if res, err := newProviderHandler(nil, nil); err != nil || res == nil {
		t.Errorf("Wrong match. expect: handler, got: %v, err: %v", res, err)
	}

	// an empty key list is accepted: the commands that need a key fail later
	setTestConfig(t, config.Config{})
	if res, err := newProviderHandler(nil, nil); err != nil || res == nil {
		t.Errorf("Wrong match. expect: handler, got: %v, err: %v", res, err)
	}
}

// providerE2E wires the real provider handler on an in-memory database. The handler comes from
// newProviderHandler, so the key and credential wiring of the commands is the tested one.
type providerE2E struct {
	db   *sql.DB
	open func() (providerhandler.ProviderHandler, error)
}

func newProviderE2E(t *testing.T, keys string) *providerE2E {
	t.Helper()

	setTestConfig(t, config.Config{NumberSecretEncryptionKeys: keys})

	name := regexp.MustCompile(`[^A-Za-z0-9_]`).ReplaceAllString(t.Name(), "_")
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", name))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile("../../scripts/database_scripts_test/table_number_providers.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}

	return &providerE2E{
		db: db,
		open: func() (providerhandler.ProviderHandler, error) {
			return newProviderHandler(nil, dbhandler.NewHandler(db, nil))
		},
	}
}

func (e *providerE2E) run(t *testing.T, stdin string, args ...string) providerRun {
	t.Helper()
	return execProvider(t, e.open, stdin, args...)
}

func jsonKeys(t *testing.T, raw string) []string {
	t.Helper()

	m := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("could not parse the output as an object. err: %v, output: %s", err, raw)
	}
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)

	return res
}

// clearCredentialEnv unsets the credential variables of the providers for the test.
func clearCredentialEnv(t *testing.T) {
	t.Helper()

	for _, name := range []string{"TELNYX_API_KEY", "TELNYX_CONNECTION_ID", "TELNYX_PROFILE_ID", "TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN"} {
		t.Setenv(name, "")
	}
}

func Test_provider_e2e(t *testing.T) {
	clearCredentialEnv(t)
	e := newProviderE2E(t, providerTestKeys())

	// a provider without a row: the defaults, a literal key set (no secret field can slip in)
	res := e.run(t, "", "provider", "list")
	if res.err != nil {
		t.Fatalf("Wrong match. list expect: ok, got: %v", res.err)
	}
	var list []map[string]any
	if err := json.Unmarshal([]byte(res.stdout), &list); err != nil || len(list) != 2 {
		t.Fatalf("Wrong match. list expect 2 providers, got: %s, err: %v", res.stdout, err)
	}
	for _, item := range list {
		keys := make([]string, 0)
		for k := range item {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		expect := []string{"credential_source", "enabled", "has_secret", "key_version", "priority", "provider_name", "usable"}
		if !reflect.DeepEqual(keys, expect) {
			t.Errorf("Wrong match. default view keys expect: %v, got: %v", expect, keys)
		}
	}

	// the config first
	res = e.run(t, "", "provider", "update", "--name", "telnyx", "--actor-id", providerTestActor, "--config", "connection_id=conn-fake-1")
	if res.err != nil {
		t.Fatalf("Wrong match. update expect: ok, got: %v", res.err)
	}
	expectUpdateKeys := []string{"config", "credential_source", "enabled", "has_secret", "key_version", "priority", "provider_name", "tm_update", "updated_by", "usable"}
	if got := jsonKeys(t, res.stdout); !reflect.DeepEqual(got, expectUpdateKeys) {
		t.Errorf("Wrong match. update view keys expect: %v, got: %v", expectUpdateKeys, got)
	}

	// then the secret
	res = e.run(t, canary+"\n", "provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor)
	if res.err != nil {
		t.Fatalf("Wrong match. secret set expect: ok, got: %v", res.err)
	}
	expectSetKeys := []string{"config", "credential_source", "enabled", "has_secret", "key_version", "priority", "provider_name", "secret_updated_by", "tm_secret_update", "tm_update", "updated_by", "usable"}
	if got := jsonKeys(t, res.stdout); !reflect.DeepEqual(got, expectSetKeys) {
		t.Errorf("Wrong match. secret set view keys expect: %v, got: %v", expectSetKeys, got)
	}
	view := map[string]any{}
	_ = json.Unmarshal([]byte(res.stdout), &view)
	// the stored secret is not used: the view follows the variable, which is not set here
	if view["has_secret"] != false || view["credential_source"] != "none" || view["key_version"] != float64(1) || view["usable"] != false {
		t.Errorf("Wrong match. secret set view: %v", view)
	}
	if !strings.Contains(res.stderr, "TELNYX_API_KEY") || !strings.Contains(res.stderr, "stored value is not used") {
		t.Errorf("Wrong match. the notice of the variable is missing: %q", res.stderr)
	}

	// the positive control: the captured log is not empty, so its silence about the secret means something
	if res.stdout == "" || !strings.Contains(res.logs, "The provider secret has been changed.") {
		t.Errorf("Wrong match. the capture is broken. stdout: %q, logs: %q", res.stdout, res.logs)
	}
	if strings.Contains(res.all(), canary) {
		t.Errorf("Wrong match. the secret is in the output: %s", res.all())
	}

	// the stored value is the secret, encrypted
	var blob []byte
	var version int
	if err := e.db.QueryRow("select secret_ciphertext, key_version from number_providers where provider_name = 'telnyx'").Scan(&blob, &version); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(blob, []byte(canary)) {
		t.Errorf("Wrong match. the ciphertext holds the plaintext")
	}
	c, err := secretcrypto.New(providerTestKeys(), secretcrypto.WithMaxPlaintext(1024))
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := c.Decrypt(blob, version); err != nil || plain != canary {
		t.Errorf("Wrong match. the stored secret does not decrypt to the input. err: %v", err)
	}

	// the same command again succeeds (the stored secret is overwritten)
	res = e.run(t, canary+"\n", "provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor)
	if res.err != nil {
		t.Errorf("Wrong match. second secret set expect: ok, got: %v", res.err)
	}

	// list shows the state only
	res = e.run(t, "", "provider", "list")
	if res.err != nil || strings.Contains(res.all(), canary) || !strings.Contains(res.stdout, `"has_secret": false`) {
		t.Errorf("Wrong match. list after set: err: %v, output: %s", res.err, res.stdout)
	}

	// delete: the secret goes, the config stays
	res = e.run(t, "", "provider", "secret", "delete", "--name", "telnyx", "--actor-id", providerTestActor)
	if res.err != nil {
		t.Fatalf("Wrong match. secret delete expect: ok, got: %v", res.err)
	}
	view = map[string]any{}
	_ = json.Unmarshal([]byte(res.stdout), &view)
	if view["has_secret"] != false || view["credential_source"] != "none" {
		t.Errorf("Wrong match. secret delete view: %v", view)
	}
	if !strings.Contains(res.stderr, "TELNYX_API_KEY") || !strings.Contains(res.stderr, "stored value is not used") {
		t.Errorf("Wrong match. the notice of the variable is missing: %q", res.stderr)
	}
	if cfg, _ := view["config"].(map[string]any); cfg["connection_id"] != "conn-fake-1" {
		t.Errorf("Wrong match. the config must stay: %v", view["config"])
	}

	// a provider that never had a row is not found
	res = e.run(t, "", "provider", "secret", "delete", "--name", "twilio", "--actor-id", providerTestActor)
	if res.err == nil || !strings.Contains(res.err.Error(), "failed to delete the provider secret") {
		t.Errorf("Wrong match. delete without a row expect an error, got: %v", res.err)
	}
}

func Test_provider_e2e_failures(t *testing.T) {
	t.Run("no key list", func(t *testing.T) {
		e := newProviderE2E(t, "")

		res := e.run(t, canary+"\n", "provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor)
		if !errors.Is(res.err, provider.ErrSecretKeyNotConfigured) {
			t.Fatalf("Wrong match. expect: ErrSecretKeyNotConfigured, got: %v", res.err)
		}
		if strings.Contains(res.all(), canary) {
			t.Errorf("Wrong match. the secret is in the output: %s", res.all())
		}

		// nothing was stored
		var n int
		if err := e.db.QueryRow("select count(*) from number_providers").Scan(&n); err != nil || n != 0 {
			t.Errorf("Wrong match. expect no row, got: %d, err: %v", n, err)
		}
	})

	t.Run("required config missing", func(t *testing.T) {
		e := newProviderE2E(t, providerTestKeys())

		res := e.run(t, canary+"\n", "provider", "secret", "set", "--name", "telnyx", "--actor-id", providerTestActor)
		if !errors.Is(res.err, provider.ErrInvalidArgument) || !strings.Contains(res.err.Error(), "connection_id") {
			t.Fatalf("Wrong match. expect: invalid argument naming connection_id, got: %v", res.err)
		}
		if strings.Contains(res.all(), canary) {
			t.Errorf("Wrong match. the secret is in the output: %s", res.all())
		}

		var n int
		if err := e.db.QueryRow("select count(*) from number_providers where secret_ciphertext is not null and length(secret_ciphertext) > 0").Scan(&n); err != nil || n != 0 {
			t.Errorf("Wrong match. expect no stored secret, got: %d, err: %v", n, err)
		}
	})

	t.Run("a config value is not echoed", func(t *testing.T) {
		e := newProviderE2E(t, providerTestKeys())

		res := e.run(t, "", "provider", "update", "--name", "telnyx", "--actor-id", providerTestActor, "--config", "bogus_key="+canary)
		if res.err == nil || !strings.Contains(res.err.Error(), "config key is not allowed") {
			t.Fatalf("Wrong match. expect: config key is not allowed, got: %v", res.err)
		}
		if strings.Contains(res.all(), canary) {
			t.Errorf("Wrong match. the value is echoed: %s", res.all())
		}
	})

	t.Run("a malformed key list stops the command", func(t *testing.T) {
		e := newProviderE2E(t, "not-a-key-list")

		for _, args := range [][]string{
			{"provider", "list"},
			{"provider", "secret", "delete", "--name", "telnyx", "--actor-id", providerTestActor},
		} {
			res := e.run(t, "", args...)
			if res.err == nil || !strings.Contains(res.err.Error(), "could not parse the number secret encryption keys") {
				t.Errorf("Wrong match. args %v: got: %v", args, res.err)
			}
			if strings.Contains(res.all(), "not-a-key-list") {
				t.Errorf("Wrong match. args %v: the key text is echoed: %s", args, res.all())
			}
		}
	})

	t.Run("a malformed twilio sid does not panic the warning path", func(t *testing.T) {
		e := newProviderE2E(t, providerTestKeys())

		res := e.run(t, "", "provider", "update", "--name", "twilio", "--actor-id", providerTestActor, "--enabled=true", "--config", "sid=bad", "--config", "trunk_sid=bad")
		if res.err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", res.err)
		}
		res = e.run(t, canary+"\n", "provider", "secret", "set", "--name", "twilio", "--actor-id", providerTestActor)
		if res.err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", res.err)
		}
		if strings.Contains(res.all(), canary) {
			t.Errorf("Wrong match. the secret is in the output: %s", res.all())
		}
	})
}

// Test_provider_e2e_environmentCredentials pins what the operator sees when the credential
// variables are set: the secret comes from the variable (source env), the stored row config is
// shown as it is, and no secret value appears in the output.
func Test_provider_e2e_environmentCredentials(t *testing.T) {
	clearCredentialEnv(t)
	e := newProviderE2E(t, providerTestKeys())
	t.Setenv("TELNYX_API_KEY", "env-fake-telnyx-token")
	t.Setenv("TELNYX_CONNECTION_ID", "env-fake-connection")
	t.Setenv("TWILIO_AUTH_TOKEN", "env-fake-twilio-token")
	t.Setenv("TWILIO_ACCOUNT_SID", "env-fake-sid")

	res := e.run(t, "", "provider", "list")
	if res.err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", res.err)
	}

	var list []map[string]any
	if err := json.Unmarshal([]byte(res.stdout), &list); err != nil || len(list) != 2 {
		t.Fatalf("Wrong match. got: %s, err: %v", res.stdout, err)
	}

	for _, item := range list {
		if item["credential_source"] != "env" || item["has_secret"] != true {
			t.Errorf("Wrong match. view: %v", item)
		}
		if _, ok := item["config"]; ok {
			t.Errorf("Wrong match. a variable value is shown as the config: %v", item)
		}
		// telnyx is usable through the variables, twilio is disabled by default and needs a trunk_sid
		if item["provider_name"] == "telnyx" && item["usable"] != true {
			t.Errorf("Wrong match. telnyx view: %v", item)
		}
		if item["provider_name"] == "twilio" && item["usable"] != false {
			t.Errorf("Wrong match. twilio view: %v", item)
		}
	}
	if strings.Contains(res.all(), "env-fake-telnyx-token") || strings.Contains(res.all(), "env-fake-twilio-token") {
		t.Errorf("Wrong match. an environment secret is in the output: %s", res.all())
	}
}
