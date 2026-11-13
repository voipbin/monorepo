package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"monorepo/bin-pipecat-manager/internal/config"
)

// composeServiceBlocks splits the services of the compose file into blocks by the key of the
// service (two spaces of indent), so a test can tell which container gets a variable. A block
// ends at the next service key or at the next top level key.
func composeServiceBlocks(t *testing.T) map[string]string {
	t.Helper()

	raw, err := os.ReadFile("../../komodo/docker-compose.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	_, services, found := strings.Cut(string(raw), "\nservices:\n")
	if !found {
		t.Fatalf("Wrong match. the compose file has no services section")
	}

	key := regexp.MustCompile(`^  ([a-z0-9-]+):\s*$`)
	top := regexp.MustCompile(`^[a-z]`)

	blocks := map[string]string{}
	current := ""
	for _, line := range strings.Split(services, "\n") {
		if top.MatchString(line) {
			break
		}
		if m := key.FindStringSubmatch(line); m != nil {
			current = m[1]
			blocks[current] = ""
			continue
		}
		if current != "" {
			blocks[current] += line + "\n"
		}
	}

	return blocks
}

// The deployment definition passes the master key list to the two Go services (the stored
// platform keys can not be written or read without it) and to nothing else. The python runner
// never gets it: it does not read the database, it gets the keys in the request.
func Test_komodoComposePassesTheMasterKeyListToTheGoServicesOnly(t *testing.T) {
	blocks := composeServiceBlocks(t)
	want := "- " + config.EnvConfigEncryptionKeys + "=[[BIN_MANAGER__PIPECAT_MANAGER_CONFIG_ENCRYPTION_KEYS]]"

	for _, name := range []string{"pipecat-manager-1", "pipecat-manager-2"} {
		block, ok := blocks[name]
		if !ok {
			t.Fatalf("Wrong match. the service %s is missing", name)
		}
		if got := strings.Count(block, want); got != 1 {
			t.Errorf("Wrong match. %s expect the line once: %s, got: %d", name, want, got)
		}
		if !strings.Contains(block, "command: ./pipecat-manager") {
			t.Errorf("Wrong match. %s is expected to be the Go service", name)
		}
	}

	for _, name := range []string{"pipecat-script-runner-1", "pipecat-script-runner-2"} {
		block, ok := blocks[name]
		if !ok {
			t.Fatalf("Wrong match. the service %s is missing", name)
		}
		if strings.Contains(block, config.EnvConfigEncryptionKeys) || strings.Contains(block, "CONFIG_ENCRYPTION_KEYS") {
			t.Errorf("Wrong match. the runner %s must not get the master key list", name)
		}
		if !strings.Contains(block, "scripts/pipecat/main.py") {
			t.Errorf("Wrong match. %s is expected to be the python runner", name)
		}
	}

	if len(blocks) != 4 {
		t.Errorf("Wrong match. expect: 4 services, got: %d", len(blocks))
	}
}

// The master key list is mapped once per Go service and never to another variable name.
func Test_komodoComposeMasterKeyOccurrences(t *testing.T) {
	raw, err := os.ReadFile("../../komodo/docker-compose.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if got := strings.Count(string(raw), "CONFIG_ENCRYPTION_KEYS"); got != 4 {
		// two variable names per line: the container variable and the Komodo variable
		t.Errorf("Wrong match. expect: 4 mentions (2 lines), got: %d", got)
	}
}
