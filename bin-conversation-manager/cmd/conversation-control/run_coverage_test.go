package main

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-conversation-manager/internal/config"
)

// unreachableDSN points to a local port that refuses the connection at once, so no test depends on a real database.
const unreachableDSN = "tcp(127.0.0.1:1)/db"

func Test_initHandlers_databaseFailure(t *testing.T) {
	resetState(t)
	config.Get().DatabaseDSN = unreachableDSN

	conversationHandler, accountHandler, messageHandler, err := initHandlers()
	if err == nil {
		t.Fatalf("initHandlers() error = nil, want a database connection error")
	}
	if conversationHandler != nil || accountHandler != nil || messageHandler != nil {
		t.Errorf("initHandlers() must not return handlers on failure")
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("initHandlers() error = %q, want the database context", err.Error())
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("initHandlers() error does not keep the cause")
	}
}

func Test_run_databaseFailureComesFirst(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*cobra.Command, []string) error

		// flags that would be invalid or missing; the database failure must still be reported first
		flags map[string]string
	}{
		{name: "conversation get", fn: runConversationGet},
		{name: "conversation list without customer id", fn: runConversationList},
		{name: "account create without any flag", fn: runAccountCreate},
		{name: "account get with an invalid id", fn: runAccountGet, flags: map[string]string{"id": "not-a-uuid"}},
		{name: "account list", fn: runAccountList},
		{name: "account update without fields", fn: runAccountUpdate},
		{name: "account delete", fn: runAccountDelete},
		{name: "message get", fn: runMessageGet},
		{name: "message list with an invalid conversation id", fn: runMessageList, flags: map[string]string{"conversation-id": "x"}},
		{name: "message delete", fn: runMessageDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			config.Get().DatabaseDSN = unreachableDSN
			for k, v := range tt.flags {
				viper.Set(k, v)
			}

			err := tt.fn(nil, nil)
			if err == nil {
				t.Fatalf("error = nil, want the initialization failure")
			}
			if !strings.HasPrefix(err.Error(), "failed to initialize handlers: ") {
				t.Errorf("error = %q, want the prefix %q", err.Error(), "failed to initialize handlers: ")
			}
			if !strings.Contains(err.Error(), "could not connect to the database") {
				t.Errorf("error = %q, want the database context", err.Error())
			}
		})
	}
}
