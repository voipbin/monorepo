package configrpc_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
	"monorepo/bin-common-handler/pkg/secretcrypto"
)

const (
	fakeSecret = "test-secret-not-real"
	fakeEnv    = "env-value-not-real"
)

var testActor = uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111")

func testKeyList() string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = 1 + byte(i)
	}

	return "1:" + base64.StdEncoding.EncodeToString(k)
}

func testRegistry(t *testing.T) *confighandler.Registry {
	t.Helper()

	reg, err := confighandler.NewRegistry([]confighandler.Key{
		{Name: "a.api_key", Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "The key of A.", Env: "A_API_KEY", MaxLen: 32},
		{Name: "b.api_key", Kind: confighandler.KindSecret, Type: confighandler.TypeString, Description: "The key of B.", MaxLen: 32},
		{Name: "c.limit", Kind: confighandler.KindParam, Type: confighandler.TypeInt, Description: "A limit.", Env: "C_LIMIT", Default: 10},
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return reg
}

type fixture struct {
	server *configrpc.Server
	h      confighandler.Handler
	store  *storetest.Memory
	env    map[string]string
}

func newFixture(t *testing.T, keyList string) *fixture {
	t.Helper()

	crypto, err := secretcrypto.New(keyList)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	f := &fixture{store: storetest.NewMemory(), env: map[string]string{}}
	f.h = confighandler.NewHandler(testRegistry(t), f.store, crypto, func(name string) (string, bool) {
		v, ok := f.env[name]
		return v, ok
	})
	f.server = configrpc.NewServer(f.h, outline.ServiceNameEmailManager)

	return f
}

func req(method sock.RequestMethod, uri string, body any) *sock.Request {
	m := &sock.Request{URI: uri, Method: method, DataType: "application/json"}
	if body != nil {
		if s, ok := body.(string); ok {
			m.Data = json.RawMessage(s)
		} else {
			m.Data, _ = json.Marshal(body)
		}
	}

	return m
}

func (f *fixture) do(t *testing.T, m *sock.Request) *sock.Response {
	t.Helper()

	res, err := f.server.Handle(context.Background(), m)
	if err != nil {
		t.Fatalf("Wrong match. Handle returns its errors as responses, got: %v", err)
	}
	if res == nil {
		t.Fatalf("Wrong match. expect: a response, got: nil")
	}

	return res
}

func typedError(t *testing.T, res *sock.Response) *cerrors.VoipbinError {
	t.Helper()

	ve := cerrors.FromResponse(res)
	if ve == nil {
		t.Fatalf("Wrong match. expect: a typed error, got: %d %s %s", res.StatusCode, res.DataType, res.Data)
	}

	return ve
}

func Test_Match(t *testing.T) {
	f := newFixture(t, testKeyList())

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		expect bool
	}{
		{"service", sock.RequestMethodGet, "/v1/configs", true},
		{"secret put", sock.RequestMethodPut, "/v1/configs/a.api_key/secret", true},
		{"key delete", sock.RequestMethodDelete, "/v1/configs/a.api_key", true},
		{"histories", sock.RequestMethodGet, "/v1/configs/a.api_key/histories", true},
		{"histories with query", sock.RequestMethodGet, "/v1/configs/a.api_key/histories?page_size=2&page_token=abc", true},
		{"service post", sock.RequestMethodPost, "/v1/configs", false},
		{"secret get", sock.RequestMethodGet, "/v1/configs/a.api_key/secret", false},
		{"prefix only", sock.RequestMethodGet, "/v1/configsx", false},
		{"deeper path", sock.RequestMethodGet, "/v1/configs/a.api_key/histories/x", false},
		{"key with an upper case letter", sock.RequestMethodDelete, "/v1/configs/A.api_key", false},
		{"key with a slash", sock.RequestMethodDelete, "/v1/configs/a/b", false},
		{"key starting with a dot", sock.RequestMethodDelete, "/v1/configs/.a", false},
		{"upper case route", sock.RequestMethodPut, "/V1/CONFIGS/a.api_key/SECRET", false},
		{"other resource", sock.RequestMethodGet, "/v1/emails", false},
		{"empty", sock.RequestMethodGet, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.server.Match(&sock.Request{URI: tt.uri, Method: tt.method}); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}

	// a listener without a config server keeps its other routes
	var none *configrpc.Server
	if none.Match(&sock.Request{URI: "/v1/configs", Method: sock.RequestMethodGet}) {
		t.Errorf("Wrong match. a nil server must not match")
	}
}

func Test_LogSafe(t *testing.T) {
	body := json.RawMessage(`{"secret":"` + fakeSecret + `"}`)

	tests := []struct {
		name   string
		uri    string
		expect bool
	}{
		{"secret route", "/v1/configs/a.api_key/secret", true},
		{"service route", "/v1/configs", true},
		{"delete route", "/v1/configs/a.api_key", true},
		{"typo in the resource", "/v1/config/a.api_key/secret", true},
		{"typo only in the prefix", "/v1/configss/a.api_key", true},
		{"upper case", "/V1/CONFIGS/A.API_KEY/SECRET", true},
		{"mixed case suffix", "/v1/anything/Secret", true},
		{"unrelated", "/v1/emails", false},
		{"unrelated with a query", "/v1/emails?page_size=5", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &sock.Request{URI: tt.uri, Method: sock.RequestMethodPut, Data: body}

			got := configrpc.LogSafe(m)
			if !tt.expect {
				if got != m {
					t.Errorf("Wrong match. expect: the same request")
				}
				return
			}

			if got == m {
				t.Fatalf("Wrong match. expect: a copy")
			}
			if strings.Contains(fmt.Sprintf("%v %+v %s", got, got, got.Data), fakeSecret) {
				t.Errorf("Wrong match. the copy still holds the secret: %+v", got)
			}
			if got.URI != m.URI || got.Method != m.Method {
				t.Errorf("Wrong match. the copy changed more than the body: %+v", got)
			}
			// the original is untouched: the handler still needs the body
			if !strings.Contains(string(m.Data), fakeSecret) {
				t.Errorf("Wrong match. the original request was changed")
			}
		})
	}
}

// walk returns the JSON field names of a decoded document, nested names included.
func walk(v any, into map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			into[k] = true
			walk(child, into)
		}
	case []any:
		for _, child := range x {
			walk(child, into)
		}
	}
}

