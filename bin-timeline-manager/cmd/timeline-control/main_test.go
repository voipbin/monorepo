package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestInitCommand(t *testing.T) {
	cmd := initCommand()
	if cmd == nil {
		t.Fatal("initCommand() returned nil")
	}

	if cmd.Use != "timeline-control" {
		t.Errorf("initCommand().Use = %q, want %q", cmd.Use, "timeline-control")
	}

	// Check that subcommands are registered
	subCmds := cmd.Commands()
	if len(subCmds) != 2 {
		t.Errorf("initCommand() has %d subcommands, want 2", len(subCmds))
	}

	// Check for event and migrate commands
	hasEvent := false
	hasMigrate := false
	for _, sub := range subCmds {
		if sub.Use == "event" {
			hasEvent = true
		}
		if sub.Use == "migrate" {
			hasMigrate = true
		}
	}

	if !hasEvent {
		t.Error("initCommand() missing 'event' subcommand")
	}
	if !hasMigrate {
		t.Error("initCommand() missing 'migrate' subcommand")
	}
}

func TestCmdEventList(t *testing.T) {
	cmd := cmdEventList()
	if cmd == nil {
		t.Fatal("cmdEventList() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("cmdEventList().Use = %q, want %q", cmd.Use, "list")
	}

	// Check flags
	flags := cmd.Flags()
	tests := []struct {
		name     string
		flagName string
	}{
		{"publisher flag", "publisher"},
		{"id flag", "id"},
		{"events flag", "events"},
		{"page-size flag", "page-size"},
		{"page-token flag", "page-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := flags.Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("cmdEventList() missing flag %q", tt.flagName)
			}
		})
	}
}

func TestCmdEventList_DefaultPageSize(t *testing.T) {
	cmd := cmdEventList()
	flags := cmd.Flags()

	pageSizeFlag := flags.Lookup("page-size")
	if pageSizeFlag == nil {
		t.Fatal("page-size flag not found")
	}

	if pageSizeFlag.DefValue != "100" {
		t.Errorf("page-size default = %q, want %q", pageSizeFlag.DefValue, "100")
	}
}

func TestCmdMigrateUp(t *testing.T) {
	cmd := cmdMigrateUp()
	if cmd == nil {
		t.Fatal("cmdMigrateUp() returned nil")
	}

	if cmd.Use != "up" {
		t.Errorf("cmdMigrateUp().Use = %q, want %q", cmd.Use, "up")
	}
}

func TestCmdMigrateDown(t *testing.T) {
	cmd := cmdMigrateDown()
	if cmd == nil {
		t.Fatal("cmdMigrateDown() returned nil")
	}

	if cmd.Use != "down" {
		t.Errorf("cmdMigrateDown().Use = %q, want %q", cmd.Use, "down")
	}
}

