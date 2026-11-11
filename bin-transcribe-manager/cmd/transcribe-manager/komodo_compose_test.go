package main

import (
	"os"
	"strings"
	"testing"

	"monorepo/bin-transcribe-manager/internal/config"
)

// The deployment definition must pass the master key list to the container under the name that the
// service reads (the stored AWS credential can not be written without it).
func Test_komodoComposePassesTheMasterKeyList(t *testing.T) {
	raw, err := os.ReadFile("../../komodo/docker-compose.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	want := config.EnvConfigEncryptionKeys + "=[[BIN_MANAGER__TRANSCRIBE_MANAGER_CONFIG_ENCRYPTION_KEYS]]"
	if !strings.Contains(string(raw), "- "+want) {
		t.Errorf("Wrong match. expect the line: %s", want)
	}
}