// The response of every route has exactly the documented fields: no secret, no length, no hash,
// no environment value, no ciphertext.
func Test_responseFieldWhitelist(t *testing.T) {
	f := newFixture(t, testKeyList())
	f.env["A_API_KEY"] = fakeEnv
	f.env["C_LIMIT"] = "77"

	allowed := map[string]bool{
		// Service (the manager does not send "service": the request handler adds it)
		"secret_writable": true, "keys": true,
		// Key
		"key": true, "kind": true, "type": true, "description": true, "source": true, "has_secret": true,
		"env_name": true, "env_shadowed": true, "key_version": true, "revision": true, "updated_by": true, "tm_update": true,
		// History list
		"result": true, "next_page_token": true,
		// History
		"id": true, "action": true, "is_secret": true, "old_key_version": true, "new_key_version": true, "actor_id": true, "tm_create": true,
	}
	forbidden := []string{"secret", "value", "length", "len", "hash", "ciphertext", "env_value", "old_value", "new_value", "plaintext"}

	responses := map[string]*sock.Response{}
	responses["service before"] = f.do(t, req(sock.RequestMethodGet, "/v1/configs", nil))
	responses["put"] = f.do(t, req(sock.RequestMethodPut, "/v1/configs/a.api_key/secret", map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 0}))
	responses["put again"] = f.do(t, req(sock.RequestMethodPut, "/v1/configs/a.api_key/secret", map[string]any{"actor_id": testActor, "secret": fakeSecret + "2", "revision": 1}))
	responses["service after"] = f.do(t, req(sock.RequestMethodGet, "/v1/configs", nil))
	responses["histories"] = f.do(t, req(sock.RequestMethodGet, "/v1/configs/a.api_key/histories", nil))
	responses["delete"] = f.do(t, req(sock.RequestMethodDelete, "/v1/configs/a.api_key", map[string]any{"actor_id": testActor, "revision": 2}))

	for name, res := range responses {
		t.Run(name, func(t *testing.T) {
			if res.StatusCode != http.StatusOK {
				t.Fatalf("Wrong match. expect: 200, got: %d %s", res.StatusCode, res.Data)
			}

			if strings.Contains(string(res.Data), fakeSecret) || strings.Contains(string(res.Data), fakeEnv) {
				t.Errorf("Wrong match. the response holds a secret or an environment value: %s", res.Data)
			}

			var doc any
			if err := json.Unmarshal(res.Data, &doc); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			names := map[string]bool{}
			walk(doc, names)
			for n := range names {
				if !allowed[n] {
					t.Errorf("Wrong match. the field %q is not in the whitelist", n)
				}
				for _, bad := range forbidden {
					if n == bad {
						t.Errorf("Wrong match. the forbidden field %q is in the response", n)
					}
				}
			}
		})
	}
}

