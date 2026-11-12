package listenhandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/configkeys"
	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/confighandler"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/confighandler/storetest"
	"monorepo/bin-common-handler/pkg/secretcrypto"
)

const (
	configTestSecret = "test-secret-not-real"
	configTestActor  = "11111111-1111-4111-8111-111111111111"
	configTestKey    = "llm.gemini.api_key"
	configTestAIID   = "22222222-2222-4222-8222-222222222222"
	configTestType   = "/v1/configs"
)

func configTestKeyList() string {
	k := make([]byte, 32)
	for i := range k {
		k[i] = 7 + byte(i)
	}

	return "1:" + base64.StdEncoding.EncodeToString(k)
}

// panicSetSecret is a config handler whose write of a secret panics with the plaintext secret that
// it receives: the worst case of what a panic value can carry.
type panicSetSecret struct {
	confighandler.Handler
}

func (p *panicSetSecret) SetSecret(_ context.Context, _ uuid.UUID, _, secret string, _ ...confighandler.WriteOption) (*confighandler.Resolved, error) {
	panic("boom with " + secret)
}

// failingStore is a store that fails every write.
type failingStore struct {
	*storetest.Memory
}

func (f *failingStore) Insert(context.Context, *confighandler.Row, *confighandler.History) error {
	return errors.New("the database is unavailable")
}

func newConfigListenHandler(t *testing.T, keyList string, wrap func(confighandler.Handler) confighandler.Handler, store confighandler.Store) (*listenHandler, *storetest.Memory, *aihandler.MockAIHandler) {
	t.Helper()

	reg, err := configkeys.NewRegistry()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	crypto, err := secretcrypto.New(keyList)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	mem := storetest.NewMemory()
	if store == nil {
		store = mem
	}
	h := confighandler.NewHandler(reg, store, crypto, func(string) (string, bool) { return "", false })
	if wrap != nil {
		h = wrap(h)
	}

	// the mock is strict: a call that a test does not expect fails the test
	mc := gomock.NewController(t)
	mockAI := aihandler.NewMockAIHandler(mc)

	return &listenHandler{aiHandler: mockAI, configServer: configrpc.NewServer(h, commonoutline.ServiceNameAIManager)}, mem, mockAI
}

func configPutSecret(uri, secret string, revision int) *sock.Request {
	body, _ := json.Marshal(map[string]any{"actor_id": configTestActor, "secret": secret, "revision": revision})

	return &sock.Request{URI: uri, Method: sock.RequestMethodPut, DataType: "application/json", Data: body}
}

func Test_processRequest_configs(t *testing.T) {
	h, store, _ := newConfigListenHandler(t, configTestKeyList(), nil, nil)

	// the list of the keys
	res, err := h.processRequest(&sock.Request{URI: "/v1/configs", Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("Wrong match. got: %v %v", res, err)
	}
	var service configrpc.Service
	if errUnmarshal := json.Unmarshal(res.Data, &service); errUnmarshal != nil || len(service.Keys) != 2 || !service.SecretWritable {
		t.Fatalf("Wrong match. got: %s %v", res.Data, errUnmarshal)
	}
	names := []string{}
	for _, k := range service.Keys {
		names = append(names, k.Key)
		if k.Description == "" || k.EnvName != "" || k.Kind != "secret" {
			t.Errorf("Wrong match. got: %+v", k)
		}
	}
	if strings.Join(names, ",") != "llm.openai_compat.api_key,llm.gemini.api_key" {
		t.Errorf("Wrong match. got: %v", names)
	}

	// write, read the history, delete
	res, _ = h.processRequest(configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0))
	if res.StatusCode != 200 || strings.Contains(string(res.Data), configTestSecret) {
		t.Fatalf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}
	if len(store.Histories()) != 1 || store.Histories()[0].Source != "api" {
		t.Errorf("Wrong match. got: %+v", store.Histories())
	}

	res, _ = h.processRequest(&sock.Request{URI: "/v1/configs/" + configTestKey + "/histories", Method: sock.RequestMethodGet})
	if res.StatusCode != 200 || !strings.Contains(string(res.Data), `"is_secret":true`) {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}

	res, _ = h.processRequest(&sock.Request{URI: "/v1/configs/" + configTestKey, Method: sock.RequestMethodDelete, Data: json.RawMessage(`{"actor_id":"` + configTestActor + `","revision":1}`)})
	if res.StatusCode != 200 {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}

	// the errors are the typed errors of the shared library, with the domain of this service
	tests := []struct {
		name         string
		m            *sock.Request
		expectStatus int
		expectReason string
	}{
		{"stale revision", configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 7), 409, "REVISION_MISMATCH"},
		{"unknown key", configPutSecret("/v1/configs/nothing/secret", configTestSecret, 0), 404, "CONFIG_KEY_NOT_FOUND"},
		{"white space", configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret+"\n", 1), 400, "CONFIG_INVALID"},
		{"too long", configPutSecret("/v1/configs/"+configTestKey+"/secret", strings.Repeat("a", 513), 1), 400, "CONFIG_INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _ := h.processRequest(tt.m)
			ve := cerrors.FromResponse(res)
			if res.StatusCode != tt.expectStatus || ve == nil || ve.Reason != tt.expectReason || ve.Domain != string(commonoutline.ServiceNameAIManager) {
				t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
			}
		})
	}

	// without a master key the secret can not be stored
	noKey, _, _ := newConfigListenHandler(t, "", nil, nil)
	res, _ = noKey.processRequest(configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0))
	if ve := cerrors.FromResponse(res); res.StatusCode != 503 || ve == nil || ve.Reason != "ENCRYPTION_KEY_NOT_CONFIGURED" {
		t.Errorf("Wrong match. got: %d %s", res.StatusCode, res.Data)
	}
}

