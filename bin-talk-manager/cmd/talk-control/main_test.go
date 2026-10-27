package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/spf13/viper"
)

func TestMain(m *testing.M) {
	// A DSN that the mysql driver rejects while parsing it, so every code path that reaches the
	// database fails at once and offline. An empty DSN would try to connect to a local server.
	if err := os.Setenv("DATABASE_DSN", "bad"); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// execute runs talk-control with the given arguments on a freshly built command tree and
// returns the error of the command. The viper state is global, so it is reset before the tree
// is built (the flag bindings are made while it is built).
func execute(t *testing.T, args ...string) error {
	t.Helper()

	// environment variables that viper.AutomaticEnv would otherwise map onto the flags
	for _, key := range []string{"TYPE", "NAME", "DETAIL", "ID", "TOKEN", "CHAT_ID", "OWNER_TYPE", "OWNER_ID", "EMOJI", "TEXT", "SIZE"} {
		t.Setenv(key, "")
	}

	viper.Reset()
	t.Cleanup(viper.Reset)

	root := initCommand()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetArgs(args)

	return root.Execute()
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name  string
		value string

		wantID  uuid.UUID
		wantErr string
	}{
		{"empty", "", uuid.Nil, "Customer ID is required"},
		{"not a uuid", "not-a-uuid", uuid.Nil, "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID"},
		{"valid", "5e4a0680-804e-11ec-8477-2fea5968d85b", uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("customer-id", tt.value)

			got, err := resolveUUID("customer-id", "Customer ID")
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.wantID {
				t.Errorf("resolveUUID() = %v, %v, want %v", got, err, tt.wantID)
			}
		})
	}
}

// captureStdout runs fn and returns what it wrote to the standard output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("could not read the output: %v", err)
	}

	return buf.String()
}

func Test_printJSON(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(map[string]any{"name": "chat", "members": []int{1, 2}})
	})
	if err != nil {
		t.Fatalf("printJSON() error = %v", err)
	}

	want := "{\n  \"members\": [\n    1,\n    2\n  ],\n  \"name\": \"chat\"\n}\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func Test_printJSON_error(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(make(chan int))
	})

	if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
		t.Errorf("error = %v, want a marshal failure", err)
	}
	if out != "" {
		t.Errorf("output = %q, want nothing to be printed", out)
	}
}

func Test_chatCreate_validation_error(t *testing.T) {
	const customerID = "5e4a0680-804e-11ec-8477-2fea5968d85b"

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "customer id is missing",
			args:    []string{"chat", "create"},
			wantErr: "failed to resolve customer ID: Customer ID is required",
		},
		{
			name:    "customer id is not a uuid",
			args:    []string{"chat", "create", "--customer-id", "bad"},
			wantErr: "invalid format for Customer ID",
		},
		{
			name:    "type is missing",
			args:    []string{"chat", "create", "--customer-id", customerID},
			wantErr: "chat type is required",
		},
		{
			name:    "creator type is missing",
			args:    []string{"chat", "create", "--customer-id", customerID, "--type", "group"},
			wantErr: "creator type is required",
		},
		{
			name:    "creator id is missing",
			args:    []string{"chat", "create", "--customer-id", customerID, "--type", "group", "--creator-type", "agent"},
			wantErr: "failed to resolve creator ID: Creator ID is required",
		},
		{
			name:    "creator id is not a uuid",
			args:    []string{"chat", "create", "--customer-id", customerID, "--type", "group", "--creator-type", "agent", "--creator-id", "bad"},
			wantErr: "invalid format for Creator ID",
		},
		{
			// every input is valid, so the command goes on to open the database, which fails
			name: "valid input reaches the database",
			args: []string{"chat", "create", "--customer-id", customerID, "--type", "group", "--creator-type", "agent",
				"--creator-id", "7fcd7990-42eb-11ed-9fa6-b4cd93af9796"},
			wantErr: "failed to initialize handlers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := execute(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

// The commands other than "chat create" open the database before they look at their flags, so
// with a database address that cannot be parsed they all stop at the same place.
func Test_databaseCommands_initHandlers_error(t *testing.T) {
	tests := [][]string{
		{"chat", "get"},
		{"chat", "list"},
		{"chat", "update"},
		{"chat", "delete"},
		{"message", "create"},
		{"message", "get"},
		{"message", "list"},
		{"message", "delete"},
		{"participant", "add"},
		{"participant", "list"},
		{"participant", "remove"},
		{"reaction", "add"},
		{"reaction", "remove"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			err := execute(t, args...)
			if err == nil || !strings.Contains(err.Error(), "failed to initialize handlers") || !strings.Contains(err.Error(), "invalid DSN") {
				t.Errorf("error = %v, want the database address failure", err)
			}
		})
	}
}

func Test_migrate_usesTheDatabaseAddress_error(t *testing.T) {
	err := execute(t, "migrate", "up")
	if err == nil || !strings.Contains(err.Error(), "could not initialize the migrator") || !strings.Contains(err.Error(), "invalid DSN") {
		t.Errorf("error = %v, want the migrator to fail on the database address", err)
	}
}