func Test_service(t *testing.T) {
	f := newFixture(t, testKeyList())
	f.env["A_API_KEY"] = fakeEnv

	res := f.do(t, req(sock.RequestMethodGet, "/v1/configs", nil))
	var got configrpc.Service
	if err := json.Unmarshal(res.Data, &got); err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. got: %d %s %v", res.StatusCode, res.Data, err)
	}

	if !got.SecretWritable || len(got.Keys) != 3 {
		t.Fatalf("Wrong match. got: %+v", got)
	}

	a := got.Keys[0]
	expectA := configrpc.Key{Key: "a.api_key", Kind: "secret", Type: "string", Description: "The key of A.", Source: "env", HasSecret: true, EnvName: "A_API_KEY"}
	if !reflect.DeepEqual(a, expectA) {
		t.Errorf("Wrong match. expect: %+v, got: %+v", expectA, a)
	}

	b := got.Keys[1]
	expectB := configrpc.Key{Key: "b.api_key", Kind: "secret", Type: "string", Description: "The key of B.", Source: "none"}
	if !reflect.DeepEqual(b, expectB) {
		t.Errorf("Wrong match. expect: %+v, got: %+v", expectB, b)
	}

	if c := got.Keys[2]; c.Kind != "param" || c.EnvName != "C_LIMIT" || c.Source != "default" || c.HasSecret {
		t.Errorf("Wrong match. got: %+v", c)
	}

	// without a master key the secrets can not be written
	noKey := newFixture(t, "")
	res = noKey.do(t, req(sock.RequestMethodGet, "/v1/configs", nil))
	got = configrpc.Service{}
	if err := json.Unmarshal(res.Data, &got); err != nil || got.SecretWritable {
		t.Errorf("Wrong match. got: %+v %v", got, err)
	}
}

func putSecret(key string, body any) *sock.Request {
	return req(sock.RequestMethodPut, "/v1/configs/"+key+"/secret", body)
}

func Test_putSecret(t *testing.T) {
	f := newFixture(t, testKeyList())

	res := f.do(t, putSecret("a.api_key", map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 0}))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}

	var got configrpc.Key
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got.Key != "a.api_key" || got.Source != "db" || !got.HasSecret || got.Revision != 1 || got.KeyVersion != 1 || got.UpdatedBy == nil || *got.UpdatedBy != testActor {
		t.Errorf("Wrong match. got: %+v", got)
	}

	// the stored value is the secret: the next read through the handler returns it
	if secret, _, err := f.h.Secret(context.Background(), "a.api_key"); err != nil || secret.Reveal() != fakeSecret {
		t.Errorf("Wrong match. the secret was not stored: %v", err)
	}

	// the history row is written with the api source and without a value
	hists := f.store.Histories()
	if len(hists) != 1 || hists[0].Source != "api" || hists[0].ActorID != testActor || hists[0].NewValueJSON != nil {
		t.Errorf("Wrong match. got: %+v", hists)
	}
}

