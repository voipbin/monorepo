package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-agent-manager/internal/config"
)

// unparsableDSN is rejected by the database driver while it is parsed, so a
// connection attempt fails at once without any network access.
const unparsableDSN = "not-a-dsn"

// setGlobalConfig overwrites the process-wide config for one test and restores
// the previous value when the test ends.
func setGlobalConfig(t *testing.T, mutate func(c *config.Config)) {
	t.Helper()

	orig := *config.Get()
	mutate(config.Get())
	t.Cleanup(func() { *config.Get() = orig })
}

// respStub is a minimal in-process RESP server on a loopback port chosen by the
// operating system. It answers PING and records the commands it receives.
type respStub struct {
	ln net.Listener

	mu       sync.Mutex
	commands []string

	// failWith, when set, is sent as an error reply to every command.
	failWith string
}

func newRespStub(t *testing.T, failWith string) *respStub {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen on loopback: %v", err)
	}

	s := &respStub{ln: ln, failWith: failWith}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })

	return s
}

func (s *respStub) addr() string { return s.ln.Addr().String() }

func (s *respStub) received() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string{}, s.commands...)
}

func (s *respStub) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *respStub) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	r := bufio.NewReader(conn)
	for {
		args, err := readRespCommand(r)
		if err != nil {
			return
		}

		s.mu.Lock()
		s.commands = append(s.commands, strings.ToUpper(args[0]))
		s.mu.Unlock()

		switch {
		case s.failWith != "":
			_, _ = fmt.Fprintf(conn, "-ERR %s\r\n", s.failWith)
		case strings.EqualFold(args[0], "PING"):
			_, _ = conn.Write([]byte("+PONG\r\n"))
		default:
			_, _ = fmt.Fprintf(conn, "-ERR unknown command %q\r\n", args[0])
		}
	}
}

func readRespCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected line: %q", line)
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(header[1:]))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}

	return args, nil
}

func Test_initHandler_databaseFailure(t *testing.T) {
	setGlobalConfig(t, func(c *config.Config) { c.DatabaseDSN = unparsableDSN })

	res, err := initHandler()
	if err == nil {
		t.Fatalf("Wrong match. expect an error, got a handler: %v", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect no handler, got: %v", res)
	}
	if !strings.HasPrefix(err.Error(), "could not connect to the database: database open error: invalid DSN") {
		t.Errorf("Wrong match. got: %q", err.Error())
	}
}

// Test_run_initHandlerFailure drives every run function with a valid input up to
// the handler initialization, which must fail on the unparsable DSN and be
// reported as an initialization failure, not as an input error.
func Test_run_initHandlerFailure(t *testing.T) {
	const validID = "464a277e-2d8d-11ef-8bc6-d7b95604d6f6"

	tests := []struct {
		name string

		run      func(*cobra.Command, []string) error
		settings map[string]string
	}{
		{"create", runCreate, map[string]string{"customer-id": validID, "username": "user-a", "password": "secret-a"}},
		{"get", runGet, map[string]string{"id": validID}},
		{"list", runList, map[string]string{"customer-id": validID}},
		{"update-permission", runUpdatePermission, map[string]string{"id": validID}},
		{"update-password", runUpdatePassword, map[string]string{"id": validID, "password": "secret-a"}},
		{"delete", runDelete, map[string]string{"id": validID}},
		{"login", runLogin, map[string]string{"username": "user-a", "password": "secret-a"}},
		{"update-basic-info", runUpdateBasicInfo, map[string]string{"id": validID}},
		{"update-addresses", runUpdateAddresses, map[string]string{"id": validID, "addresses": "[]"}},
		{"update-status", runUpdateStatus, map[string]string{"id": validID, "status": "available"}},
		{"update-tag-ids", runUpdateTagIDs, map[string]string{"id": validID, "tag-ids": "[]"}},
	}

	const expectPrefix = "failed to initialize handlers: could not connect to the database: database open error: invalid DSN"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			setGlobalConfig(t, func(c *config.Config) { c.DatabaseDSN = unparsableDSN })
			for k, v := range tt.settings {
				viper.Set(k, v)
			}

			err := tt.run(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect an error, got nil")
			}
			if !strings.HasPrefix(err.Error(), expectPrefix) {
				t.Errorf("Wrong match.\nexpect prefix: %q\ngot: %q", expectPrefix, err.Error())
			}
		})
	}
}

