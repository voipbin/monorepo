package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

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
// never gets it: it does not read the database.
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

// composeEnvironment returns the environment of each service of the compose file as a map of
// variable name to value. The file is parsed, so a changed layout can not hide a wrong line.
func composeEnvironment(t *testing.T) map[string]map[string]string {
	t.Helper()

	raw, err := os.ReadFile("../../komodo/docker-compose.yml")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Environment []string `yaml:"environment"`
		} `yaml:"services"`
	}
	if errParse := yaml.Unmarshal(raw, &doc); errParse != nil {
		t.Fatalf("Wrong match. expect: valid yaml, got: %v", errParse)
	}

	res := map[string]map[string]string{}
	for name, svc := range doc.Services {
		env := map[string]string{}
		for _, item := range svc.Environment {
			key, value, found := strings.Cut(item, "=")
			if !found {
				t.Fatalf("Wrong match. %s expect: NAME=value, got: %s", name, item)
			}
			if _, dup := env[key]; dup {
				t.Errorf("Wrong match. %s sets %s twice", name, key)
			}
			env[key] = value
		}
		res[name] = env
	}

	return res
}

// The runner containers read the vendor keys and the local endpoints from their environment; the
// Go containers read none of them and send no key. The service account path stays with the
// runners and the list of the local models stays with the Go services.
func Test_komodoComposeGivesPlatformKeyVariablesToTheRunnersOnly(t *testing.T) {
	envs := composeEnvironment(t)

	keyVariables := []string{
		"OPENAI_API_KEY", "XAI_API_KEY", "GOOGLE_API_KEY", "CARTESIA_API_KEY",
		"ELEVENLABS_API_KEY", "DEEPGRAM_API_KEY", "OPENROUTER_API_KEY", "LOCAL_LLM_ENDPOINTS",
	}

	for _, name := range []string{"pipecat-script-runner-1", "pipecat-script-runner-2"} {
		env, ok := envs[name]
		if !ok {
			t.Fatalf("Wrong match. the service %s is missing", name)
		}
		for _, v := range keyVariables {
			want := "[[BIN_MANAGER__" + v + "]]"
			if got := env[v]; got != want {
				t.Errorf("Wrong match. %s %s expect: %s, got: %s", name, v, want, got)
			}
		}
		if _, ok := env["GOOGLE_APPLICATION_CREDENTIALS"]; !ok {
			t.Errorf("Wrong match. %s expect: GOOGLE_APPLICATION_CREDENTIALS", name)
		}
	}

	for _, name := range []string{"pipecat-manager-1", "pipecat-manager-2"} {
		env, ok := envs[name]
		if !ok {
			t.Fatalf("Wrong match. the service %s is missing", name)
		}
		for _, v := range keyVariables {
			if _, found := env[v]; found {
				t.Errorf("Wrong match. %s must not get %s", name, v)
			}
		}
		if _, ok := env["LOCAL_LLM_MODELS"]; !ok {
			t.Errorf("Wrong match. %s expect: LOCAL_LLM_MODELS", name)
		}
	}
}
