package aiconfig

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/internal/configkeys"
)

const (
	testVendor      = configkeys.VendorLinear
	envLinearID     = "MCP_OAUTH_LINEAR_CLIENT_ID"
	envLinearSecret = "MCP_OAUTH_LINEAR_CLIENT_SECRET"
)

// storeMcpOAuth writes the stored pair of a vendor in the documented order: the client_id first,
// the client_secret (the anchor) last.
func storeMcpOAuth(t *testing.T, h confighandler.Handler, vendor, id, secret string) {
	t.Helper()

	actor := uuid.Must(uuid.NewV4())
	idKey, _ := configkeys.MCPOAuthClientIDKey(vendor)
	secretKey, _ := configkeys.MCPOAuthClientSecretKey(vendor)

	if id != "" {
		if _, err := h.Set(context.Background(), actor, idKey, id); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
	if secret != "" {
		if _, err := h.SetSecret(context.Background(), actor, secretKey, secret); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func newMcpOAuthHandler(t *testing.T, store confighandler.Store) confighandler.Handler {
	t.Helper()

	return newTestHandler(t, &config.Config{ConfigEncryptionKeys: testKeyList(1)}, store)
}

// With no stored row the pair comes from the environment, and the source is env.
func Test_MCPOAuthClient_noRowUsesTheEnvironment(t *testing.T) {
	t.Setenv(envLinearID, "env-id")
	t.Setenv(envLinearSecret, "env-secret")

	h := newMcpOAuthHandler(t, storetest.NewMemory())

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if err != nil || id != "env-id" || secret != "env-secret" {
		t.Errorf("Wrong match. expect: the environment pair, got: %q %q %v", id, secret, err)
	}

	for _, key := range []string{"mcp_oauth.linear.client_id", "mcp_oauth.linear.client_secret"} {
		res, errDescribe := describeKey(h, key)
		if errDescribe != nil || res.Source != confighandler.SourceEnv {
			t.Errorf("Wrong match. %s expect: source env, got: %+v %v", key, res, errDescribe)
		}
	}
}

// With a stored pair the database wins, and the environment is shadowed.
func Test_MCPOAuthClient_storedPairWinsOverTheEnvironment(t *testing.T) {
	t.Setenv(envLinearID, "env-id")
	t.Setenv(envLinearSecret, "env-secret")

	h := newMcpOAuthHandler(t, storetest.NewMemory())
	storeMcpOAuth(t, h, testVendor, "db-id", "db-secret")

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if err != nil || id != "db-id" || secret != "db-secret" {
		t.Errorf("Wrong match. expect: the stored pair, got: %q %q %v", id, secret, err)
	}
}

// The group is atomic. The client_secret (anchor) is stored and the client_id is not: the
// environment client_id is not used, and the result is the incomplete error.
func Test_MCPOAuthClient_anchorWithoutClientIDIsIncomplete(t *testing.T) {
	t.Setenv(envLinearID, "env-id")
	t.Setenv(envLinearSecret, "env-secret")

	h := newMcpOAuthHandler(t, storetest.NewMemory())
	storeMcpOAuth(t, h, testVendor, "", "db-secret")

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if !errors.Is(err, ErrMCPOAuthIncomplete) || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: ErrMCPOAuthIncomplete and no value, got: %q %q %v", id, secret, err)
	}
	for _, v := range []string{"env-id", "env-secret", "db-secret"} {
		if strings.Contains(err.Error(), v) {
			t.Errorf("Wrong match. the error holds a value: %v", err)
		}
	}
}

// The client_secret is cleared and the client_id row is left: the group is in environment mode
// again, so the pair is the environment pair. This is why the documented way to remove an app is to
// clear both keys: a left client_id row is ignored, and the environment pair is then used whole.
func Test_MCPOAuthClient_clearedAnchorLeavesTheEnvironmentMode(t *testing.T) {
	t.Setenv(envLinearID, "env-id")
	t.Setenv(envLinearSecret, "env-secret")

	h := newMcpOAuthHandler(t, storetest.NewMemory())
	storeMcpOAuth(t, h, testVendor, "db-id", "db-secret")

	if _, err := h.Clear(context.Background(), uuid.Must(uuid.NewV4()), "mcp_oauth.linear.client_secret"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if err != nil || id != "env-id" || secret != "env-secret" {
		t.Errorf("Wrong match. expect: the environment pair, got: %q %q %v", id, secret, err)
	}
}

// Neither stored nor in the environment is "not configured": an empty pair and no error.
func Test_MCPOAuthClient_notConfigured(t *testing.T) {
	h := newMcpOAuthHandler(t, storetest.NewMemory())

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if err != nil || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: an empty pair and no error, got: %q %q %v", id, secret, err)
	}
}

// One side alone in the environment is passed on as before (not an error).
func Test_MCPOAuthClient_environmentOneSideKeepsTheCurrentBehavior(t *testing.T) {
	t.Setenv(envLinearID, "env-id")

	h := newMcpOAuthHandler(t, storetest.NewMemory())

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if err != nil || id != "env-id" || secret != "" {
		t.Errorf("Wrong match. expect: the id alone and no error, got: %q %q %v", id, secret, err)
	}
}

// A read that fails is ErrKeyUnavailable and never the environment pair.
func Test_MCPOAuthClient_readErrorIsNotTheEnvironment(t *testing.T) {
	const envID, envSecret = "env-id-zz1", "env-secret-zz1"
	t.Setenv(envLinearID, envID)
	t.Setenv(envLinearSecret, envSecret)

	actor := uuid.Must(uuid.NewV4())

	tests := []struct {
		name  string
		setup func(t *testing.T) confighandler.Handler
	}{
		{
			name: "database read error",
			setup: func(t *testing.T) confighandler.Handler {
				store := storetest.NewMemory()
				store.GetErr = errors.New("database is down")
				return newMcpOAuthHandler(t, store)
			},
		},
		{
			name: "only the client_id row fails to read",
			setup: func(t *testing.T) confighandler.Handler {
				store := storetest.NewMemory()
				storeMcpOAuth(t, newMcpOAuthHandler(t, store), testVendor, "db-id", "db-secret")
				return newMcpOAuthHandler(t, &failingKeyStore{Store: store, key: "mcp_oauth.linear.client_id"})
			},
		},
		{
			name: "undecryptable (no master key)",
			setup: func(t *testing.T) confighandler.Handler {
				store := storetest.NewMemory()
				storeMcpOAuth(t, newMcpOAuthHandler(t, store), testVendor, "db-id", "db-secret")
				return newTestHandler(t, &config.Config{}, store)
			},
		},
		{
			name: "undecryptable (wrong master key)",
			setup: func(t *testing.T) confighandler.Handler {
				store := storetest.NewMemory()
				storeMcpOAuth(t, newMcpOAuthHandler(t, store), testVendor, "db-id", "db-secret")
				return newTestHandler(t, &config.Config{ConfigEncryptionKeys: "1:" + strings.Repeat("A", 43) + "="}, store)
			},
		},
		{
			name: "stored client_id of a wrong type",
			setup: func(t *testing.T) confighandler.Handler {
				store := storetest.NewMemory()
				h := newMcpOAuthHandler(t, store)
				storeMcpOAuth(t, h, testVendor, "db-id", "db-secret")
				// Replace the client_id row by a row of another type: reads as invalid.
				row, err := store.Get(context.Background(), "mcp_oauth.linear.client_id")
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				row.ValueType = "int"
				hist := &confighandler.History{ID: uuid.Must(uuid.NewV4()), Key: row.Key, Action: "set", ActorID: actor, Source: "test"}
				if err := store.Update(context.Background(), row, row.Revision, hist); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				return h
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setup(t)

			id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
			if !errors.Is(err, ErrKeyUnavailable) || id != "" || secret != "" {
				t.Fatalf("Wrong match. expect: ErrKeyUnavailable and no value, got: %q %q %v", id, secret, err)
			}
			for _, v := range []string{envID, envSecret, "db-secret", "db-id"} {
				if strings.Contains(err.Error(), v) {
					t.Errorf("Wrong match. the error holds a value: %v", err)
				}
			}
		})
	}
}

// failingKeyStore fails the read of one key and passes everything else on.
type failingKeyStore struct {
	confighandler.Store
	key string
}

func (s *failingKeyStore) Get(ctx context.Context, key string) (*confighandler.Row, error) {
	if key == s.key {
		return nil, errors.New("database is down")
	}

	return s.Store.Get(ctx, key)
}

// hookStore runs a function before the n-th read of one key.
type hookStore struct {
	confighandler.Store
	key  string
	n    int
	hook func()

	mu    sync.Mutex
	count int
}

func (s *hookStore) Get(ctx context.Context, key string) (*confighandler.Row, error) {
	if key == s.key {
		s.mu.Lock()
		s.count++
		fire := s.count == s.n
		s.mu.Unlock()

		if fire {
			s.hook()
		}
	}

	return s.Store.Get(ctx, key)
}

// A pair that is written while it is read (the anchor lands between the two reads) is never
// returned half and half: the read is repeated and gives the stored pair.
func Test_MCPOAuthClient_writeBetweenTheReadsGivesNoMixedPair(t *testing.T) {
	t.Setenv(envLinearID, "env-id")
	t.Setenv(envLinearSecret, "env-secret")

	mem := storetest.NewMemory()
	writer := newMcpOAuthHandler(t, mem)

	// The anchor is read by the first Secret call (count 1), and again inside the String call
	// (count 2). The pair is stored just before that second read.
	store := &hookStore{Store: mem, key: "mcp_oauth.linear.client_secret", n: 2, hook: func() {
		storeMcpOAuth(t, writer, testVendor, "db-id", "db-secret")
	}}
	h := newMcpOAuthHandler(t, store)

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if id != "db-id" || secret != "db-secret" {
		t.Errorf("Wrong match. expect: the stored pair, got: %q %q", id, secret)
	}
}

// If the pair keeps changing under the read, the result is an error and not a mixed pair.
func Test_MCPOAuthClient_keepsDisagreeingIsAnError(t *testing.T) {
	mem := storetest.NewMemory()
	writer := newMcpOAuthHandler(t, mem)
	storeMcpOAuth(t, writer, testVendor, "db-id", "")

	// Whenever the anchor is read, the anchor row is absent for the Secret call and present for
	// the client_id read: simulate it with a store that answers the first anchor read of each
	// attempt with not found.
	store := &flippingAnchorStore{Store: mem, anchor: "mcp_oauth.linear.client_secret"}
	storeMcpOAuth(t, writer, testVendor, "", "db-secret")
	h := newMcpOAuthHandler(t, store)

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if !errors.Is(err, ErrKeyUnavailable) || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: ErrKeyUnavailable and no value, got: %q %q %v", id, secret, err)
	}
}

// flippingAnchorStore answers every odd read of the anchor as missing, and every even one with the
// stored row.
type flippingAnchorStore struct {
	confighandler.Store
	anchor string

	mu    sync.Mutex
	count int
}

func (s *flippingAnchorStore) Get(ctx context.Context, key string) (*confighandler.Row, error) {
	if key == s.anchor {
		s.mu.Lock()
		s.count++
		odd := s.count%2 == 1
		s.mu.Unlock()

		if odd {
			return nil, confighandler.ErrNotFound
		}
	}

	return s.Store.Get(ctx, key)
}

func Test_MCPOAuthClient_unknownVendor(t *testing.T) {
	h := newMcpOAuthHandler(t, storetest.NewMemory())

	if _, _, err := MCPOAuthClient(context.Background(), h, "not-a-vendor"); err == nil {
		t.Error("Wrong match. expect: an error for an unknown vendor")
	}
}

// Every vendor of the registry resolves through its own keys.
func Test_MCPOAuthClient_everyVendor(t *testing.T) {
	for _, vendor := range configkeys.MCPOAuthVendors() {
		t.Run(vendor, func(t *testing.T) {
			h := newMcpOAuthHandler(t, storetest.NewMemory())
			storeMcpOAuth(t, h, vendor, "id-"+vendor, "secret-"+vendor)

			id, secret, err := MCPOAuthClient(context.Background(), h, vendor)
			if err != nil || id != "id-"+vendor || secret != "secret-"+vendor {
				t.Errorf("Wrong match. expect: the stored pair, got: %q %q %v", id, secret, err)
			}
		})
	}
}

// A read that runs out of time is ErrKeyUnavailable and keeps its cause.
func Test_MCPOAuthClient_timeout(t *testing.T) {
	h := newTestHandler(t, &config.Config{}, blockingStore{Store: storetest.NewMemory()})

	started := time.Now()
	_, _, err := MCPOAuthClient(context.Background(), h, testVendor, WithTimeout(20*time.Millisecond))
	if !errors.Is(err, ErrKeyUnavailable) || !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 2*time.Second {
		t.Errorf("Wrong match. expect: a prompt ErrKeyUnavailable with the deadline, got: %v after %v", err, time.Since(started))
	}
}

// A client_id of the environment that is longer than the key allows is an error and not a value.
func Test_MCPOAuthClient_environmentClientIDTooLongIsAnError(t *testing.T) {
	t.Setenv(envLinearID, strings.Repeat("a", 257))
	t.Setenv(envLinearSecret, "env-secret")

	h := newMcpOAuthHandler(t, storetest.NewMemory())

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if !errors.Is(err, ErrKeyUnavailable) || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: ErrKeyUnavailable and no value, got: %q %q %v", id, secret, err)
	}
}

