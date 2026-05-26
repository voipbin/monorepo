package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amtool "monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAis(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAI *amai.WebhookMessage

		expectedName        string
		expectedDetail      string
		expectedEngineModel amai.EngineModel
		expectedParameter   map[string]any
		expectedEngineKey   string
		expectedInitPrompt  string
		expectedTTSType     amai.TTSType
		expectedTTSVoiceID  string
		expectedSTTType     amai.STTType
		expectedSTTLanguage string
		expectedRagID       uuid.UUID
		expectedToolNames   []amtool.ToolName
		expectedRes         string
	}{
		{
			name: "normal without tool_names",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   nil,
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with tool_names",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia","tool_names":["connect_call","send_email"]}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   []amtool.ToolName{amtool.ToolNameConnectCall, amtool.ToolNameSendEmail},
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with all tools enabled",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia","tool_names":["all"]}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   []amtool.ToolName{amtool.ToolNameAll},
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with valid rag_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			expectedToolNames:   nil,
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with empty rag_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","rag_id":"","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   nil,
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with stt_language",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia","stt_language":"ko-KR"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "ko-KR",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   nil,
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AICreate(
				req.Context(),
				tt.agent,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedEngineModel,
				tt.expectedParameter,
				tt.expectedEngineKey,
				tt.expectedRagID,
				tt.expectedInitPrompt,
				tt.expectedTTSType,
				tt.expectedTTSVoiceID,
				tt.expectedSTTType,
				tt.expectedSTTLanguage,
				tt.expectedToolNames,
			).Return(tt.responseAI, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetAis(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAIs []*amai.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAIs: []*amai.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 items",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAIs: []*amai.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6a812daf-6ca6-4c34-892f-6e83dfd976f2"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("aff6883a-b24f-4d93-ba09-32a276cedcb7"),
					},
					TMCreate: timePtr("2020-09-20T03:23:22.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e9a4b1e2-100a-4433-a854-e4fb9b668681"),
					},
					TMCreate: timePtr("2020-09-20T03:23:23.995000Z"),
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6a812daf-6ca6-4c34-892f-6e83dfd976f2","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null},{"id":"aff6883a-b24f-4d93-ba09-32a276cedcb7","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:22.995Z","tm_update":null,"tm_delete":null},{"id":"e9a4b1e2-100a-4433-a854-e4fb9b668681","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:23.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:23.995000Z"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AIGetsByCustomerID(req.Context(), tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseAIs, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetAisId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAI *amai.WebhookMessage

		expectAIID uuid.UUID
		expectRes  string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/07f52215-8366-4060-902f-a86857243351",

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
				},
			},

			expectAIID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
			expectRes:  `{"id":"07f52215-8366-4060-902f-a86857243351","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AIGet(req.Context(), tt.agent, tt.expectAIID).Return(tt.responseAI, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_DeleteAisId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAI *amai.WebhookMessage

		expectAIID uuid.UUID
		expectRes  string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/ab6f6c84-b9c2-4350-9978-4336b677603c",

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				},
			},

			expectAIID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
			expectRes:  `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().AIDelete(req.Context(), tt.agent, tt.expectAIID).Return(tt.responseAI, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_PutAisId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAI *amai.WebhookMessage

		expectedAIID        uuid.UUID
		expectedName        string
		expectedDetail      string
		epxectedEngineModel amai.EngineModel
		expectedParameter   map[string]any
		expectedEngineKey   string
		expectedInitPrompt  string
		expectedTTSType     amai.TTSType
		expectedTTSVoiceID  string
		expectedSTTType     amai.STTType
		expectedSTTLanguage string
		expectedRagID       uuid.UUID
		expectedToolNames   []amtool.ToolName
		expectedRes         string
	}{
		{
			name: "normal without tool_names",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectedAIID:        uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			epxectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   nil,
			expectedRes:         `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with tool_names",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia","tool_names":["connect_call","send_email"]}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectedAIID:        uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			epxectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   []amtool.ToolName{amtool.ToolNameConnectCall, amtool.ToolNameSendEmail},
			expectedRes:         `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with valid rag_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectedAIID:        uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			epxectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			expectedToolNames:   nil,
			expectedRes:         `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with empty rag_id",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","rag_id":"","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectedAIID:        uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			epxectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   nil,
			expectedRes:         `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "with stt_language",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/ais/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_model":"openai.gpt-5","parameter":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia","stt_language":"ko-KR"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectedAIID:        uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			epxectedEngineModel: amai.EngineModelOpenaiGPT5,
			expectedParameter: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:   "test engine key",
			expectedInitPrompt:  "test init prompt",
			expectedTTSType:     amai.TTSTypeElevenLabs,
			expectedTTSVoiceID:  "test voice id",
			expectedSTTType:     amai.STTTypeCartesia,
			expectedSTTLanguage: "ko-KR",
			expectedRagID:       uuid.Nil,
			expectedToolNames:   nil,
			expectedRes:         `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000","rag_id":"00000000-0000-0000-0000-000000000000","current_prompt_history_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AIUpdate(
				req.Context(),
				tt.agent,
				tt.expectedAIID,
				tt.expectedName,
				tt.expectedDetail,
				tt.epxectedEngineModel,
				tt.expectedParameter,
				tt.expectedEngineKey,
				tt.expectedRagID,
				tt.expectedInitPrompt,
				tt.expectedTTSType,
				tt.expectedTTSVoiceID,
				tt.expectedSTTType,
				tt.expectedSTTLanguage,
				tt.expectedToolNames,
			).Return(tt.responseAI, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

// Test_aisPost_MissingAuthIdentity verifies PostAis emits the canonical
// UNAUTHENTICATED / AUTHENTICATION_REQUIRED envelope when auth_identity
// is missing from the gin context.
func Test_aisPost_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPost, "/ais",
		[]byte(`{"name":"test","detail":"test","engine_model":"openai.gpt-4","engine_key":"key","init_prompt":"hi","tts_type":"google","tts_voice_id":"en-US","stt_type":"google"}`))
}

// Test_aisPost_InvalidJSONBody verifies PostAis rejects malformed JSON
// with INVALID_ARGUMENT / INVALID_JSON_BODY.
func Test_aisPost_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("c96bf1c2-a2e9-11ec-a8e3-a716ee72ed9d"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodPost, "/ais", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY")
}

// Test_aisIDPut_InvalidID verifies that a malformed UUID in the path
// triggers INVALID_ARGUMENT / INVALID_ID before the servicehandler is
// consulted.
func Test_aisIDPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("c96bf1c2-a2e9-11ec-a8e3-a716ee72ed9d"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// "not-a-uuid" passes the path-shape check but uuid.FromStringOrNil
	// returns uuid.Nil, so the handler rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodPut, "/ais/not-a-uuid",
		bytes.NewBufferString(`{"name":"test","detail":"test","engine_model":"openai.gpt-4","engine_key":"key","init_prompt":"hi","tts_type":"google","tts_voice_id":"en-US","stt_type":"google"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID")
}
