package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cbmessage "monorepo/bin-chatbot-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostChatbotmessages(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseChatbotmessage *cbmessage.WebhookMessage

		expectChatbotcallID uuid.UUID
		expectRole          cbmessage.Role
		expectContent       string
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotmessages",
			reqBody:  []byte(`{"chatbotcall_id":"9fa30c3a-f31e-11ef-a4df-9f6bf108282e","role":"user","content":"test text"}`),

			responseChatbotmessage: &cbmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("9fa30c3a-f31e-11ef-a4df-9f6bf108282e"),
			expectRole:          cbmessage.RoleUser,
			expectContent:       "test text",

			expectRes: `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":""}`,
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
			mockSvc.EXPECT().ChatbotmessageCreate(
				req.Context(),
				&tt.agent,
				tt.expectChatbotcallID,
				tt.expectRole,
				tt.expectContent,
			).Return(tt.responseChatbotmessage, nil)

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

func Test_GetChatbotmessages(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbotmessages []*cbmessage.WebhookMessage

		expectChatbotcallID uuid.UUID
		expectPageSize      uint64
		expectPageToken     string
		expectRes           string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotmessages?page_size=10&page_token=2020-09-20%2003:23:20.995000&chatbotcall_id=ecebd332-f31e-11ef-9ab5-33426e3ee4ff",

			responseChatbotmessages: []*cbmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ed2346dc-f31e-11ef-acd5-67a8f966fe17"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("ecebd332-f31e-11ef-9ab5-33426e3ee4ff"),
			expectPageSize:      10,
			expectPageToken:     "2020-09-20 03:23:20.995000",
			expectRes:           `{"result":[{"id":"ed2346dc-f31e-11ef-acd5-67a8f966fe17","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotmessages?page_size=10&page_token=2020-09-20%2003:23:20.995000&chatbotcall_id=ed487b96-f31e-11ef-9337-e792818f3609",

			responseChatbotmessages: []*cbmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1707f380-f31f-11ef-bfe4-7ff769b357b3"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("17268426-f31f-11ef-aa11-6f21c1723af6"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("17468500-f31f-11ef-b7b6-9b29397f4894"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectChatbotcallID: uuid.FromStringOrNil("ed487b96-f31e-11ef-9337-e792818f3609"),
			expectPageSize:      10,
			expectPageToken:     "2020-09-20 03:23:20.995000",
			expectRes:           `{"result":[{"id":"1707f380-f31f-11ef-bfe4-7ff769b357b3","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:21.995000"},{"id":"17268426-f31f-11ef-aa11-6f21c1723af6","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:22.995000"},{"id":"17468500-f31f-11ef-b7b6-9b29397f4894","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":"","tm_create":"2020-09-20T03:23:23.995000"}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().ChatbotmessageGetsByChatbotcallID(req.Context(), &tt.agent, tt.expectChatbotcallID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseChatbotmessages, nil)

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

func Test_GetChatbotmessagesId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbot *cbmessage.WebhookMessage

		expectChatbotmessageID uuid.UUID
		expectRes              string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotmessages/796924c2-f31f-11ef-8589-c3efd79e11d5",

			responseChatbot: &cbmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("796924c2-f31f-11ef-8589-c3efd79e11d5"),
				},
			},

			expectChatbotmessageID: uuid.FromStringOrNil("796924c2-f31f-11ef-8589-c3efd79e11d5"),
			expectRes:              `{"id":"796924c2-f31f-11ef-8589-c3efd79e11d5","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":""}`,
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
			mockSvc.EXPECT().ChatbotmessageGet(req.Context(), &tt.agent, tt.expectChatbotmessageID).Return(tt.responseChatbot, nil)

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

func Test_DeleteChatbotmessagesId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbotmessage *cbmessage.WebhookMessage

		expectChatbotmessageID uuid.UUID
		expectRes              string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotmessages/b3fc4312-f31f-11ef-8661-939776978f23",

			responseChatbotmessage: &cbmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				},
			},

			expectChatbotmessageID: uuid.FromStringOrNil("b3fc4312-f31f-11ef-8661-939776978f23"),
			expectRes:              `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"00000000-0000-0000-0000-000000000000","chatbotcall_id":"00000000-0000-0000-0000-000000000000","role":"","content":"","direction":""}`,
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
			mockSvc.EXPECT().ChatbotmessageDelete(req.Context(), &tt.agent, tt.expectChatbotmessageID).Return(tt.responseChatbotmessage, nil)

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