func Test_putSecret_errors(t *testing.T) {
	long := strings.Repeat("a", 33)

	tests := []struct {
		name string
		key  string
		body any

		prepare func(f *fixture)

		expectStatus int
		expectReason string
		expectCode   cerrors.Status
		noKeyList    bool
	}{
		{name: "stale revision", key: "a.api_key", body: map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 5},
			expectStatus: 409, expectReason: "REVISION_MISMATCH", expectCode: cerrors.StatusFailedPrecondition},
		{name: "revision zero with a row", key: "a.api_key", body: map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 0},
			prepare:      func(f *fixture) { _, _ = f.h.SetSecret(context.Background(), testActor, "a.api_key", "old") },
			expectStatus: 409, expectReason: "REVISION_MISMATCH", expectCode: cerrors.StatusFailedPrecondition},
		{name: "unknown key", key: "nothing.here", body: map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 0},
			expectStatus: 404, expectReason: "CONFIG_KEY_NOT_FOUND", expectCode: cerrors.StatusNotFound},
		{name: "not a secret key", key: "c.limit", body: map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 0},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "too long", key: "a.api_key", body: map[string]any{"actor_id": testActor, "secret": long, "revision": 0},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "empty secret", key: "a.api_key", body: map[string]any{"actor_id": testActor, "secret": "", "revision": 0},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "missing secret", key: "a.api_key", body: map[string]any{"actor_id": testActor, "revision": 0},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "missing revision", key: "a.api_key", body: map[string]any{"actor_id": testActor, "secret": fakeSecret},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "missing actor", key: "a.api_key", body: map[string]any{"secret": fakeSecret, "revision": 0},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "nil actor", key: "a.api_key", body: map[string]any{"actor_id": uuid.Nil, "secret": fakeSecret, "revision": 0},
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "negative revision", key: "a.api_key", body: `{"actor_id":"11111111-1111-4111-8111-111111111111","secret":"x","revision":-1}`,
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "not json", key: "a.api_key", body: `{"secret":"` + fakeSecret,
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "no body", key: "a.api_key", body: nil,
			expectStatus: 400, expectReason: "CONFIG_INVALID", expectCode: cerrors.StatusInvalidArgument},
		{name: "no master key", key: "a.api_key", body: map[string]any{"actor_id": testActor, "secret": fakeSecret, "revision": 0}, noKeyList: true,
			expectStatus: 503, expectReason: "ENCRYPTION_KEY_NOT_CONFIGURED", expectCode: cerrors.StatusUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyList := testKeyList()
			if tt.noKeyList {
				keyList = ""
			}
			f := newFixture(t, keyList)
			if tt.prepare != nil {
				tt.prepare(f)
			}
			before := len(f.store.Histories())

			res := f.do(t, putSecret(tt.key, tt.body))
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong match. expect: %d, got: %d %s", tt.expectStatus, res.StatusCode, res.Data)
			}
			ve := typedError(t, res)
			if ve.Reason != tt.expectReason || ve.Status != tt.expectCode || ve.Domain != string(outline.ServiceNameEmailManager) {
				t.Errorf("Wrong match. got: %+v", ve)
			}
			if strings.Contains(string(res.Data), fakeSecret) {
				t.Errorf("Wrong match. the error response holds the secret: %s", res.Data)
			}
			if len(f.store.Histories()) != before {
				t.Errorf("Wrong match. a failed write left a history row")
			}
		})
	}
}

