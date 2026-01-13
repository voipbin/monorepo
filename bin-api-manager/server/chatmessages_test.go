package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmmedia "monorepo/bin-chat-manager/models/media"
	cmmessagechat "monorepo/bin-chat-manager/models/messagechat"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_chatmessagesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseMessagechat *cmmessagechat.WebhookMessage

		expectChatID uuid.UUID
		expectSource commonaddress.Address
		expectType   cmmessagechat.Type
		expectText   string
		expectMedias []cmmedia.Media
		expectRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatmessages",
			reqBody:  []byte(`{"chat_id":"ce2f5fd6-389a-11ed-b2f2-732e87938cc1","source":{"type":"tel","target":"+821100000001"},"type":"normal","text":"test text","medias":[]}`),

			responseMessagechat: &cmmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("df61b3d0-389a-11ed-ab0f-6b738df69dbe"),
				},
			},

			expectChatID: uuid.FromStringOrNil("ce2f5fd6-389a-11ed-b2f2-732e87938cc1"),
			expectSource: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectType:   cmmessagechat.TypeNormal,
			expectText:   "test text",
			expectMedias: []cmmedia.Media{},
			expectRes:    `{"id":"df61b3d0-389a-11ed-ab0f-6b738df69dbe","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-type", "application/json")

			mockSvc.EXPECT().ChatmessageCreate(
				req.Context(),
				&tt.agent,
				tt.expectChatID,
				tt.expectSource,
				tt.expectType,
				tt.expectText,
				tt.expectMedias,
			).Return(tt.responseMessagechat, nil)

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

func Test_chatmessagesGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatmessages []*cmmessagechat.WebhookMessage

		expectChatID    uuid.UUID
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

			reqQuery: "/chatmessages?chat_id=63bdfa08-389b-11ed-9410-f3f0330b6445&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatmessages: []*cmmessagechat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6402aab8-389b-11ed-b537-57de22d7f36f"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectChatID:    uuid.FromStringOrNil("63bdfa08-389b-11ed-9410-f3f0330b6445"),
			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"6402aab8-389b-11ed-b537-57de22d7f36f","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatmessages?chat_id=44d163ac-389e-11ed-aba8-9bb6e419cfc3&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatmessages: []*cmmessagechat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("57f0ff4c-389e-11ed-9694-7b4ee5f2ad11"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5819a442-389e-11ed-bdff-177962843d00"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("583d596e-389e-11ed-84ff-83124069fc30"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectChatID:    uuid.FromStringOrNil("44d163ac-389e-11ed-aba8-9bb6e419cfc3"),
			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"57f0ff4c-389e-11ed-9694-7b4ee5f2ad11","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"5819a442-389e-11ed-bdff-177962843d00","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"583d596e-389e-11ed-84ff-83124069fc30","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ChatmessageGetsByChatID(req.Context(), &tt.agent, tt.expectChatID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseChatmessages, nil)

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

func Test_chatmessagesIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatmessage *cmmessagechat.WebhookMessage

		expectChatmessageID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatmessages/97c8071e-389e-11ed-b5ac-c3e9dbd9d066",

			responseChatmessage: &cmmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("97c8071e-389e-11ed-b5ac-c3e9dbd9d066"),
				},
			},

			expectChatmessageID: uuid.FromStringOrNil("97c8071e-389e-11ed-b5ac-c3e9dbd9d066"),
			expectRes:           `{"id":"97c8071e-389e-11ed-b5ac-c3e9dbd9d066","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatmessageGet(req.Context(), &tt.agent, tt.expectChatmessageID).Return(tt.responseChatmessage, nil)

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

func Test_chatmessagesIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatmessage *cmmessagechat.WebhookMessage

		expectChatmessageID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatmessages/fa4c5552-389e-11ed-adca-eb6b8dfe8032",

			responseChatmessage: &cmmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa4c5552-389e-11ed-adca-eb6b8dfe8032"),
				},
			},

			expectChatmessageID: uuid.FromStringOrNil("fa4c5552-389e-11ed-adca-eb6b8dfe8032"),
			expectRes:           `{"id":"fa4c5552-389e-11ed-adca-eb6b8dfe8032","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatmessageDelete(req.Context(), &tt.agent, tt.expectChatmessageID).Return(tt.responseChatmessage, nil)

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