// A listener without a config server answers the config uris with 404, and a request of another
// route is still served by its own handler.
func Test_processRequest_configsWithoutServer(t *testing.T) {
	mc := gomock.NewController(t)
	mockAI := aihandler.NewMockAIHandler(mc)
	h := &listenHandler{aiHandler: mockAI}

	for _, m := range []*sock.Request{
		{URI: "/v1/configs", Method: sock.RequestMethodGet},
		configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0),
	} {
		if res, err := h.processRequest(m); err != nil || res.StatusCode != 404 {
			t.Errorf("Wrong match. got: %v %v", res, err)
		}
	}

	// an existing route is not touched by the config branch
	mockAI.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(configTestAIID)).Return(nil, dbhandler.ErrNotFound)
	res, err := h.processRequest(&sock.Request{URI: "/v1/ais/" + configTestAIID, Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 404 {
		t.Errorf("Wrong match. got: %v %v", res, err)
	}
	if res, _ := h.processRequest(&sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet}); res.StatusCode != 404 {
		t.Errorf("Wrong match. got: %v", res)
	}
}

// An existing route is served as before when the config server is mounted.
func Test_processRequest_otherRoutesUnchangedWithServer(t *testing.T) {
	h, _, mockAI := newConfigListenHandler(t, configTestKeyList(), nil, nil)

	mockAI.EXPECT().Get(gomock.Any(), uuid.FromStringOrNil(configTestAIID)).Return(nil, dbhandler.ErrNotFound)
	res, err := h.processRequest(&sock.Request{URI: "/v1/ais/" + configTestAIID, Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 404 {
		t.Errorf("Wrong match. got: %v %v", res, err)
	}
}

// metricSampleCount returns the sample count of the latency series of the given labels, and
// whether the series exists. It also fails the test when a label value of any series holds text.
func metricSampleCount(t *testing.T, typ, method string, forbidden ...string) (uint64, bool) {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}

	var count uint64
	found := false
	for _, f := range families {
		for _, m := range f.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
				for _, text := range forbidden {
					if strings.Contains(l.GetValue(), text) {
						t.Errorf("Wrong match. a metric label holds the secret. family: %s", f.GetName())
					}
				}
			}
			if f.GetName() == "ai_manager_receive_request_process_time" && labels["type"] == typ && labels["method"] == method {
				found = true
				count = m.GetHistogram().GetSampleCount()
			}
		}
	}

	return count, found
}

