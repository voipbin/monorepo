package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	chmedia "monorepo/bin-chat-manager/models/media"
	chmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_chatroommessagesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseMessagechatroom *chmessagechatroom.WebhookMessage

		expectChatroomID uuid.UUID
		expectText       string
		expectMedias     []chmedia.Media
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatroommessages",
			reqBody:  []byte(`{"chatroom_id":"eac45700-bbfc-11ee-8a32-ef7ecccd51ae","text":"test text","medias":[{"type":"address","address":{"type":"tel","target":"+123456789"}}]}`),

			responseMessagechatroom: &chmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eaf8712a-bbfc-11ee-96cd-4f42c7f2accd"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("eac45700-bbfc-11ee-8a32-ef7ecccd51ae"),
			expectText:       "test text",
			expectMedias: []chmedia.Media{
				{
					Type: chmedia.TypeAddress,
					Address: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+123456789",
					},
				},
			},
			expectRes: `{"id":"eaf8712a-bbfc-11ee-96cd-4f42c7f2accd","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroommessageCreate(
				req.Context(),
				&tt.agent,
				tt.expectChatroomID,
				tt.expectText,
				tt.expectMedias,
			.Return(tt.responseMessagechatroom, nil)

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

func Test_chatroommessagesGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatroommessages []*chmessagechatroom.WebhookMessage

		expectChatroomID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string
		expectRes        string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatroommessages?chatroom_id=5f0a2dd4-389f-11ed-9b81-5b36a877165b&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatroommessages: []*chmessagechatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6402aab8-389b-11ed-b537-57de22d7f36f"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectChatroomID: uuid.FromStringOrNil("5f0a2dd4-389f-11ed-9b81-5b36a877165b"),
			expectPageSize:   10,
			expectPageToken:  "2020-09-20 03:23:20.995000",
			expectRes:        `{"result":[{"id":"6402aab8-389b-11ed-b537-57de22d7f36f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatroommessages?chatroom_id=d5246de0-389f-11ed-9790-23d937d90529&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatroommessages: []*chmessagechatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d54e7a4a-389f-11ed-9891-97fc0cc84808"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d575e648-389f-11ed-b67b-5b8f49688d39"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d6687bb0-389f-11ed-94c7-5b68300fcff9"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectChatroomID: uuid.FromStringOrNil("d5246de0-389f-11ed-9790-23d937d90529"),
			expectPageSize:   10,
			expectPageToken:  "2020-09-20 03:23:20.995000",
			expectRes:        `{"result":[{"id":"d54e7a4a-389f-11ed-9891-97fc0cc84808","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"d575e648-389f-11ed-b67b-5b8f49688d39","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"d6687bb0-389f-11ed-94c7-5b68300fcff9","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ChatroommessageGetsByChatroomID(req.Context(), &tt.agent, tt.expectChatroomID, tt.expectPageSize, tt.expectPageToken.Return(tt.responseChatroommessages, nil)

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

func Test_chatroommessagesIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatroommessage *chmessagechatroom.WebhookMessage

		expectChatroommessageID uuid.UUID
		expectRes               string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatroommessages/0e2aab86-38a0-11ed-895e-c3af9cdbb491",

			responseChatroommessage: &chmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0e2aab86-38a0-11ed-895e-c3af9cdbb491"),
				},
			},

			expectChatroommessageID: uuid.FromStringOrNil("0e2aab86-38a0-11ed-895e-c3af9cdbb491"),
			expectRes:               `{"id":"0e2aab86-38a0-11ed-895e-c3af9cdbb491","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroommessageGet(req.Context(), &tt.agent, tt.expectChatroommessageID.Return(tt.responseChatroommessage, nil)

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

func Test_chatroommessagesIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatroommessage *chmessagechatroom.WebhookMessage

		expectChatroommessageID uuid.UUID
		expectRes               string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatroommessages/72c7cdee-38a0-11ed-a3b4-737e737b5977",

			responseChatroommessage: &chmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72c7cdee-38a0-11ed-a3b4-737e737b5977"),
				},
			},

			expectChatroommessageID: uuid.FromStringOrNil("72c7cdee-38a0-11ed-a3b4-737e737b5977"),
			expectRes:               `{"id":"72c7cdee-38a0-11ed-a3b4-737e737b5977","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroommessageDelete(req.Context(), &tt.agent, tt.expectChatroommessageID.Return(tt.responseChatroommessage, nil)

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