func Test_runNormalizeAddresses_databaseFailure(t *testing.T) {
	resetViper(t)
	setGlobalConfig(t, func(c *config.Config) { c.DatabaseDSN = unparsableDSN })

	// the default of the command is the preview mode, which must stay on
	cmd := cmdNormalizeAddresses()
	if got, _ := cmd.Flags().GetBool("dry-run"); !got {
		t.Fatalf("Wrong match. expect the dry-run default to be true")
	}
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		t.Fatalf("could not bind the flags: %v", err)
	}

	err := runNormalizeAddresses(cmd, nil)
	if err == nil {
		t.Fatalf("Wrong match. expect an error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "could not connect to the database: database open error: invalid DSN") {
		t.Errorf("Wrong match. got: %q", err.Error())
	}
}

func Test_initCache(t *testing.T) {
	t.Run("a pong reply returns the connected handler", func(t *testing.T) {
		stub := newRespStub(t, "")
		setGlobalConfig(t, func(c *config.Config) {
			c.RedisAddress = stub.addr()
			c.RedisDatabase = 0
		})

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if res == nil {
			t.Fatalf("Wrong match. expect a handler, got nil")
		}
		if got := stub.received(); len(got) != 1 || got[0] != "PING" {
			t.Errorf("Wrong match. expect exactly one PING, got: %v", got)
		}
	})

	t.Run("an error reply is a connect failure", func(t *testing.T) {
		stub := newRespStub(t, "down for test")
		setGlobalConfig(t, func(c *config.Config) { c.RedisAddress = stub.addr() })

		res, err := initCache()
		if err == nil {
			t.Fatalf("Wrong match. expect an error, got a handler: %v", res)
		}
		if res != nil {
			t.Errorf("Wrong match. expect no handler, got: %v", res)
		}
		if !strings.HasPrefix(err.Error(), "could not connect to the cache: ") || !strings.Contains(err.Error(), "down for test") {
			t.Errorf("Wrong match. got: %q", err.Error())
		}
	})
}

// Test_initCommand_persistentPreRun runs a command whose input validation fails,
// so the root pre-run executes and the command ends before any connection.
func Test_initCommand_persistentPreRun(t *testing.T) {
	resetViper(t)
	t.Setenv("DATABASE_DSN", unparsableDSN)

	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})
	orig := *config.Get()
	t.Cleanup(func() { *config.Get() = orig })

	root := initCommand()
	root.SilenceUsage = true
	root.SilenceErrors = true

	calledPreRun := false
	preRun := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		calledPreRun = true
		return preRun(cmd, args)
	}

	root.SetArgs([]string{"agent", "get", "--id", "not-a-uuid"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("Wrong match. expect the validation error, got nil")
	}
	if !calledPreRun {
		t.Errorf("Wrong match. expect the persistent pre-run to run")
	}
	if !strings.HasPrefix(err.Error(), "invalid agent ID: invalid format for Agent ID: 'not-a-uuid' is not a valid UUID") {
		t.Errorf("Wrong match. got: %q", err.Error())
	}
}

// Test_initCommand_migrateFailure runs the migrate command against the
// unparsable DSN. The root pre-run is removed so that the process-wide config
// that this test sets is the one the command reads.
func Test_initCommand_migrateFailure(t *testing.T) {
	resetViper(t)

	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})
	setGlobalConfig(t, func(c *config.Config) { c.DatabaseDSN = unparsableDSN })

	root := initCommand()
	root.PersistentPreRunE = nil
	root.SilenceUsage = true
	root.SilenceErrors = true

	root.SetArgs([]string{"migrate", "status"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("Wrong match. expect an error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "dsn") {
		t.Errorf("Wrong match. expect the DSN error, got: %q", err.Error())
	}
}

func Test_main_help(t *testing.T) {
	resetViper(t)

	origFormatter := logrus.StandardLogger().Formatter
	origLevel := logrus.GetLevel()
	origArgs := os.Args
	t.Cleanup(func() {
		os.Args = origArgs
		logrus.SetFormatter(origFormatter)
		logrus.SetLevel(origLevel)
	})
	os.Args = []string{"agent-control", "--help"}

	out := captureStdout(t, main)
	if !strings.Contains(out, "Voipbin Agent Management CLI") {
		t.Errorf("Wrong match. expect the help text, got: %q", out)
	}
}