// No log line, response or metric label written while a config request is processed holds the
// secret: the routed requests, the failing ones, the unrouted and mistyped ones, and a panic.
func Test_processRequest_configsHoldNoSecret(t *testing.T) {
	prev := logrus.GetLevel()
	defer logrus.SetLevel(prev)

	panicking := func(h confighandler.Handler) confighandler.Handler { return &panicSetSecret{Handler: h} }

	tests := []struct {
		name         string
		keyList      string
		wrap         func(confighandler.Handler) confighandler.Handler
		failStore    bool
		m            *sock.Request
		expectStatus int
		expectLog    bool
	}{
		{name: "routed", keyList: configTestKeyList(), m: configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0), expectStatus: 200},
		{name: "stale", keyList: configTestKeyList(), m: configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 9), expectStatus: 409},
		{name: "unknown key", keyList: configTestKeyList(), m: configPutSecret("/v1/configs/nothing/secret", configTestSecret, 0), expectStatus: 404},
		{name: "white space", keyList: configTestKeyList(), m: configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret+"\n", 0), expectStatus: 400},
		{name: "broken body", keyList: configTestKeyList(), m: &sock.Request{URI: "/v1/configs/" + configTestKey + "/secret", Method: sock.RequestMethodPut, Data: json.RawMessage(`{"secret":"` + configTestSecret)}, expectStatus: 400},
		{name: "no master key", keyList: "", m: configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0), expectStatus: 503},
		{name: "store error", keyList: configTestKeyList(), failStore: true, m: configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0), expectStatus: 500, expectLog: true},
		{name: "typo in the resource", keyList: configTestKeyList(), m: configPutSecret("/v1/config/"+configTestKey+"/secret", configTestSecret, 0), expectStatus: 404},
		{name: "typo in the key", keyList: configTestKeyList(), m: configPutSecret("/v1/configs/Llm.Gemini.Api_Key/secret", configTestSecret, 0), expectStatus: 404},
		{name: "upper case", keyList: configTestKeyList(), m: configPutSecret("/V1/CONFIGS/"+strings.ToUpper(configTestKey)+"/SECRET", configTestSecret, 0), expectStatus: 404},
		{name: "key of 129 characters", keyList: configTestKeyList(), m: configPutSecret("/v1/configs/"+strings.Repeat("a", 129)+"/secret", configTestSecret, 0), expectStatus: 404},
		{name: "wrong method", keyList: configTestKeyList(), m: &sock.Request{URI: "/v1/configs/" + configTestKey + "/secret", Method: sock.RequestMethodPost, Data: json.RawMessage(`{"secret":"` + configTestSecret + `"}`)}, expectStatus: 404},
		{name: "another resource", keyList: configTestKeyList(), m: configPutSecret("/v1/other/secret", configTestSecret, 0), expectStatus: 404},
		{name: "panic", keyList: configTestKeyList(), wrap: panicking, m: configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0), expectStatus: 500, expectLog: true},
		// The key holds a slash: the key pattern refuses it, but the pattern of the ai route
		// regV1AIsID is not anchored at the start and would take the uri. The AI handler is a
		// strict mock, so a call of it fails the test.
		{name: "key that overlaps an ai route", keyList: configTestKeyList(), m: &sock.Request{URI: "/v1/configs/" + configTestAIID + "/v1/ais/" + configTestAIID, Method: sock.RequestMethodPut, DataType: "application/json", Data: json.RawMessage(`{"secret":"` + configTestSecret + `"}`)}, expectStatus: 404},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := logtest.New(t)
			logrus.SetLevel(logrus.TraceLevel)

			var store confighandler.Store
			if tt.failStore {
				store = &failingStore{Memory: storetest.NewMemory()}
			}
			h, _, _ := newConfigListenHandler(t, tt.keyList, tt.wrap, store)

			before, _ := metricSampleCount(t, configTestType, string(tt.m.Method))

			res, err := h.processRequest(tt.m)
			if err != nil || res == nil {
				t.Fatalf("Wrong match. got: %v %v", res, err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d %s", tt.expectStatus, res.StatusCode, res.Data)
			}
			if strings.Contains(string(res.Data), configTestSecret) {
				t.Errorf("Wrong match. the response holds the secret")
			}
			if strings.Contains(logs.Dump(), configTestSecret) {
				t.Errorf("Wrong match. a log line holds the secret: %s", logs.Dump())
			}
			// a failure that is logged must show a line (the check above is not vacuous)
			if tt.expectLog && logs.Len() == 0 {
				t.Errorf("Wrong match. expect: a log line, got: none")
			}

			// the secret of a panic is not in the response either: the answer is a bare 500
			if tt.name == "panic" && len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: a bare 500, got: %s", res.Data)
			}

			// the series exists and counts the request, also after a panic
			after, found := metricSampleCount(t, configTestType, string(tt.m.Method), configTestSecret)
			if !found || after != before+1 {
				t.Errorf("Wrong match. expect the series %q to count one request, got before: %d, after: %d, found: %v", configTestType, before, after, found)
			}
		})
	}
}

// The request that the handler works on keeps its body: only the log copy is made safe, and the
// config server is the only one that reads it.
func Test_processRequest_configsKeepsTheBody(t *testing.T) {
	h, _, _ := newConfigListenHandler(t, configTestKeyList(), nil, nil)

	m := configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0)
	if res, _ := h.processRequest(m); res.StatusCode != 200 || !strings.Contains(string(m.Data), configTestSecret) {
		t.Errorf("Wrong match. got: %d, body: %s", res.StatusCode, m.Data)
	}
}

// A panic does not stop the next request.
func Test_processConfig_panicIsRecovered(t *testing.T) {
	h, _, _ := newConfigListenHandler(t, configTestKeyList(), func(h confighandler.Handler) confighandler.Handler { return &panicSetSecret{Handler: h} }, nil)

	res, err := h.processConfig(configPutSecret("/v1/configs/"+configTestKey+"/secret", configTestSecret, 0))
	if err != nil || res.StatusCode != 500 {
		t.Fatalf("Wrong match. got: %v %v", res, err)
	}

	res, err = h.processConfig(&sock.Request{URI: "/v1/configs", Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 200 {
		t.Errorf("Wrong match. got: %v %v", res, err)
	}
}

// The constructor keeps the config server, and accepts a nil one.
func Test_NewListenHandler_configServer(t *testing.T) {
	srv := configrpc.NewServer(nil, commonoutline.ServiceNameAIManager)

	h, ok := NewListenHandler(nil, "queue", "delay", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, srv).(*listenHandler)
	if !ok || h.configServer != srv || h.queueListen != "queue" {
		t.Errorf("Wrong match. got: %+v", h)
	}

	h, ok = NewListenHandler(nil, "queue", "delay", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).(*listenHandler)
	if !ok || h.configServer != nil {
		t.Errorf("Wrong match. got: %+v", h)
	}
}
