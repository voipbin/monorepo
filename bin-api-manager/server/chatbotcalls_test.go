package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostChatbotcalls(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		response *cbchatbotcall.WebhookMessage

		expectChatbotID     uuid.UUID
		expectReferenceType cbchatbotcall.ReferenceType
		expectReferenceID   uuid.UUID
		expectGender        cbchatbotcall.Gender
		expectLanguage      string
		expectRes           string
	}{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotcalls",
			reqBody:  []byte(`{"chatbot_id":"d9e18e8c-efac-11ef-903a-9710c6837217","reference_type":"call","reference_id":"da12e23e-efac-11ef-aa18-172cb9693b33","gender":"male","language":"en-US"}`),

			response: &cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b71393dc-efac-11ef-829f-5330fc080fd2"),
				},
				Messages: []cbchatbotcall.Message{
					{
						Role:    cbchatbotcall.MessageRoleUser,
						Content: "hello world",
					},
				},
			},

			expectChatbotID:     uuid.FromStringOrNil("d9e18e8c-efac-11ef-903a-9710c6837217"),
			expectReferenceType: cbchatbotcall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("da12e23e-efac-11ef-aa18-172cb9693b33"),
			expectGender:        cbchatbotcall.GenderMale,
			expectLanguage:      "en-US",
			expectRes:           `{"id":"b71393dc-efac-11ef-829f-5330fc080fd2","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","messages":[{"role":"user","content":"hello world"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ChatbotcallCreate(
				req.Context(),
				&tt.agent,
				tt.expectChatbotID,
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectGender,
				tt.expectLanguage,
			).Return(tt.response, nil)

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

func Test_chatbotcallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbotcalls []*cbchatbotcall.WebhookMessage

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

			reqQuery: "/chatbotcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatbotcalls: []*cbchatbotcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("fa136fec-eca6-4958-b9a8-21fd8d61b8aa"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"fa136fec-eca6-4958-b9a8-21fd8d61b8aa","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatbotcalls: []*cbchatbotcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f7576695-a944-4427-b7d6-1a776f83aa9a"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f34d51d0-4a74-40d7-9050-edc6fd1654f7"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("227edc68-c2da-4ed8-bd28-08d8fab8c17c"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"f7576695-a944-4427-b7d6-1a776f83aa9a","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000"},{"id":"f34d51d0-4a74-40d7-9050-edc6fd1654f7","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:22.995000"},{"id":"227edc68-c2da-4ed8-bd28-08d8fab8c17c","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:23.995000"}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().ChatbotcallGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseChatbotcalls, nil)

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

func Test_chatbotcallsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbotcall *cbchatbotcall.WebhookMessage

		expectChatbotcallID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotcalls/f199188b-8d78-4778-8891-8f276cd56de5",

			responseChatbotcall: &cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f199188b-8d78-4778-8891-8f276cd56de5"),
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("f199188b-8d78-4778-8891-8f276cd56de5"),
			expectRes:           `{"id":"f199188b-8d78-4778-8891-8f276cd56de5","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().ChatbotcallGet(req.Context(), &tt.agent, tt.expectChatbotcallID).Return(tt.responseChatbotcall, nil)

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

func Test_chatbotcallsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbotcall *cbchatbotcall.WebhookMessage

		expectChatbotcallID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotcalls/c1a95988-5382-4769-98a9-b404823a64bf",

			responseChatbotcall: &cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
			expectRes:           `{"id":"c1a95988-5382-4769-98a9-b404823a64bf","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().ChatbotcallDelete(req.Context(), &tt.agent, tt.expectChatbotcallID).Return(tt.responseChatbotcall, nil)

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

func Test_PostChatbotcallsIdMessages(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		response *cbchatbotcall.WebhookMessage

		expectChatbotcallID uuid.UUID
		expectRole          cbchatbotcall.MessageRole
		expectText          string
		expectRes           string
	}{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotcalls/81eb3506-efad-11ef-8d67-133377751a43/messages",
			reqBody:  []byte(`{"role":"user","text":"hello world"}`),

			response: &cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b71393dc-efac-11ef-829f-5330fc080fd2"),
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("81eb3506-efad-11ef-8d67-133377751a43"),
			expectRole:          cbchatbotcall.MessageRoleUser,
			expectText:          "hello world",
			expectRes:           `{"id":"b71393dc-efac-11ef-829f-5330fc080fd2","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ChatbotcallSendMessage(
				req.Context(),
				&tt.agent,
				tt.expectChatbotcallID,
				tt.expectRole,
				tt.expectText,
			).Return(tt.response, nil)

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
