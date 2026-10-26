package listenhandler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/aihandler"
)

func Test_processV1AIsPostPut_EndpointFields(t *testing.T) {
	const aiID = "de99e522-a770-11ed-a0ab-5b39ee2db203"

	tests := []struct {
		name       string
		body       string
		expectURL  string
		expectKey  *string // nil = omitted
		expectNull bool
	}{
		{"omitted", `{"engine_model":"customendpoint.m"}`, "", nil, true},
		{"null", `{"engine_model":"customendpoint.m","engine_base_key":null}`, "", nil, true},
		{"empty_string", `{"engine_model":"customendpoint.m","engine_base_url":"https://llm.example.com","engine_base_key":""}`, "https://llm.example.com", strPtr(""), false},
		{"value", `{"engine_model":"customendpoint.m","engine_base_url":"https://llm.example.com","engine_base_key":"sk-abc"}`, "https://llm.example.com", strPtr("sk-abc"), false},
	}

	for _, method := range []sock.RequestMethod{sock.RequestMethodPost, sock.RequestMethodPut} {
		for _, tt := range tests {
			t.Run(string(method)+"_"+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockAI := aihandler.NewMockAIHandler(mc)
				h := &listenHandler{sockHandler: sockhandler.NewMockSockHandler(mc), aiHandler: mockAI}

				var gotURL string
				var gotKey *string
				capture := func(url string, key *string) {
					gotURL, gotKey = url, key
				}
				response := &ai.AI{Identity: identity.Identity{ID: uuid.FromStringOrNil(aiID)}, EngineBaseKeyEnc: []byte("cipher")}

				req := &sock.Request{URI: "/v1/ais", Method: method, DataType: "application/json"}
				if method == sock.RequestMethodPost {
					req.Data = []byte(strings.Replace(tt.body, "{", `{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847",`, 1))
					mockAI.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _ uuid.UUID, _, _ string, _ ai.Type, _ ai.EngineModel, _ map[string]any, _ string, url string, key *string,
							_ uuid.UUID, _ string, _ ai.TTSType, _ string, _ ai.STTType, _ string, _ []tool.ToolName, _ *ai.VADConfig, _, _ bool) (*ai.AI, error) {
							capture(url, key)
							return response, nil
						})
				} else {
					req.URI = "/v1/ais/" + aiID
					req.Data = []byte(tt.body)
					mockAI.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, _ uuid.UUID, _, _ string, _ ai.Type, _ ai.EngineModel, _ map[string]any, _ *string, url string, key *string,
							_ uuid.UUID, _ string, _ ai.TTSType, _ string, _ ai.STTType, _ string, _ []tool.ToolName, _ *ai.VADConfig, _, _ bool) (*ai.AI, error) {
							capture(url, key)
							return response, nil
						})
				}
				// no FillEngineBaseKey expectation: create and update responses must never carry the plaintext

				res, err := h.processRequest(req)
				if err != nil || res.StatusCode != 200 {
					t.Fatalf("unexpected result. err: %v, res: %+v", err, res)
				}

				if gotURL != tt.expectURL {
					t.Errorf("url: expect %q, got %q", tt.expectURL, gotURL)
				}
				if tt.expectNull != (gotKey == nil) || (gotKey != nil && *gotKey != *tt.expectKey) {
					t.Errorf("key pointer state differs. expect: %v, got: %v", tt.expectKey, gotKey)
				}
				if strings.Contains(string(res.Data), "engine_base_key\"") {
					t.Errorf("the response must not carry a plaintext key: %s", res.Data)
				}
			})
		}
	}
}

func Test_processV1AIsIDGet_FillsThePlaintextOnACopyOnly(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAI := aihandler.NewMockAIHandler(mc)
	h := &listenHandler{sockHandler: sockhandler.NewMockSockHandler(mc), aiHandler: mockAI}

	id := uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203")
	stored := &ai.AI{Identity: identity.Identity{ID: id}, EngineBaseURL: "https://llm.example.com", EngineBaseKeyEnc: []byte("cipher")}
	filled := *stored
	filled.EngineBaseKey = "plain-from-fill"

	mockAI.EXPECT().Get(gomock.Any(), id).Return(stored, nil)
	mockAI.EXPECT().FillEngineBaseKey(stored).Return(&filled)

	res, err := h.processRequest(&sock.Request{URI: "/v1/ais/" + id.String(), Method: sock.RequestMethodGet})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("unexpected result. err: %v", err)
	}

	var got ai.AI
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("could not parse the response: %v", err)
	}
	if string(got.EngineBaseKey) != "plain-from-fill" || string(got.EngineBaseKeyEnc) != "cipher" {
		t.Errorf("the get response must carry the plaintext and the ciphertext")
	}
	if stored.EngineBaseKey != "" {
		t.Errorf("the stored value must stay untouched")
	}
}

func Test_processV1AIsGet_ListDoesNotDecrypt(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAI := aihandler.NewMockAIHandler(mc)
	h := &listenHandler{sockHandler: sockhandler.NewMockSockHandler(mc), aiHandler: mockAI}

	mockAI.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*ai.AI{{Identity: identity.Identity{ID: uuid.Must(uuid.NewV4())}, EngineBaseKeyEnc: []byte("cipher")}}, nil)

	res, err := h.processRequest(&sock.Request{URI: "/v1/ais?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{}`)})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("unexpected result. err: %v", err)
	}
	if strings.Contains(string(res.Data), `"engine_base_key":`) {
		t.Errorf("the list must not carry a plaintext key: %s", res.Data)
	}
}