// A secret with surrounding white space, or a line break anywhere, is refused: a key pasted with a
// trailing newline would be stored and then fail at the provider without a trace.
func Test_putSecret_whitespace(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		ok     bool
	}{
		{"plain", "abc", true},
		{"inner space", "ab c", true},
		{"leading space", " abc", false},
		{"trailing space", "abc ", false},
		{"trailing newline", "abc\n", false},
		{"trailing carriage return", "abc\r", false},
		{"leading newline", "\nabc", false},
		{"leading tab", "\tabc", false},
		{"inner newline", "ab\nc", false},
		{"inner carriage return", "ab\rc", false},
		{"only spaces", "   ", false},
		{"unicode space at the end", "abc\u00a0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, testKeyList())

			body, _ := json.Marshal(map[string]any{"actor_id": testActor, "secret": tt.secret, "revision": 0})
			res := f.do(t, putSecret("a.api_key", string(body)))
			if tt.ok {
				if res.StatusCode != http.StatusOK {
					t.Errorf("Wrong match. expect: 200, got: %d %s", res.StatusCode, res.Data)
				}
				return
			}
			if res.StatusCode != http.StatusBadRequest || typedError(t, res).Reason != "CONFIG_INVALID" {
				t.Errorf("Wrong match. expect: 400 CONFIG_INVALID, got: %d %s", res.StatusCode, res.Data)
			}
			if len(f.store.Histories()) != 0 {
				t.Errorf("Wrong match. a refused secret was stored")
			}
		})
	}
}

func Test_delete(t *testing.T) {
	del := func(key string, rev any) *sock.Request {
		return req(sock.RequestMethodDelete, "/v1/configs/"+key, map[string]any{"actor_id": testActor, "revision": rev})
	}

	t.Run("match", func(t *testing.T) {
		f := newFixture(t, testKeyList())
		f.env["A_API_KEY"] = fakeEnv
		_, _ = f.h.SetSecret(context.Background(), testActor, "a.api_key", "old")

		res := f.do(t, del("a.api_key", 1))
		var got configrpc.Key
		if err := json.Unmarshal(res.Data, &got); err != nil || res.StatusCode != http.StatusOK {
			t.Fatalf("Wrong match. got: %d %s %v", res.StatusCode, res.Data, err)
		}
		// the stored value is gone, and the environment value is used again
		if got.Source != "env" || got.Revision != 0 || len(f.store.Histories()) != 2 {
			t.Errorf("Wrong match. got: %+v, histories: %d", got, len(f.store.Histories()))
		}
	})

	t.Run("stale", func(t *testing.T) {
		f := newFixture(t, testKeyList())
		_, _ = f.h.SetSecret(context.Background(), testActor, "a.api_key", "old")

		res := f.do(t, del("a.api_key", 9))
		if res.StatusCode != 409 || typedError(t, res).Reason != "REVISION_MISMATCH" {
			t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
		}
	})

	t.Run("no row and revision zero is the current state", func(t *testing.T) {
		f := newFixture(t, testKeyList())

		res := f.do(t, del("a.api_key", 0))
		var got configrpc.Key
		if err := json.Unmarshal(res.Data, &got); err != nil || res.StatusCode != http.StatusOK || got.Source != "none" {
			t.Fatalf("Wrong match. got: %d %s %v", res.StatusCode, res.Data, err)
		}
		if len(f.store.Histories()) != 0 {
			t.Errorf("Wrong match. a clear of nothing wrote a history row")
		}
	})

	t.Run("no row and another revision", func(t *testing.T) {
		f := newFixture(t, testKeyList())

		res := f.do(t, del("a.api_key", 3))
		if res.StatusCode != 409 || typedError(t, res).Reason != "REVISION_MISMATCH" {
			t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
		}
	})

	t.Run("unknown key", func(t *testing.T) {
		f := newFixture(t, testKeyList())

		res := f.do(t, del("nothing.here", 0))
		if res.StatusCode != 404 || typedError(t, res).Reason != "CONFIG_KEY_NOT_FOUND" {
			t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
		}
	})

	t.Run("a setting is not cleared by this API", func(t *testing.T) {
		f := newFixture(t, testKeyList())

		res := f.do(t, del("c.limit", 0))
		if res.StatusCode != 400 || typedError(t, res).Reason != "CONFIG_INVALID" {
			t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
		}
	})

	for name, body := range map[string]any{
		"missing revision": map[string]any{"actor_id": testActor},
		"missing actor":    map[string]any{"revision": 0},
		"not json":         `{"revision":`,
		"no body":          nil,
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, testKeyList())

			res := f.do(t, req(sock.RequestMethodDelete, "/v1/configs/a.api_key", body))
			if res.StatusCode != 400 || typedError(t, res).Reason != "CONFIG_INVALID" {
				t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
			}
		})
	}
}

