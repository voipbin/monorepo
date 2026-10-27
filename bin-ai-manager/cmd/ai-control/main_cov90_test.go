package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-ai-manager/models/ai"
)

// resetGlobalState restores the process-wide state that the command tree touches:
// the logger level and formatter that the config bootstrap sets, and the global
// viper instance. It is registered before the state is changed.
func resetGlobalState(t *testing.T) {
	t.Helper()

	level := logrus.GetLevel()
	formatter := logrus.StandardLogger().Formatter
	viper.Reset()

	t.Cleanup(func() {
		logrus.SetLevel(level)
		logrus.SetFormatter(formatter)
		viper.Reset()
	})
}

func subcommandNames(cmd *cobra.Command) []string {
	names := []string{}
	for _, c := range cmd.Commands() {
		names = append(names, c.Name())
	}
	return names
}

func Test_initCommand_Tree(t *testing.T) {
	resetGlobalState(t)

	root := initCommand()
	if root.Use != "ai-control" {
		t.Errorf("expected the root command ai-control, got: %s", root.Use)
	}

	tests := []struct {
		path   []string
		expect []string
	}{
		{[]string{"ai"}, []string{"create", "delete", "get", "list", "update"}},
		{[]string{"aicall"}, []string{"delete", "get", "get-by-reference", "list"}},
		{[]string{"migrate"}, []string{"down", "force", "status", "up"}},
	}
	for _, tt := range tests {
		sub, _, err := root.Find(tt.path)
		if err != nil || sub == root {
			t.Fatalf("subcommand %v not found: %v", tt.path, err)
		}

		got := subcommandNames(sub)
		if len(got) != len(tt.expect) {
			t.Errorf("%v: expected subcommands %v, got %v", tt.path, tt.expect, got)
			continue
		}
		for i := range got {
			if got[i] != tt.expect[i] {
				t.Errorf("%v: expected subcommands %v, got %v", tt.path, tt.expect, got)
				break
			}
		}
	}

	if root.PersistentPreRunE == nil {
		t.Errorf("the root command must validate and load the configuration before any subcommand runs")
	}
}

