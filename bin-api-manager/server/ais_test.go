package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAis(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAI *amai.WebhookMessage

		expectedName        string
		expectedDetail      string
		expectedEngineType  amai.EngineType
		expectedEngineModel amai.EngineModel
		expectedEngineData  map[string]any
		expectedEngineKey   string
		expectedInitPrompt  string
		expectedTTSType     amai.TTSType
		expectedTTSVoiceID  string
		expectedSTTType     amai.STTType
		expectedRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/ais",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_type":"","engine_model":"openai.gpt-4","engine_data":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineType:  amai.EngineTypeNone,
			expectedEngineModel: amai.EngineModelOpenaiGPT4,
			expectedEngineData: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:  "test engine key",
			expectedInitPrompt: "test init prompt",
			expectedTTSType:    amai.TTSTypeElevenLabs,
			expectedTTSVoiceID: "test voice id",
			expectedSTTType:    amai.STTTypeCartesia,
			expectedRes:        `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AICreate(
				req.Context(),
				&tt.agent,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedEngineType,
				tt.expectedEngineModel,
				tt.expectedEngineData,
				tt.expectedEngineKey,
				tt.expectedInitPrompt,
				tt.expectedTTSType,
				tt.expectedTTSVoiceID,
				tt.expectedSTTType,
				nil, // toolNames - not yet exposed in OpenAPI
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
		agent amagent.Agent

		reqQuery string

		responseAIs []*amai.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/ais?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseAIs: []*amai.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/ais?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseAIs: []*amai.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6a812daf-6ca6-4c34-892f-6e83dfd976f2"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("aff6883a-b24f-4d93-ba09-32a276cedcb7"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e9a4b1e2-100a-4433-a854-e4fb9b668681"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"6a812daf-6ca6-4c34-892f-6e83dfd976f2","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"},{"id":"aff6883a-b24f-4d93-ba09-32a276cedcb7","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:22.995000"},{"id":"e9a4b1e2-100a-4433-a854-e4fb9b668681","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:23.995000"}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AIGetsByCustomerID(req.Context(), &tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseAIs, nil)

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
		agent amagent.Agent

		reqQuery string

		responseAI *amai.WebhookMessage

		expectAIID uuid.UUID
		expectRes  string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/ais/07f52215-8366-4060-902f-a86857243351",

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
				},
			},

			expectAIID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
			expectRes:  `{"id":"07f52215-8366-4060-902f-a86857243351","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AIGet(req.Context(), &tt.agent, tt.expectAIID).Return(tt.responseAI, nil)

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
		agent amagent.Agent

		reqQuery string

		responseAI *amai.WebhookMessage

		expectAIID uuid.UUID
		expectRes  string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/ais/ab6f6c84-b9c2-4350-9978-4336b677603c",

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				},
			},

			expectAIID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
			expectRes:  `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().AIDelete(req.Context(), &tt.agent, tt.expectAIID).Return(tt.responseAI, nil)

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
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAI *amai.WebhookMessage

		expectedAIID        uuid.UUID
		expectedName        string
		expectedDetail      string
		expectedEngineType  amai.EngineType
		epxectedEngineModel amai.EngineModel
		expectedEngineData  map[string]any
		expectedEngineKey   string
		expectedInitPrompt  string
		expectedTTSType     amai.TTSType
		expectedTTSVoiceID  string
		expectedSTTType     amai.STTType
		expectedRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/ais/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_type":"","engine_model":"openai.gpt-4","engine_data":{"key1":"val1"},"engine_key":"test engine key","init_prompt":"test init prompt","tts_type":"elevenlabs","tts_voice_id":"test voice id","stt_type":"cartesia"}`),

			responseAI: &amai.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectedAIID:        uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedEngineType:  amai.EngineTypeNone,
			epxectedEngineModel: amai.EngineModelOpenaiGPT4,
			expectedEngineData: map[string]any{
				"key1": "val1",
			},
			expectedEngineKey:  "test engine key",
			expectedInitPrompt: "test init prompt",
			expectedTTSType:    amai.TTSTypeElevenLabs,
			expectedTTSVoiceID: "test voice id",
			expectedSTTType:    amai.STTTypeCartesia,
			expectedRes:        `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AIUpdate(
				req.Context(),
				&tt.agent,
				tt.expectedAIID,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedEngineType,
				tt.epxectedEngineModel,
				tt.expectedEngineData,
				tt.expectedEngineKey,
				tt.expectedInitPrompt,
				tt.expectedTTSType,
				tt.expectedTTSVoiceID,
				tt.expectedSTTType,
				nil, // toolNames - not yet exposed in OpenAPI
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