// The names of the environment variables of the registry are the names New reads: every key that
// declares an environment name gets its value from os.LookupEnv, under exactly that name. A name
// that differs by one letter would leave the app silently not configured.
func Test_New_readsEveryDeclaredEnvironmentName(t *testing.T) {
	var declared int
	for _, k := range configkeys.Keys() {
		if k.Env == "" {
			continue
		}
		declared++
		t.Setenv(k.Env, "value-of-"+k.Env)
	}
	if declared != 8 {
		t.Fatalf("Wrong match. expect: 8 keys with an environment name, got: %d", declared)
	}

	h := newMcpOAuthHandler(t, storetest.NewMemory())

	for _, vendor := range configkeys.MCPOAuthVendors() {
		upper := strings.ToUpper(vendor)
		id, secret, err := MCPOAuthClient(context.Background(), h, vendor)
		if err != nil || id != "value-of-MCP_OAUTH_"+upper+"_CLIENT_ID" || secret != "value-of-MCP_OAUTH_"+upper+"_CLIENT_SECRET" {
			t.Errorf("Wrong match. %s expect: the values of its variables, got: %q %q %v", vendor, id, secret, err)
		}
	}

	// The LLM keys have no environment name: their old variables change nothing.
	t.Setenv("ENGINE_KEY_CHATGPT", "synthetic-old-env-openai")
	t.Setenv("GOOGLE_API_KEY", "synthetic-old-env-gemini")
	for _, key := range []string{configkeys.KeyOpenAICompatAPIKey, configkeys.KeyGeminiAPIKey} {
		got, err := Supplier(h, key)(context.Background())
		if err != nil || got != "" {
			t.Errorf("Wrong match. %s expect: empty, got: %q %v", key, got, err)
		}
	}
}

// A stored but empty client_id (the parameter accepts an empty value) with a stored client_secret
// is as incomplete as a missing client_id row, and the environment is not used for it.
func Test_MCPOAuthClient_storedEmptyClientIDIsIncomplete(t *testing.T) {
	t.Setenv(envLinearID, "env-id")
	t.Setenv(envLinearSecret, "env-secret")

	h := newMcpOAuthHandler(t, storetest.NewMemory())
	actor := uuid.Must(uuid.NewV4())
	idKey, _ := configkeys.MCPOAuthClientIDKey(testVendor)
	if _, err := h.Set(context.Background(), actor, idKey, ""); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	storeMcpOAuth(t, h, testVendor, "", "db-secret")

	id, secret, err := MCPOAuthClient(context.Background(), h, testVendor)
	if !errors.Is(err, ErrMCPOAuthIncomplete) || id != "" || secret != "" {
		t.Errorf("Wrong match. expect: ErrMCPOAuthIncomplete and no value, got: %q %q %v", id, secret, err)
	}
}
