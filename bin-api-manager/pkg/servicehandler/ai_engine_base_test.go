package servicehandler

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

const (
	engineBaseTestURL = "https://sentinel-url-must-not-leak.example.com/v1"
	engineBaseTestKey = "sentinel-base-key-must-not-leak-7e3a"
	engineBaseTestCID = "5f621078-8e5f-11ee-97b2-cfe7337b701c"
	engineBaseTestAID = "ea4b81a9-ffab-4c20-8a77-c9e4d80df548"
)

func engineBaseAgent() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil(engineBaseTestCID),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
}

// engineBaseSock is a socket stub whose behavior is swapped per test.
// NewRequestHandler registers prometheus collectors and may be called once per
// process, so the real requesthandler is built once on top of this stub.
type engineBaseSock struct {
	sockhandler.SockHandler
}

var (
	engineBaseSockFn  func(req *sock.Request) (*sock.Response, error)
	engineBaseReqOnce sync.Once
	engineBaseReq     requesthandler.RequestHandler
)

func (engineBaseSock) RequestPublish(_ context.Context, _ string, req *sock.Request) (*sock.Response, error) {
	return engineBaseSockFn(req)
}

// engineBaseHandler builds a servicehandler on top of the REAL requesthandler
// with a stubbed socket; the returned pointer receives the data of the last
// POST or PUT sent to ai-manager.
func engineBaseHandler(t *testing.T) (*serviceHandler, *[]byte) {
	t.Helper()

	engineBaseReqOnce.Do(func() {
		engineBaseReq = requesthandler.NewRequestHandler(engineBaseSock{}, commonoutline.ServiceNameAPIManager)
	})

	captured := new([]byte)
	engineBaseSockFn = func(req *sock.Request) (*sock.Response, error) {
		if req.Method == sock.RequestMethodGet {
			return &sock.Response{StatusCode: 200, DataType: "application/json",
				Data: []byte(`{"id":"` + engineBaseTestAID + `","customer_id":"` + engineBaseTestCID + `"}`)}, nil
		}
		*captured = req.Data
		return &sock.Response{StatusCode: 200, DataType: "application/json",
			Data: []byte(`{"id":"` + engineBaseTestAID + `","customer_id":"` + engineBaseTestCID + `","engine_base_url":"` + engineBaseTestURL + `"}`)}, nil
	}

	return &serviceHandler{reqHandler: engineBaseReq}, captured
}

// Test_AIUpdate_EngineBaseKeyStatesRoundTrip sends the four states of
// engine_base_key from the generated request type through servicehandler and
// the real requesthandler and unmarshals the wire bytes into the ai-manager DTO.
func Test_AIUpdate_EngineBaseKeyStatesRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		body string

		expectKeyNil bool
		expectKey    string
		expectInWire bool
	}{
		{"omitted", `{}`, true, "", false},
		{"null", `{"engine_base_key":null}`, true, "", false},
		{"empty string", `{"engine_base_key":""}`, false, "", true},
		{"value", `{"engine_base_key":"` + engineBaseTestKey + `"}`, false, engineBaseTestKey, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, captured := engineBaseHandler(t)

			var req openapi_server.PutAisIdJSONBody
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatalf("could not unmarshal: %v", err)
			}

			_, err := h.AIUpdate(context.Background(), engineBaseAgent(), uuid.FromStringOrNil(engineBaseTestAID),
				"", "", "", amai.EngineModel("customendpoint.m1"), nil, stringPtr(""), engineBaseTestURL, req.EngineBaseKey,
				uuid.Nil, "", "", "", "", "", nil, nil, false)
			if err != nil {
				t.Fatalf("AIUpdate failed: %v", err)
			}

			if got := strings.Contains(string(*captured), `"engine_base_key"`); got != tt.expectInWire {
				t.Errorf("Wrong wire presence of engine_base_key. expect: %v, got: %v, wire: %s", tt.expectInWire, got, *captured)
			}

			var dto engineBaseWire
			if err := json.Unmarshal(*captured, &dto); err != nil {
				t.Fatalf("could not unmarshal the DTO: %v", err)
			}
			if dto.EngineBaseURL != engineBaseTestURL {
				t.Errorf("Wrong url. got: %q", dto.EngineBaseURL)
			}
			if (dto.EngineBaseKey == nil) != tt.expectKeyNil {
				t.Fatalf("Wrong nil state. expect nil: %v, got: %v", tt.expectKeyNil, dto.EngineBaseKey)
			}
			if dto.EngineBaseKey != nil && *dto.EngineBaseKey != tt.expectKey {
				t.Errorf("Wrong key. expect: %q, got: %q", tt.expectKey, *dto.EngineBaseKey)
			}
		})
	}
}

