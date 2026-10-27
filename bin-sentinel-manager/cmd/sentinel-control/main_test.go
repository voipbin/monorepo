package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

// fakeDocker is a loopback-only stand-in for the docker-socket-proxy. It answers just the three
// endpoints sentinel-control uses and records every request path so a test can assert that a
// failing precondition never reached the Docker API.
type fakeDocker struct {
	srv *httptest.Server

	mu       sync.Mutex
	requests []string
	queries  []string

	listStatus int
	listBody   string

	inspect map[string]string
}

func newFakeDocker(t *testing.T) *fakeDocker {
	t.Helper()

	f := &fakeDocker{listStatus: http.StatusOK, listBody: "[]", inspect: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.srv.Close)

	return f
}

func (f *fakeDocker) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// the client pings first (HEAD) to negotiate the API version. A fixed version below the
	// client default makes it prefix every following path with /v1.43/.
	if r.URL.Path == "/_ping" {
		w.Header().Set("Api-Version", "1.43")
		w.WriteHeader(http.StatusOK)
		return
	}

	f.mu.Lock()
	f.requests = append(f.requests, r.URL.Path)
	f.queries = append(f.queries, r.URL.RawQuery)
	f.mu.Unlock()

	if r.URL.Path == "/v1.43/containers/json" {
		w.WriteHeader(f.listStatus)
		_, _ = io.WriteString(w, f.listBody)
		return
	}

	const prefix = "/v1.43/containers/"
	if strings.HasPrefix(r.URL.Path, prefix) && strings.HasSuffix(r.URL.Path, "/json") {
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, prefix), "/json")
		if body, ok := f.inspect[name]; ok {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, body)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"No such container: `+name+`"}`)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	_, _ = io.WriteString(w, `{"message":"unexpected path"}`)
}

func (f *fakeDocker) address() string {
	return "tcp://" + strings.TrimPrefix(f.srv.URL, "http://")
}

func (f *fakeDocker) requestLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.requests...)
}

func (f *fakeDocker) lastQuery() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.queries) == 0 {
		return ""
	}
	return f.queries[len(f.queries)-1]
}

// runCLI executes the real command tree with the given arguments and returns what it printed to
// stdout. Every environment variable the config and flag bindings read is set explicitly, so a
// value present on a developer or CI machine (NAME and ALL in particular, which the flag
// bindings would otherwise pick up through AutomaticEnv) cannot change the outcome.
func runCLI(t *testing.T, proxyAddress string, backend string, args ...string) (string, error) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("NAME", "")
	t.Setenv("ALL", "")
	t.Setenv("SENTINEL_BACKEND", backend)
	t.Setenv("DOCKER_SOCKET_PROXY_ADDRESS", proxyAddress)
	t.Setenv("REDIS_ADDRESS", "127.0.0.1:6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DATABASE", "1")

	cmd := initCommand()
	cmd.SetArgs(args)
	cmd.SetErr(io.Discard)
	cmd.SetOut(io.Discard)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	return captureStdout(t, func() error { return cmd.Execute() })
}

// captureStdout redirects os.Stdout for the duration of fn. The reader runs concurrently so a
// large output cannot fill the pipe and deadlock, and stdout is always restored.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	reader, writer, errPipe := os.Pipe()
	if errPipe != nil {
		t.Fatalf("Could not create the pipe. err: %v", errPipe)
	}

	orig := os.Stdout
	os.Stdout = writer

	done := make(chan string)
	go func() {
		data, _ := io.ReadAll(reader)
		done <- string(data)
	}()

	err := func() error {
		defer func() {
			os.Stdout = orig
			_ = writer.Close()
		}()
		return fn()
	}()

	out := <-done
	_ = reader.Close()

	return out, err
}

const listFixture = `[
  {"Id":"id-1","Names":["/voip-asterisk-call-docker-1"],"Image":"asterisk:1","State":"running","Status":"Up 2 hours",
   "NetworkSettings":{"Networks":{"net-a":{"IPAddress":"192.0.2.10"},"net-b":null}}},
  {"Id":"id-2","Names":["/voip-asterisk-conference-docker-1"],"Image":"asterisk:2","State":"exited","Status":"Exited (0) 1 hour ago",
   "NetworkSettings":{"Networks":{"net-a":{"IPAddress":"192.0.2.20"}}}},
  {"Id":"id-3","Names":[],"Image":"nameless:1","State":"running","Status":"Up 5 minutes"}
]`

func Test_runContainerList(t *testing.T) {
	tests := []struct {
		name string

		args []string

		expectContainers []ContainerInfo
		expectQuery      string
	}{
		{
			name: "no filter returns every container with trimmed names and per-network addresses",

			args: []string{"container", "list"},

			expectContainers: []ContainerInfo{
				{ID: "id-1", Name: "voip-asterisk-call-docker-1", Image: "asterisk:1", State: "running", Status: "Up 2 hours", Addresses: map[string]string{"net-a": "192.0.2.10"}},
				{ID: "id-2", Name: "voip-asterisk-conference-docker-1", Image: "asterisk:2", State: "exited", Status: "Exited (0) 1 hour ago", Addresses: map[string]string{"net-a": "192.0.2.20"}},
				{ID: "id-3", Name: "", Image: "nameless:1", State: "running", Status: "Up 5 minutes", Addresses: map[string]string{}},
			},
			expectQuery: "",
		},
		{
			name: "name filter keeps only matching containers",

			args: []string{"container", "list", "--name", "conference"},

			expectContainers: []ContainerInfo{
				{ID: "id-2", Name: "voip-asterisk-conference-docker-1", Image: "asterisk:2", State: "exited", Status: "Exited (0) 1 hour ago", Addresses: map[string]string{"net-a": "192.0.2.20"}},
			},
			expectQuery: "",
		},
		{
			name: "name filter matching nothing prints an empty list",

			args: []string{"container", "list", "--name", "no-such-container"},

			expectContainers: []ContainerInfo{},
			expectQuery:      "",
		},
		{
			name: "all flag is forwarded to the docker query",

			args: []string{"container", "list", "--all", "--name", "call"},

			expectContainers: []ContainerInfo{
				{ID: "id-1", Name: "voip-asterisk-call-docker-1", Image: "asterisk:1", State: "running", Status: "Up 2 hours", Addresses: map[string]string{"net-a": "192.0.2.10"}},
			},
			expectQuery: "all=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeDocker(t)
			fake.listBody = listFixture

			out, err := runCLI(t, fake.address(), "docker", tt.args...)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var res []ContainerInfo
			if errUnmarshal := json.Unmarshal([]byte(out), &res); errUnmarshal != nil {
				t.Fatalf("Could not decode the output. out: %q, err: %v", out, errUnmarshal)
			}
			if res == nil {
				res = []ContainerInfo{}
			}

			if !reflect.DeepEqual(res, tt.expectContainers) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expectContainers, res)
			}
			if got := fake.lastQuery(); got != tt.expectQuery {
				t.Errorf("Wrong match. expect query: %q, got: %q", tt.expectQuery, got)
			}
		})
	}
}

func Test_runContainerList_dockerError(t *testing.T) {
	fake := newFakeDocker(t)
	fake.listStatus = http.StatusInternalServerError
	fake.listBody = `{"message":"daemon unavailable"}`

	out, err := runCLI(t, fake.address(), "docker", "container", "list")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "failed to list containers") {
		t.Errorf("Wrong match. expect: a wrapped list error, got: %v", err)
	}
	if out != "" {
		t.Errorf("Wrong match. expect: no output on failure, got: %q", out)
	}
}

func Test_runContainerGet(t *testing.T) {
	const full = `{
  "Id":"id-1","Name":"/voip-asterisk-call-docker-1",
  "State":{"Status":"running","StartedAt":"2026-01-02T03:04:05Z"},
  "Config":{"Image":"asterisk:1"},
  "NetworkSettings":{"Networks":{
    "net-a":{"IPAddress":"192.0.2.10","MacAddress":"02:42:c0:00:02:0a"},
    "net-b":null}}
}`

	tests := []struct {
		name string

		body string

		expect ContainerDetailInfo
	}{
		{
			name: "fully populated inspect response",

			body: full,

			expect: ContainerDetailInfo{
				ID:        "id-1",
				Name:      "voip-asterisk-call-docker-1",
				Image:     "asterisk:1",
				State:     "running",
				StartedAt: "2026-01-02T03:04:05Z",
				Addresses: map[string]string{"net-a": "192.0.2.10"},
				MACs:      map[string]string{"net-a": "02:42:c0:00:02:0a"},
			},
		},
		{
			name: "response without the container base leaves identity fields empty",

			body: `{}`,

			expect: ContainerDetailInfo{Addresses: map[string]string{}, MACs: map[string]string{}},
		},
		{
			name: "response with an id but no state, config or network settings",

			body: `{"Id":"id-2","Name":"/bare"}`,

			expect: ContainerDetailInfo{ID: "id-2", Name: "bare", Addresses: map[string]string{}, MACs: map[string]string{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := newFakeDocker(t)
			fake.inspect["target"] = tt.body

			out, err := runCLI(t, fake.address(), "docker", "container", "get", "--name", "target")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			var res ContainerDetailInfo
			if errUnmarshal := json.Unmarshal([]byte(out), &res); errUnmarshal != nil {
				t.Fatalf("Could not decode the output. out: %q, err: %v", out, errUnmarshal)
			}

			if !reflect.DeepEqual(res, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expect, res)
			}
		})
	}
}

func Test_runContainerGet_notFound(t *testing.T) {
	fake := newFakeDocker(t)

	_, err := runCLI(t, fake.address(), "docker", "container", "get", "--name", "missing")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "failed to inspect container") {
		t.Errorf("Wrong match. expect: a wrapped inspect error, got: %v", err)
	}
}

func Test_runContainerGet_nameRequired(t *testing.T) {
	fake := newFakeDocker(t)

	_, err := runCLI(t, fake.address(), "docker", "container", "get")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	// the exact text matters: Docker's own client also rejects an empty id, with a different
	// message, so a looser assertion could not tell this guard from the client's.
	if err.Error() != "container name is required" {
		t.Errorf("Wrong match. expect: container name is required, got: %v", err)
	}
	if got := fake.requestLog(); len(got) != 0 {
		t.Errorf("Wrong match. expect: no docker request, got: %v", got)
	}
}

func Test_initCommand_configValidation(t *testing.T) {
	tests := []struct {
		name string

		backend string
		address string
		args    []string

		expectErr string
	}{
		{
			name: "an unset backend is refused before any subcommand runs",

			backend:   "",
			address:   "tcp://127.0.0.1:1",
			args:      []string{"container", "list"},
			expectErr: "failed to load config",
		},
		{
			name: "an unknown backend is refused",

			backend:   "nomad",
			address:   "tcp://127.0.0.1:1",
			args:      []string{"container", "list"},
			expectErr: "failed to load config",
		},
		{
			name: "docker backend with an empty proxy address given by flag is refused",

			backend:   "docker",
			address:   "tcp://127.0.0.1:1",
			args:      []string{"container", "list", "--docker_socket_proxy_address="},
			expectErr: "failed to load config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runCLI(t, tt.address, tt.backend, tt.args...)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
			if out != "" {
				t.Errorf("Wrong match. expect: no output, got: %q", out)
			}
		})
	}
}

func Test_newDockerClient_invalidHost(t *testing.T) {
	// both addresses pass config validation (non-empty) and are rejected only when the docker
	// client is constructed.
	for _, address := range []string{"no-scheme", "tcp://"} {
		t.Run(address, func(t *testing.T) {
			_, err := runCLI(t, address, "docker", "container", "list")
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), "failed to create docker client") {
				t.Errorf("Wrong match. expect: a client construction error, got: %v", err)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("encodes with two-space indentation", func(t *testing.T) {
		out, err := captureStdout(t, func() error {
			return printJSON(map[string]int{"a": 1})
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if out != "{\n  \"a\": 1\n}\n" {
			t.Errorf("Wrong match. got: %q", out)
		}
	})

	t.Run("a value that cannot be encoded returns an error and prints nothing", func(t *testing.T) {
		out, err := captureStdout(t, func() error {
			return printJSON(make(chan int))
		})
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Errorf("Wrong match. expect: a marshal error, got: %v", err)
		}
		if out != "" {
			t.Errorf("Wrong match. expect: no output, got: %q", out)
		}
	})
}

func Test_runContainerGet_invalidHost(t *testing.T) {
	// the address passes config validation (non-empty) and is rejected only when the docker
	// client is constructed, so the failure must surface before any inspect request.
	_, err := runCLI(t, "no-scheme", "docker", "container", "get", "--name", "voip-asterisk-call-docker-1")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "failed to create docker client") {
		t.Errorf("Wrong match. expect: a client construction error, got: %v", err)
	}
}

func Test_main_helpPrintsUsage(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	previousArgs := os.Args
	os.Args = []string{"sentinel-control", "--help"}
	t.Cleanup(func() { os.Args = previousArgs })

	out, err := captureStdout(t, func() error {
		main()
		return nil
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !strings.Contains(out, "Voipbin Sentinel Management CLI") {
		t.Errorf("Wrong match. expect: the CLI description in the help output, got: %q", out)
	}
}
