package service_agents

import (
	"bytes"
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
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
		reqBody  request.BodyServiceAgentsChatroommessagesPOST

		responseChatroommessage *chatmessagechatroom.WebhookMessage

		expectedChatroomID uuid.UUID
		expectedText       string
		expectedMedias     []chatmedia.Media
		expectedRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/chatroommessages",
			reqBody: request.BodyServiceAgentsChatroommessagesPOST{
				ChatroomID: uuid.FromStringOrNil("7ee036a0-bd43-11ef-bff0-bfe67915b551"),
				Text:       "Hello, World!",
				Medias:     []chatmedia.Media{},
			},

			responseChatroommessage: &chatmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f497462-bd43-11ef-8385-838cfb49385f"),
				},
			},

			expectedChatroomID: uuid.FromStringOrNil("7ee036a0-bd43-11ef-bff0-bfe67915b551"),
			expectedText:       "Hello, World!",
			expectedMedias:     []chatmedia.Media{},
			expectedRes:        `{"id":"7f497462-bd43-11ef-8385-838cfb49385f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			mockSvc.EXPECT().ServiceAgentChatroommessageCreate(req.Context(), &tt.agent, tt.expectedChatroomID, tt.expectedText, tt.expectedMedias).Return(tt.responseChatroommessage, nil)

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

func Test_chatroommessagesGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery                 string
		responseChatroommessages []*chatmessagechatroom.WebhookMessage

		expectChatroomID uuid.UUID
		expectPageToken  string
		expectPageSize   uint64
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/chatroommessages?page_token=2020-09-20%2003:23:20.995000&page_size=10&chatroom_id=9d6ec3e6-bd45-11ef-8188-ab883a261177",

			responseChatroommessages: []*chatmessagechatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1e5ee31c-3bb1-11ef-983b-67a86019900d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1ef41086-3bb1-11ef-a7b9-a39300b3cc30"),
					},
				},
			},

			expectChatroomID: uuid.FromStringOrNil("9d6ec3e6-bd45-11ef-8188-ab883a261177"),
			expectPageToken:  "2020-09-20 03:23:20.995000",
			expectPageSize:   10,
			expectRes:        `{"result":[{"id":"1e5ee31c-3bb1-11ef-983b-67a86019900d","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"1ef41086-3bb1-11ef-a7b9-a39300b3cc30","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentChatroommessageGets(req.Context(), &tt.agent, tt.expectChatroomID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseChatroommessages, nil)

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

		reqQuery                string
		responseChatroommessage *chatmessagechatroom.WebhookMessage

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

			reqQuery: "/v1.0/service_agents/chatroommessages/463f0742-bd46-11ef-be81-a3ad3d1e8fd6",

			responseChatroommessage: &chatmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("463f0742-bd46-11ef-be81-a3ad3d1e8fd6"),
				},
			},

			expectChatroommessageID: uuid.FromStringOrNil("463f0742-bd46-11ef-be81-a3ad3d1e8fd6"),
			expectRes:               `{"id":"463f0742-bd46-11ef-be81-a3ad3d1e8fd6","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentChatroommessageGet(req.Context(), &tt.agent, tt.expectChatroommessageID).Return(tt.responseChatroommessage, nil)

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

		reqQuery                string
		responseChatroommessage *chatmessagechatroom.WebhookMessage

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

			reqQuery: "/v1.0/service_agents/chatroommessages/92850d4a-bd46-11ef-89c8-0f2d2afc4982",

			responseChatroommessage: &chatmessagechatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("92850d4a-bd46-11ef-89c8-0f2d2afc4982"),
				},
			},

			expectChatroommessageID: uuid.FromStringOrNil("92850d4a-bd46-11ef-89c8-0f2d2afc4982"),
			expectRes:               `{"id":"92850d4a-bd46-11ef-89c8-0f2d2afc4982","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentChatroommessageDelete(req.Context(), &tt.agent, tt.expectChatroommessageID).Return(tt.responseChatroommessage, nil)

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