func Test_AICreate_EngineBaseKeyStatesRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		body string

		expectKeyNil bool
		expectKey    string
	}{
		{"omitted", `{}`, true, ""},
		{"null", `{"engine_base_key":null}`, true, ""},
		{"empty string", `{"engine_base_key":""}`, false, ""},
		{"value", `{"engine_base_key":"` + engineBaseTestKey + `"}`, false, engineBaseTestKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, captured := engineBaseHandler(t)

			var req openapi_server.PostAisJSONBody
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatalf("could not unmarshal: %v", err)
			}

			_, err := h.AICreate(context.Background(), engineBaseAgent(),
				"n", "d", "", amai.EngineModel("customendpoint.m1"), nil, "", engineBaseTestURL, req.EngineBaseKey,
				uuid.Nil, "", "", "", "", "", nil, nil, false)
			if err != nil {
				t.Fatalf("AICreate failed: %v", err)
			}

			var dto engineBaseWire
			if err := json.Unmarshal(*captured, &dto); err != nil {
				t.Fatalf("could not unmarshal the DTO: %v", err)
			}
			if dto.EngineBaseURL != engineBaseTestURL {
				t.Errorf("Wrong url. got: %q", dto.EngineBaseURL)
			}
			if (dto.EngineBaseKey == nil) != tt.expectKeyNil {
				t.Fatalf("Wrong nil state. expect nil: %v, got: %v", tt.expectKeyNil, dto.EngineBaseKey)
			}
			if dto.EngineBaseKey != nil && *dto.EngineBaseKey != tt.expectKey {
				t.Errorf("Wrong key. expect: %q, got: %q", tt.expectKey, *dto.EngineBaseKey)
			}
		})
	}
}

// Test_AIWrite_EngineBaseNotLogged runs create and update with a distinctive
// URL, key and model id and requires that no log entry carries any of them.
func Test_AIWrite_EngineBaseNotLogged(t *testing.T) {
	hook := newSecretLogHook(t)
	key := engineBaseTestKey

	h, _ := engineBaseHandler(t)
	ctx := context.Background()
	model := amai.EngineModel("customendpoint.sentinel-model-id-9c1d")

	if _, err := h.AICreate(ctx, engineBaseAgent(), "n", "d", "", model, nil, "", engineBaseTestURL, &key,
		uuid.Nil, "", "", "", "", "", nil, nil, false); err != nil {
		t.Fatalf("AICreate failed: %v", err)
	}
	if _, err := h.AIUpdate(ctx, engineBaseAgent(), uuid.FromStringOrNil(engineBaseTestAID), "n", "d", "", model, nil, stringPtr(""), engineBaseTestURL, &key,
		uuid.Nil, "", "", "", "", "", nil, nil, false); err != nil {
		t.Fatalf("AIUpdate failed: %v", err)
	}

	assertNoSecretInLogs(t, hook, "Created a new ai", engineBaseTestURL, "sentinel-url-must-not-leak", engineBaseTestKey, "sentinel-model-id-9c1d")
	assertNoSecretInLogs(t, hook, "Updated ai info", engineBaseTestURL, engineBaseTestKey, "sentinel-model-id-9c1d")
}

func Test_loggableEngineModel_customendpoint(t *testing.T) {
	tests := []struct {
		model     amai.EngineModel
		expectRes string
	}{
		{"customendpoint.llama-3", "customendpoint.<redacted>"},
		{"CustomEndpoint.X", "customendpoint.<redacted>"},
		{" customendpoint.x", "customendpoint.<redacted>"},
		{"customendpoint.", "customendpoint.<redacted>"},
		{"local.llama", "local.llama"},
		{"openai.gpt-5", "openai.gpt-5"},
	}
	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			if res := loggableEngineModel(tt.model); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// engineBaseWire is the part of the ai-manager request body these tests read (the transport DTO
// package of ai-manager is not imported here).
type engineBaseWire struct {
	EngineBaseURL string  `json:"engine_base_url"`
	EngineBaseKey *string `json:"engine_base_key"`
}
