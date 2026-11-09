package listenhandler

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
	"monorepo/bin-common-handler/pkg/secretcrypto"

	"monorepo/bin-email-manager/internal/configkeys"
)

const (
	configTestSecret = "test-secret-not-real"
	configTestActor  = "11111111-1111-4111-8111-111111111111"
)

func newConfigTestHandler(t *testing.T, keyList string) (*listenHandler, *storetest.Memory) {
	t.Helper()

	reg, err := configkeys.NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	crypto, err := secretcrypto.New(keyList)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	store := storetest.NewMemory()
	h := confighandler.NewHandler(reg, store, crypto, func(string) (string, bool) { return "", false })

	return &listenHandler{configServer: configrpc.NewServer(h, commonoutline.ServiceNameEmailManager)}, store
}

func configTestKeyList() string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = 7 + byte(i)
	}

	return "1:" + base64.StdEncoding.EncodeToString(k)
}

func putSecretRequest(uri, secret string, revision int) *sock.Request {
	body, _ := json.Marshal(map[string]any{"actor_id": configTestActor, "secret": secret, "revision": revision})

	return &sock.Request{URI: uri, Method: sock.RequestMethodPut, DataType: "application/json", Data: body}
}

func Test_processRequest_configs(t *testing.T) {
	h, store := newConfigTestHandler(t, configTestKeyList())

	// the list of the keys
	res, err := h.processRequest(&sock.Request{URI: "/v1/configs", Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("Wrong match. got: %v %v", res, err)
	}
	var service configrpc.Service
	if err := json.Unmarshal(res.Data, &service); err != nil || len(service.Keys) != 3 || !service.SecretWritable {
		t.Fatalf("Wrong match. got: %s %v", res.Data, err)
	}
	for _, k := range service.Keys {
		if k.Description == "" || k.EnvName == "" || k.Kind != "secret" {
			t.Errorf("Wrong match. got: %+v", k)
		}
	}

	// write, read, clear
	res, _ = h.processRequest(putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret, 0))
	if res.StatusCode != 200 || strings.Contains(string(res.Data), configTestSecret) {
		t.Fatalf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}
	if len(store.Histories()) != 1 || store.Histories()[0].Source != "api" {
		t.Errorf("Wrong match. got: %+v", store.Histories())
	}

	res, _ = h.processRequest(&sock.Request{URI: "/v1/configs/mailgun.api_key/histories", Method: sock.RequestMethodGet})
	if res.StatusCode != 200 || !strings.Contains(string(res.Data), `"is_secret":true`) {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}

	res, _ = h.processRequest(&sock.Request{URI: "/v1/configs/mailgun.api_key", Method: sock.RequestMethodDelete, Data: json.RawMessage(`{"actor_id":"` + configTestActor + `","revision":1}`)})
	if res.StatusCode != 200 {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}

	// a stale write is a typed 409
	res, _ = h.processRequest(putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret, 7))
	if ve := cerrors.FromResponse(res); res.StatusCode != 409 || ve == nil || ve.Reason != "REVISION_MISMATCH" {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}
}

// A listener without a config server keeps its behavior: the config URIs are not found.
func Test_processRequest_configsWithoutServer(t *testing.T) {
	h := &listenHandler{}

	res, err := h.processRequest(&sock.Request{URI: "/v1/configs", Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 404 {
		t.Errorf("Wrong match. got: %v %v", res, err)
	}
}

// No log line written while a request is processed holds the secret: the routed requests, the failing
// ones (the error logs of the listener repeat the request), the unrouted ones (the default branch
// logs the body) and the mistyped or upper case URIs.
func Test_processRequest_logsHoldNoSecret(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetLevel(prev)

	h, _ := newConfigTestHandler(t, configTestKeyList())
	noKey, _ := newConfigTestHandler(t, "")

	requests := []struct {
		name string
		h    *listenHandler
		m    *sock.Request
	}{
		{"routed", h, putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret, 0)},
		{"stale", h, putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret, 9)},
		{"unknown key", h, putSecretRequest("/v1/configs/nothing/secret", configTestSecret, 0)},
		{"white space", h, putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret+"\n", 1)},
		{"broken body", h, &sock.Request{URI: "/v1/configs/mailgun.api_key/secret", Method: sock.RequestMethodPut, Data: json.RawMessage(`{"secret":"` + configTestSecret)}},
		{"no master key", noKey, putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret, 0)},
		{"typo in the resource", h, putSecretRequest("/v1/config/mailgun.api_key/secret", configTestSecret, 0)},
		{"typo in the key", h, putSecretRequest("/v1/configs/Mailgun.Api_Key/secret", configTestSecret, 0)},
		{"upper case", h, putSecretRequest("/V1/CONFIGS/MAILGUN.API_KEY/SECRET", configTestSecret, 0)},
		{"wrong method", h, &sock.Request{URI: "/v1/configs/mailgun.api_key/secret", Method: sock.RequestMethodPost, Data: json.RawMessage(`{"secret":"` + configTestSecret + `"}`)}},
		{"another service name", h, putSecretRequest("/v1/other/secret", configTestSecret, 0)},
	}
	for _, tt := range requests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()

			res, err := tt.h.processRequest(tt.m)
			if err != nil || res == nil {
				t.Fatalf("Wrong match. got: %v %v", res, err)
			}

			for _, e := range hook.AllEntries() {
				// the fields hold pointers (the request): the content is marshaled, not the address
				fields, _ := json.Marshal(e.Data)
				line := e.Message + " " + string(fields)
				if strings.Contains(line, configTestSecret) {
					t.Errorf("Wrong match. a log line holds the secret: %s", line)
				}
			}
			if strings.Contains(string(res.Data), configTestSecret) {
				t.Errorf("Wrong match. the response holds the secret")
			}
		})
	}

	// the request that the handler works on keeps its body: only the log copy is replaced
	fresh, _ := newConfigTestHandler(t, configTestKeyList())
	m := putSecretRequest("/v1/configs/mailgun.api_key/secret", configTestSecret, 0)
	if res, _ := fresh.processRequest(m); res.StatusCode != 200 || !strings.Contains(string(m.Data), configTestSecret) {
		t.Errorf("Wrong match. got: %d, body: %s", res.StatusCode, m.Data)
	}
}

// The other routes still log the request as before (the body of an email request stays visible).
func Test_processRequest_otherRequestsAreLoggedAsBefore(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()

	h := &listenHandler{}
	m := &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet, Data: json.RawMessage(`{"x":"visible-body"}`)}
	if res, _ := h.processRequest(m); res.StatusCode != 404 {
		t.Fatalf("Wrong match. expect: 404, got: %d", res.StatusCode)
	}

	found := false
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "visible-body") {
			found = true
		}
	}
	if !found {
		t.Errorf("Wrong match. the default branch no longer logs the body of an unrelated request")
	}
}