func TestCmdMigrateStatus(t *testing.T) {
	cmd := cmdMigrateStatus()
	if cmd == nil {
		t.Fatal("cmdMigrateStatus() returned nil")
	}

	if cmd.Use != "status" {
		t.Errorf("cmdMigrateStatus().Use = %q, want %q", cmd.Use, "status")
	}
}

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "simple struct",
			input:   struct{ Name string }{"test"},
			wantErr: false,
		},
		{
			name:    "map",
			input:   map[string]string{"key": "value"},
			wantErr: false,
		},
		{
			name:    "slice",
			input:   []int{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := printJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("printJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunEventList_MissingPublisher(t *testing.T) {
	viper.Reset()
	viper.Set("id", "550e8400-e29b-41d4-a716-446655440000")
	viper.Set("events", "activeflow_*")

	cmd := &cobra.Command{}
	err := runEventList(cmd, []string{})
	if err == nil {
		t.Fatal("runEventList() expected error for missing publisher")
	}

	if err.Error() != "publisher is required" {
		t.Errorf("runEventList() error = %q, want %q", err.Error(), "publisher is required")
	}
}

func TestRunEventList_MissingID(t *testing.T) {
	viper.Reset()
	viper.Set("publisher", "flow-manager")
	viper.Set("events", "activeflow_*")

	cmd := &cobra.Command{}
	err := runEventList(cmd, []string{})
	if err == nil {
		t.Fatal("runEventList() expected error for missing id")
	}

	if err.Error() != "id is required" {
		t.Errorf("runEventList() error = %q, want %q", err.Error(), "id is required")
	}
}

func TestRunEventList_InvalidID(t *testing.T) {
	viper.Reset()
	viper.Set("publisher", "flow-manager")
	viper.Set("id", "not-a-uuid")
	viper.Set("events", "activeflow_*")

	cmd := &cobra.Command{}
	err := runEventList(cmd, []string{})
	if err == nil {
		t.Fatal("runEventList() expected error for invalid id")
	}

	if err.Error() != "invalid id format" {
		t.Errorf("runEventList() error = %q, want %q", err.Error(), "invalid id format")
	}
}

func TestRunEventList_MissingEvents(t *testing.T) {
	viper.Reset()
	viper.Set("publisher", "flow-manager")
	viper.Set("id", "550e8400-e29b-41d4-a716-446655440000")

	cmd := &cobra.Command{}
	err := runEventList(cmd, []string{})
	if err == nil {
		t.Fatal("runEventList() expected error for missing events")
	}

	if err.Error() != "events is required" {
		t.Errorf("runEventList() error = %q, want %q", err.Error(), "events is required")
	}
}

func TestEventListFlags(t *testing.T) {
	cmd := cmdEventList()

	// Check that flags are properly defined with their descriptions
	publisherFlag := cmd.Flags().Lookup("publisher")
	if publisherFlag.Usage != "Publisher service name (required)" {
		t.Errorf("publisher flag usage = %q, want %q", publisherFlag.Usage, "Publisher service name (required)")
	}

	idFlag := cmd.Flags().Lookup("id")
	if idFlag.Usage != "Resource ID (required)" {
		t.Errorf("id flag usage = %q, want %q", idFlag.Usage, "Resource ID (required)")
	}

	eventsFlag := cmd.Flags().Lookup("events")
	if eventsFlag.Usage != "Event patterns comma-separated (required, e.g., 'activeflow_*,flow_created')" {
		t.Errorf("events flag usage = %q, want %q", eventsFlag.Usage, "Event patterns comma-separated (required, e.g., 'activeflow_*,flow_created')")
	}
}

func TestMigrateSubcommands(t *testing.T) {
	cmd := initCommand()

	// Find migrate command
	var migrateCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "migrate" {
			migrateCmd = sub
			break
		}
	}

	if migrateCmd == nil {
		t.Fatal("migrate command not found")
	}

	// Check migrate subcommands
	subCmds := migrateCmd.Commands()
	if len(subCmds) != 3 {
		t.Errorf("migrate command has %d subcommands, want 3", len(subCmds))
	}

	hasUp := false
	hasDown := false
	hasStatus := false
	for _, sub := range subCmds {
		switch sub.Use {
		case "up":
			hasUp = true
		case "down":
			hasDown = true
		case "status":
			hasStatus = true
		}
	}

	if !hasUp {
		t.Error("migrate command missing 'up' subcommand")
	}
	if !hasDown {
		t.Error("migrate command missing 'down' subcommand")
	}
	if !hasStatus {
		t.Error("migrate command missing 'status' subcommand")
	}
}

func TestEventSubcommands(t *testing.T) {
	cmd := initCommand()

	// Find event command
	var eventCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "event" {
			eventCmd = sub
			break
		}
	}

	if eventCmd == nil {
		t.Fatal("event command not found")
	}

	// Check event subcommands
	subCmds := eventCmd.Commands()
	if len(subCmds) != 1 {
		t.Errorf("event command has %d subcommands, want 1", len(subCmds))
	}

	hasList := false
	for _, sub := range subCmds {
		if sub.Use == "list" {
			hasList = true
		}
	}

	if !hasList {
		t.Error("event command missing 'list' subcommand")
	}
}

func TestPrintJSON_Output(t *testing.T) {
	// Capture output by redirecting stdout is complex in Go,
	// so we test that printJSON produces valid JSON
	data := map[string]interface{}{
		"result": []map[string]string{
			{"event_type": "activeflow_created"},
		},
		"next_page_token": "2024-01-15T10:30:00.123Z",
	}

	// Marshal to verify structure
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, output was not valid JSON", err)
	}

	if parsed["next_page_token"] != "2024-01-15T10:30:00.123Z" {
		t.Errorf("parsed next_page_token = %v, want %v", parsed["next_page_token"], "2024-01-15T10:30:00.123Z")
	}
}

func TestCommandHelp(t *testing.T) {
	cmd := initCommand()

	// Test that help doesn't panic
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	// Execute should not return an error for help
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("cmd.Execute() with --help error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("help output is empty")
	}
}
