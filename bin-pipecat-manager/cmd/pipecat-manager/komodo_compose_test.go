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

// environmentNames returns the variable names of the "- NAME=value" lines of a service block.
// Comment lines are not variables.
func environmentNames(block string) map[string]bool {
	names := map[string]bool{}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		if name, _, found := strings.Cut(strings.TrimPrefix(line, "- "), "="); found {
			names[name] = true
		}
	}

	return names
}

// The stored platform keys are the only source of the vendor keys: no service gets a vendor key
// or the local endpoints as a variable. The service account path stays with the runners and the
// list of the local models stays with the Go services.
func Test_komodoComposeHasNoPlatformKeyVariables(t *testing.T) {
	blocks := composeServiceBlocks(t)

	keyVariables := []string{
		"OPENAI_API_KEY", "XAI_API_KEY", "GOOGLE_API_KEY", "CARTESIA_API_KEY",
		"ELEVENLABS_API_KEY", "DEEPGRAM_API_KEY", "OPENROUTER_API_KEY", "LOCAL_LLM_ENDPOINTS",
	}
	for _, name := range []string{"pipecat-manager-1", "pipecat-manager-2", "pipecat-script-runner-1", "pipecat-script-runner-2"} {
		names := environmentNames(blocks[name])
		if len(names) == 0 {
			t.Fatalf("Wrong match. the service %s has no environment variables", name)
		}
		for _, v := range keyVariables {
			if names[v] {
				t.Errorf("Wrong match. %s must not get %s", name, v)
			}
		}
	}

	for _, name := range []string{"pipecat-manager-1", "pipecat-manager-2"} {
		if !environmentNames(blocks[name])["LOCAL_LLM_MODELS"] {
			t.Errorf("Wrong match. %s expect: LOCAL_LLM_MODELS", name)
		}
	}
	for _, name := range []string{"pipecat-script-runner-1", "pipecat-script-runner-2"} {
		if !environmentNames(blocks[name])["GOOGLE_APPLICATION_CREDENTIALS"] {
			t.Errorf("Wrong match. %s expect: GOOGLE_APPLICATION_CREDENTIALS", name)
		}
	}
}