func Test_histories(t *testing.T) {
	f := newFixture(t, testKeyList())
	for i := 0; i < 5; i++ {
		if _, err := f.h.SetSecret(context.Background(), testActor, "a.api_key", fmt.Sprintf("v-%d", i)); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	var ids []uuid.UUID
	token := ""
	for page := 0; page < 10; page++ {
		uri := "/v1/configs/a.api_key/histories?page_size=2"
		if token != "" {
			uri += "&page_token=" + token
		}
		res := f.do(t, req(sock.RequestMethodGet, uri, nil))
		if res.StatusCode != http.StatusOK {
			t.Fatalf("Wrong match. got: %d %s", res.StatusCode, res.Data)
		}

		var list configrpc.HistoryList
		if err := json.Unmarshal(res.Data, &list); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		for _, h := range list.Result {
			if h.Key != "a.api_key" || h.Action != "set" || !h.IsSecret || h.ActorID != testActor || h.Source != "api" || h.NewKeyVersion != 1 {
				t.Errorf("Wrong match. got: %+v", h)
			}
			ids = append(ids, h.ID)
		}
		if list.NextPageToken == "" {
			break
		}
		token = list.NextPageToken
	}

	if len(ids) != 5 {
		t.Errorf("Wrong match. expect: 5 rows, got: %d", len(ids))
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("Wrong match. row %s is returned twice", id)
		}
		seen[id] = true
	}

	// a key without a change is an empty list and not null
	res := f.do(t, req(sock.RequestMethodGet, "/v1/configs/b.api_key/histories", nil))
	if res.StatusCode != http.StatusOK || !strings.Contains(string(res.Data), `"result":[]`) {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}
}

func Test_histories_errors(t *testing.T) {
	f := newFixture(t, testKeyList())

	tests := []struct {
		name         string
		uri          string
		expectStatus int
		expectReason string
	}{
		{"unknown key", "/v1/configs/nothing.here/histories", 404, "CONFIG_KEY_NOT_FOUND"},
		{"bad token", "/v1/configs/a.api_key/histories?page_token=!!!", 400, "CONFIG_INVALID"},
		{"page size not a number", "/v1/configs/a.api_key/histories?page_size=many", 400, "CONFIG_INVALID"},
		{"page size negative", "/v1/configs/a.api_key/histories?page_size=-1", 400, "CONFIG_INVALID"},
		{"broken query", "/v1/configs/a.api_key/histories?page_size=%zz", 400, "CONFIG_INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := f.do(t, req(sock.RequestMethodGet, tt.uri, nil))
			if res.StatusCode != tt.expectStatus || typedError(t, res).Reason != tt.expectReason {
				t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
			}
		})
	}
}

