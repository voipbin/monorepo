package main

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"monorepo/bin-contact-manager/internal/config"
)

// TestMain fixes the process-wide configuration before the first command runs:
// config.LoadGlobalConfig reads the environment once per process.
func TestMain(m *testing.M) {
	fixed := map[string]string{
		"DATABASE_DSN":     "not-a-dsn",
		"REDIS_ADDRESS":    "redis.invalid:6379",
		"RABBITMQ_ADDRESS": "amqp://rabbit.invalid:5672",
	}
	for key, val := range fixed {
		if err := os.Setenv(key, val); err != nil {
			panic(err)
		}
	}

	os.Exit(m.Run())
}

func Test_initCommand(t *testing.T) {
	root := initCommand()

	if root.Use != "case-control" {
		t.Errorf("Wrong match. expect: case-control, got: %s", root.Use)
	}
	if root.Short != "Voipbin Case Management CLI" {
		t.Errorf("Wrong match. expect: Voipbin Case Management CLI, got: %s", root.Short)
	}

	names := []string{}
	for _, c := range root.Commands() {
		names = append(names, c.Name())
	}
	sort.Strings(names)
	if expect := []string{"case"}; !reflect.DeepEqual(names, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, names)
	}

	for _, name := range []string{"database_dsn", "redis_address", "redis_database", "redis_password", "rabbitmq_address"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("Wrong match. expect: persistent flag %s, got: none", name)
		}
	}
}

// The case subcommand is not runnable, so Execute() never invokes the pre-run hook;
// it is called directly to check the configuration it loads from the environment.
func Test_initCommand_persistentPreRun(t *testing.T) {
	root := initCommand()

	if root.PersistentPreRunE == nil {
		t.Fatalf("Wrong match. expect: PersistentPreRunE set, got: nil")
	}
	if err := root.PersistentPreRunE(root, nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if got := config.Get().DatabaseDSN; got != "not-a-dsn" {
		t.Errorf("Wrong match. expect: not-a-dsn, got: %s", got)
	}
}