// Test_commandFlags_Contract verifies the flag contract of each command (the run
// functions need a database and are not called here):
// the values given on the command line are what the run function reads from the
// global viper instance, with the types it reads them as, and the defaults it
// relies on apply when a flag is not given.
func Test_commandFlags_Contract(t *testing.T) {
	tests := []struct {
		name   string
		build  func() *cobra.Command
		args   []string
		expect map[string]any
	}{
		{
			name: "ai create", build: cmdCreate,
			args: []string{"--customer-id", "c1", "--name", "n", "--detail", "d", "--engine-model", "m", "--engine-key", "k",
				"--init-prompt", "p", "--tts-type", "tt", "--tts-voice-id", "v", "--stt-type", "st", "--stt-language", "ko-KR",
				"--vad-config", `{"stop_secs": 0.5}`, "--smart-turn-enabled"},
			expect: map[string]any{
				"customer-id": "c1", "name": "n", "detail": "d", "engine-model": "m", "engine-key": "k", "init-prompt": "p",
				"tts-type": "tt", "tts-voice-id": "v", "stt-type": "st", "stt-language": "ko-KR",
				"vad-config": `{"stop_secs": 0.5}`, "smart-turn-enabled": true,
			},
		},
		{
			name: "ai get", build: cmdGet, args: []string{"--id", "i1"},
			expect: map[string]any{"id": "i1"},
		},
		{
			name: "ai list", build: cmdList, args: []string{"--limit", "25", "--token", "tok", "--customer-id", "c1"},
			expect: map[string]any{"limit": 25, "token": "tok", "customer-id": "c1"},
		},
		{
			name: "ai list defaults", build: cmdList, args: []string{},
			expect: map[string]any{"limit": 100, "token": "", "customer-id": ""},
		},
		{
			name: "ai update", build: cmdUpdate,
			args:   []string{"--id", "i1", "--name", "n", "--engine-key", "k", "--smart-turn-enabled"},
			expect: map[string]any{"id": "i1", "name": "n", "engine-key": "k", "smart-turn-enabled": true},
		},
		{
			// an update that names no engine model must leave the stored model alone
			name: "ai update defaults", build: cmdUpdate, args: []string{"--id", "i1"},
			expect: map[string]any{"engine-model": "", "engine-key": "", "vad-config": "", "smart-turn-enabled": false},
		},
		{
			name: "ai delete", build: cmdDelete, args: []string{"--id", "i1"},
			expect: map[string]any{"id": "i1"},
		},
		{
			name: "aicall get", build: cmdAIcallGet, args: []string{"--id", "a1"},
			expect: map[string]any{"id": "a1"},
		},
		{
			name: "aicall get by reference", build: cmdAIcallGetByReferenceID, args: []string{"--reference-id", "r1"},
			expect: map[string]any{"reference-id": "r1"},
		},
		{
			name: "aicall list", build: cmdAIcallList,
			args: []string{"--limit", "5", "--token", "tok", "--customer-id", "c1", "--assistance-type", "team",
				"--assistance-id", "as1", "--reference-type", "call", "--status", "progressing"},
			expect: map[string]any{"limit": 5, "token": "tok", "customer-id": "c1", "assistance-type": "team",
				"assistance-id": "as1", "reference-type": "call", "status": "progressing"},
		},
		{
			// no filter flag given: the run function adds no filter for an empty value
			name: "aicall list defaults", build: cmdAIcallList, args: []string{},
			expect: map[string]any{"limit": 100, "assistance-type": "", "assistance-id": "", "reference-type": "", "status": ""},
		},
		{
			name: "aicall delete", build: cmdAIcallDelete, args: []string{"--id", "a1"},
			expect: map[string]any{"id": "a1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalState(t)

			cmd := tt.build()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("could not parse the flags: %v", err)
			}
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				t.Fatalf("could not bind the flags: %v", err)
			}
			if cmd.RunE == nil {
				t.Errorf("the command has no run function")
			}

			for key, want := range tt.expect {
				var got any
				switch want.(type) {
				case string:
					got = viper.GetString(key)
				case int:
					got = viper.GetInt(key)
				case bool:
					got = viper.GetBool(key)
				}
				if got != want {
					t.Errorf("key %s: expected %v, got %v", key, want, got)
				}
			}
		})
	}
}

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expectErr string
	}{
		{"missing", "", "Customer ID is required"},
		{"malformed", "not-a-uuid", "is not a valid UUID"},
		{"nil uuid is not a valid id", "00000000-0000-0000-0000-000000000000", "is not a valid UUID"},
		{"valid", "5c9d8f40-7a31-11f1-9b22-4f1c0a6e3d11", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalState(t)
			viper.Set("customer-id", tt.value)

			res, err := resolveUUID("customer-id", "Customer ID")
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("expected an error containing %q, got: %v", tt.expectErr, err)
				}
				if res.String() != "00000000-0000-0000-0000-000000000000" {
					t.Errorf("expected the nil uuid with an error, got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.String() != tt.value {
				t.Errorf("expected %s, got %s", tt.value, res)
			}
		})
	}
}

func Test_printJSON_UnserializableValue(t *testing.T) {
	if err := printJSON(make(chan int)); err == nil {
		t.Errorf("expected an error for a value that cannot be serialized, got nil")
	}
}

func Test_withoutEndpointCiphertext_NilAI(t *testing.T) {
	var none *ai.AI

	res, ok := withoutEndpointCiphertext(none).(*ai.AI)
	if !ok || res != nil {
		t.Errorf("expected the nil ai to be returned as a nil *ai.AI, got: %#v", res)
	}
}

func Test_engineCheckError_KeyRequired(t *testing.T) {
	err := engineCheckError("openrouter.some/model", ai.ErrEngineKeyRequired)
	if err == nil || !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("expected the key-required message, got: %v", err)
	}

	other := errors.New("unrelated")
	if got := engineCheckError("openai.gpt-5", other); !errors.Is(got, other) {
		t.Errorf("expected an unrelated error to pass through, got: %v", got)
	}
}
