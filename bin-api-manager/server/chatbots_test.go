package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmchatbot "monorepo/bin-chatbot-manager/models/chatbot"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_chatbotsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseChatbot *cmchatbot.WebhookMessage

		expectName                string
		expectDetail              string
		expectEngineType          cmchatbot.EngineType
		expectEngineModel         cmchatbot.EngineModel
		expectInitPrompt          string
		expectCredentialBase64    string
		expectCredentialProjectID string
		expectRes                 string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbots",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_type":"","engine_model":"openai.gpt-4","init_prompt":"test init prompt","credential_base64":"test credential base64","credential_project_id":"test credential project id"}`),

			responseChatbot: &cmchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectName:                "test name",
			expectDetail:              "test detail",
			expectEngineType:          cmchatbot.EngineTypeNone,
			expectEngineModel:         cmchatbot.EngineModelOpenaiGPT4,
			expectInitPrompt:          "test init prompt",
			expectCredentialBase64:    "test credential base64",
			expectCredentialProjectID: "test credential project id",
			expectRes:                 `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().ChatbotCreate(
				req.Context(),
				&tt.agent,
				tt.expectName,
				tt.expectDetail,
				tt.expectEngineType,
				tt.expectEngineModel,
				tt.expectInitPrompt,
				tt.expectCredentialBase64,
				tt.expectCredentialProjectID,
			).Return(tt.responseChatbot, nil)

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

func Test_chatbotsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbots []*cmchatbot.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbots?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatbots: []*cmchatbot.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbots?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatbots: []*cmchatbot.WebhookMessage{
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

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"6a812daf-6ca6-4c34-892f-6e83dfd976f2","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"},{"id":"aff6883a-b24f-4d93-ba09-32a276cedcb7","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:22.995000"},{"id":"e9a4b1e2-100a-4433-a854-e4fb9b668681","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:23.995000"}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().ChatbotGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseChatbots, nil)

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

func Test_chatbotsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbot *cmchatbot.WebhookMessage

		expectChatbotID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbots/07f52215-8366-4060-902f-a86857243351",

			responseChatbot: &cmchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
				},
			},

			expectChatbotID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
			expectRes:       `{"id":"07f52215-8366-4060-902f-a86857243351","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().ChatbotGet(req.Context(), &tt.agent, tt.expectChatbotID).Return(tt.responseChatbot, nil)

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

func Test_chatbotsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbot *cmchatbot.WebhookMessage

		expectChatbotID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbots/ab6f6c84-b9c2-4350-9978-4336b677603c",

			responseChatbot: &cmchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				},
			},

			expectChatbotID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
			expectRes:       `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().ChatbotDelete(req.Context(), &tt.agent, tt.expectChatbotID).Return(tt.responseChatbot, nil)

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

func Test_chatbotsIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseChatbot *cmchatbot.WebhookMessage

		expectChatbotID           uuid.UUID
		expectName                string
		expectDetail              string
		expectEngineType          cmchatbot.EngineType
		epxectEngineModel         cmchatbot.EngineModel
		expectInitPrompt          string
		expectCredentialBase64    string
		expectCredentialProjectID string
		expectRes                 string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbots/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","engine_type":"","engine_model":"openai.gpt-4","init_prompt":"test init prompt","credential_base64":"test credential base64","credential_project_id":"test credential project id"}`),

			responseChatbot: &cmchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			expectChatbotID:           uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectName:                "test name",
			expectDetail:              "test detail",
			expectEngineType:          cmchatbot.EngineTypeNone,
			epxectEngineModel:         cmchatbot.EngineModelOpenaiGPT4,
			expectInitPrompt:          "test init prompt",
			expectCredentialBase64:    "test credential base64",
			expectCredentialProjectID: "test credential project id",
			expectRes:                 `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().ChatbotUpdate(
				req.Context(),
				&tt.agent,
				tt.expectChatbotID,
				tt.expectName,
				tt.expectDetail,
				tt.expectEngineType,
				tt.epxectEngineModel,
				tt.expectInitPrompt,
				tt.expectCredentialBase64,
				tt.expectCredentialProjectID,
			).Return(tt.responseChatbot, nil)

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