// Every error of the library maps to a fixed status and reason, the wrapped write verification
// error comes first, and the text of the error is never part of the response.
func Test_errorMapping(t *testing.T) {
	const marker = "marker-text-of-the-error"

	tests := []struct {
		name string
		err  error

		expectStatus int
		expectReason string
		expectCode   cerrors.Status
	}{
		{"write verify wrapping a conflict", fmt.Errorf("%w: %w: %s", confighandler.ErrWriteVerify, confighandler.ErrConflict, marker), 500, "CONFIG_WRITE_VERIFY_FAILED", cerrors.StatusInternal},
		{"write verify wrapping undecryptable", fmt.Errorf("%w: %w: %s", confighandler.ErrWriteVerify, confighandler.ErrUndecryptable, marker), 500, "CONFIG_WRITE_VERIFY_FAILED", cerrors.StatusInternal},
		{"stale", fmt.Errorf("%w: %s", confighandler.ErrStale, marker), 409, "REVISION_MISMATCH", cerrors.StatusFailedPrecondition},
		{"conflict", fmt.Errorf("%w: %s", confighandler.ErrConflict, marker), 409, "REVISION_MISMATCH", cerrors.StatusFailedPrecondition},
		{"unknown key", fmt.Errorf("%w: %s", confighandler.ErrUnknownKey, marker), 404, "CONFIG_KEY_NOT_FOUND", cerrors.StatusNotFound},
		{"invalid", fmt.Errorf("%w: %s", confighandler.ErrInvalid, marker), 400, "CONFIG_INVALID", cerrors.StatusInvalidArgument},
		{"wrong kind", fmt.Errorf("%w: %s", confighandler.ErrWrongKind, marker), 400, "CONFIG_INVALID", cerrors.StatusInvalidArgument},
		{"too long", fmt.Errorf("%w: %s", confighandler.ErrTooLong, marker), 400, "CONFIG_INVALID", cerrors.StatusInvalidArgument},
		{"no master key", fmt.Errorf("could not encrypt: %w: %s", confighandler.ErrKeyNotConfigured, marker), 503, "ENCRYPTION_KEY_NOT_CONFIGURED", cerrors.StatusUnavailable},
		{"deadline", fmt.Errorf("%w: %s", context.DeadlineExceeded, marker), 500, "INTERNAL", cerrors.StatusInternal},
		{"canceled", fmt.Errorf("%w: %s", context.Canceled, marker), 500, "INTERNAL", cerrors.StatusInternal},
		{"anything else", errors.New(marker), 500, "INTERNAL", cerrors.StatusInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockHandler := confighandler.NewMockHandler(mc)
			mockHandler.EXPECT().Describe(gomock.Any()).Return(nil, tt.err)
			server := configrpc.NewServer(mockHandler, outline.ServiceNameEmailManager)

			res, err := server.Handle(context.Background(), req(sock.RequestMethodGet, "/v1/configs", nil))
			if err != nil {
				t.Fatalf("Wrong match. expect: a response and no error, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong match. expect: %d, got: %d %s", tt.expectStatus, res.StatusCode, res.Data)
			}
			ve := typedError(t, res)
			if ve.Reason != tt.expectReason || ve.Status != tt.expectCode {
				t.Errorf("Wrong match. got: %+v", ve)
			}
			if strings.Contains(string(res.Data), marker) || ve.Cause != nil {
				t.Errorf("Wrong match. the response carries the text of the error: %s", res.Data)
			}
		})
	}
}

// Handle gives every call a deadline, so that a stalled store can not hold the consumer.
func Test_handle_deadline(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockHandler := confighandler.NewMockHandler(mc)
	mockHandler.EXPECT().Describe(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]*confighandler.Resolved, error) {
		dl, ok := ctx.Deadline()
		if !ok || time.Until(dl) > 16*time.Second || time.Until(dl) < 10*time.Second {
			t.Errorf("Wrong match. expect: a deadline of about 15 seconds, got: %v %v", dl, ok)
		}
		return nil, nil
	})
	mockHandler.EXPECT().SecretWritable().Return(true)
	mockHandler.EXPECT().Registry().Return(&confighandler.Registry{}).AnyTimes()
	server := configrpc.NewServer(mockHandler, outline.ServiceNameEmailManager)

	if res, err := server.Handle(context.Background(), req(sock.RequestMethodGet, "/v1/configs", nil)); err != nil || res.StatusCode != 200 {
		t.Errorf("Wrong match. got: %v %v", res, err)
	}
}

