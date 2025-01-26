package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_chatbotcallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatbotcalls []*cmchatbotcall.WebhookMessage

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

			responseChatbotcalls: []*cmchatbotcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("fa136fec-eca6-4958-b9a8-21fd8d61b8aa"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"fa136fec-eca6-4958-b9a8-21fd8d61b8aa","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatbotcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatbotcalls: []*cmchatbotcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("f7576695-a944-4427-b7d6-1a776f83aa9a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("f34d51d0-4a74-40d7-9050-edc6fd1654f7"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("227edc68-c2da-4ed8-bd28-08d8fab8c17c"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"f7576695-a944-4427-b7d6-1a776f83aa9a","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"f34d51d0-4a74-40d7-9050-edc6fd1654f7","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"227edc68-c2da-4ed8-bd28-08d8fab8c17c","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

		responseChatbotcall *cmchatbotcall.WebhookMessage

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

			responseChatbotcall: &cmchatbotcall.WebhookMessage{
				ID: uuid.FromStringOrNil("f199188b-8d78-4778-8891-8f276cd56de5"),
			},

			expectChatbotcallID: uuid.FromStringOrNil("f199188b-8d78-4778-8891-8f276cd56de5"),
			expectRes:           `{"id":"f199188b-8d78-4778-8891-8f276cd56de5","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

		responseChatbotcall *cmchatbotcall.WebhookMessage

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

			responseChatbotcall: &cmchatbotcall.WebhookMessage{
				ID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
			},

			expectChatbotcallID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
			expectRes:           `{"id":"c1a95988-5382-4769-98a9-b404823a64bf","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
