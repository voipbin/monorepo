package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-hook-manager/internal/config"
)

func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}

func Test_getPayloadData(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(filePath, []byte(`{"from":"file"}`), 0o600); err != nil {
		t.Fatalf("could not prepare the payload file: %v", err)
	}

	tests := []struct {
		name string

		data string
		file string

		expectRes string
		expectErr bool
	}{
		{name: "neither data nor file", expectErr: true},
		{name: "both data and file", data: `{"a":1}`, file: filePath, expectErr: true},
		{name: "data only", data: `{"a":1}`, expectRes: `{"a":1}`},
		{name: "file only", file: filePath, expectRes: `{"from":"file"}`},
		{name: "file does not exist", file: filepath.Join(dir, "missing.json"), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			viper.Set("data", tt.data)
			viper.Set("file", tt.file)

			res, err := getPayloadData()
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}

			if string(res) != tt.expectRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectRes, string(res))
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	tests := []struct {
		name string

		input any

		expectErr bool
	}{
		{name: "map is printed as indented json", input: map[string]string{"status": "sent", "type": "email"}},
		{name: "unmarshalable value", input: make(chan int), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := os.Stdout
			r, w, errPipe := os.Pipe()
			if errPipe != nil {
				t.Fatalf("could not create a pipe: %v", errPipe)
			}
			os.Stdout = w
			defer func() { os.Stdout = orig }()

			err := printJSON(tt.input)

			_ = w.Close()
			os.Stdout = orig
			out, _ := io.ReadAll(r)

			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}

			if tt.expectErr {
				if len(out) != 0 {
					t.Errorf("nothing must be printed on error. got: %q", string(out))
				}
				return
			}

			var got map[string]string
			if errUnmarshal := json.Unmarshal(out, &got); errUnmarshal != nil {
				t.Fatalf("output is not json: %v. out: %q", errUnmarshal, string(out))
			}
			if got["status"] != "sent" || got["type"] != "email" {
				t.Errorf("Wrong match. got: %v", got)
			}
			if string(out) != "{\n  \"status\": \"sent\",\n  \"type\": \"email\"\n}\n" {
				t.Errorf("output must be indented with two spaces. got: %q", string(out))
			}
		})
	}
}

func Test_cmdSend(t *testing.T) {
	tests := []struct {
		name string

		build func() *cobra.Command

		expectUse string
	}{
		{name: "email", build: cmdSendEmail, expectUse: "send-email"},
		{name: "message", build: cmdSendMessage, expectUse: "send-message"},
		{name: "conversation", build: cmdSendConversation, expectUse: "send-conversation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.build()

			if cmd.Use != tt.expectUse {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectUse, cmd.Use)
			}
			if cmd.RunE == nil {
				t.Errorf("RunE must be set")
			}

			for _, name := range []string{"uri", "data", "file"} {
				f := cmd.Flags().Lookup(name)
				if f == nil {
					t.Errorf("flag %s is not defined", name)
					continue
				}
				if f.DefValue != "" {
					t.Errorf("Wrong default for flag %s. expect empty, got: %q", name, f.DefValue)
				}
			}
		})
	}
}

func Test_initCommand(t *testing.T) {
	resetViper(t)

	root := initCommand()

	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Use] = true
	}
	for _, use := range []string{"send-email", "send-message", "send-conversation"} {
		if !got[use] {
			t.Errorf("subcommand %s is not registered", use)
		}
	}
	if len(got) != 3 {
		t.Errorf("Wrong subcommand count. expect: 3, got: %d", len(got))
	}

	for _, name := range []string{
		"database_dsn", "prometheus_endpoint", "prometheus_listen_address", "rabbitmq_address",
		"ssl_privkey_base64", "ssl_cert_base64", "paddle_webhook_secret_key",
	} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("persistent flag %s is not defined", name)
		}
	}

	// run a no-op subcommand through Execute so PersistentPreRunE is exercised:
	// it must bind the subcommand's local flags into viper and load the config.
	var gotData string
	dummy := &cobra.Command{
		Use: "dummy",
		Run: func(cmd *cobra.Command, args []string) {
			gotData = viper.GetString("data")
		},
	}
	dummy.Flags().String("data", "", "")
	root.AddCommand(dummy)
	root.SetArgs([]string{"dummy", "--data", "payload-value", "--rabbitmq_address", "amqp://rabbit-test"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if gotData != "payload-value" {
		t.Errorf("local flag was not bound into viper. expect: payload-value, got: %q", gotData)
	}
	if config.Get().RabbitMQAddress != "amqp://rabbit-test" {
		t.Errorf("config was not loaded from the flag. got: %s", config.Get().RabbitMQAddress)
	}
}

func Test_runSend_validationFailsBeforeConnect(t *testing.T) {
	tests := []struct {
		name string

		uri  string
		data string

		expectErr string
	}{
		{name: "uri is missing", uri: "", data: "", expectErr: "uri is required"},
		{name: "payload is missing", uri: "example.test", data: "", expectErr: "failed to get payload data: either --data or --file is required"},
	}

	runners := map[string]func(*cobra.Command, []string) error{
		"email":        runSendEmail,
		"message":      runSendMessage,
		"conversation": runSendConversation,
	}

	for runnerName, run := range runners {
		for _, tt := range tests {
			t.Run(runnerName+" "+tt.name, func(t *testing.T) {
				resetViper(t)
				viper.Set("uri", tt.uri)
				viper.Set("data", tt.data)

				done := make(chan error, 1)
				go func() {
					done <- run(&cobra.Command{}, nil)
				}()

				select {
				case err := <-done:
					if err == nil || err.Error() != tt.expectErr {
						t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
					}
				case <-time.After(3 * time.Second):
					t.Fatalf("the command did not return before connecting")
				}
			})
		}
	}
}