// A request that matches no route is a 404 without a typed body, like the other listeners.
func Test_handle_noRoute(t *testing.T) {
	f := newFixture(t, testKeyList())

	res, err := f.server.Handle(context.Background(), req(sock.RequestMethodPost, "/v1/configs", nil))
	if err != nil || res.StatusCode != http.StatusNotFound {
		t.Errorf("Wrong match. got: %v %v", res, err)
	}
}

// Every log line written while the requests are handled is captured: none holds the secret, in the
// message or in a field. The failing paths are included.
func Test_handle_logsHoldNoSecret(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(prev)

	f := newFixture(t, testKeyList())
	noKey := newFixture(t, "")

	body := func(secret string, rev int) map[string]any {
		return map[string]any{"actor_id": testActor, "secret": secret, "revision": rev}
	}
	requests := []struct {
		f *fixture
		m *sock.Request
	}{
		{f, putSecret("a.api_key", body(fakeSecret, 0))},
		{f, putSecret("a.api_key", body(fakeSecret, 0))},                // stale
		{f, putSecret("a.api_key", body(" "+fakeSecret, 1))},            // white space
		{f, putSecret("nothing.here", body(fakeSecret, 0))},             // unknown key
		{f, putSecret("c.limit", body(fakeSecret, 0))},                  // wrong kind
		{f, putSecret("a.api_key", body(strings.Repeat("a", 40), 1))},   // too long
		{f, putSecret("a.api_key", `{"secret":"`+fakeSecret)},           // broken JSON
		{f, putSecret("a.api_key", `{"secret":"`+fakeSecret+`","x":}`)}, // broken JSON, longer
		{noKey, putSecret("a.api_key", body(fakeSecret, 0))},            // no master key
		{f, req(sock.RequestMethodDelete, "/v1/configs/a.api_key", map[string]any{"actor_id": testActor, "revision": 9})},
		{f, req(sock.RequestMethodGet, "/v1/configs", nil)},
		{f, req(sock.RequestMethodGet, "/v1/configs/a.api_key/histories", nil)},
	}
	for _, r := range requests {
		if _, err := r.f.server.Handle(context.Background(), r.m); err != nil {
			t.Errorf("Wrong match. Handle returns its errors as responses, got: %v", err)
		}
	}

	for _, e := range hook.AllEntries() {
		fields, _ := json.Marshal(e.Data)
		line := e.Message + " " + string(fields)
		if strings.Contains(line, fakeSecret) || strings.Contains(strings.ToLower(line), "test-secret") {
			t.Errorf("Wrong match. a log line holds the secret: %s", line)
		}
	}

	// the same for an error that comes from the store: the text of that error is not a secret, but a
	// request log would hold the body, so the body must not be part of any entry
	hook.Reset()
	f.store.ListHistoriesErr = errors.New("database is down")
	if res := f.do(t, req(sock.RequestMethodGet, "/v1/configs/a.api_key/histories", nil)); res.StatusCode != 500 {
		t.Errorf("Wrong match. expect: 500, got: %d", res.StatusCode)
	}
	for _, e := range hook.AllEntries() {
		if fields, _ := json.Marshal(e.Data); strings.Contains(e.Message+string(fields), fakeSecret) {
			t.Errorf("Wrong match. a log line holds the secret")
		}
	}
}

// The field names of the models are the contract with the public API: a drift is caught here.
func Test_modelFieldNames(t *testing.T) {
	names := func(v any) []string {
		var res []string
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
			res = append(res, tag)
		}
		sort.Strings(res)
		return res
	}

	tests := []struct {
		name   string
		v      any
		expect []string
	}{
		{"service", configrpc.Service{}, []string{"keys", "secret_writable", "service"}},
		{"key", configrpc.Key{}, []string{"description", "env_name", "env_shadowed", "has_secret", "key", "key_version", "kind", "revision", "source", "tm_update", "type", "updated_by"}},
		{"history", configrpc.History{}, []string{"action", "actor_id", "id", "is_secret", "key", "new_key_version", "old_key_version", "source", "tm_create"}},
		{"history list", configrpc.HistoryList{}, []string{"next_page_token", "result"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := names(tt.v); !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
