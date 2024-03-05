package chatroommessages

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_chatroommessagesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyChatroommessagesPOST

		response *chatmessagechatroom.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatroommessages",
			request.BodyChatroommessagesPOST{
				ChatroomID: uuid.FromStringOrNil("eac45700-bbfc-11ee-8a32-ef7ecccd51ae"),
				Text:       "test text",
			},

			&chatmessagechatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("eaf8712a-bbfc-11ee-96cd-4f42c7f2accd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-type", "application/json")

			mockSvc.EXPECT().ChatroommessageCreate(
				req.Context(),
				&tt.agent,
				tt.reqBody.ChatroomID,
				tt.reqBody.Text,
			).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_chatmessagesGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery   string
		chatroomID uuid.UUID
		pageSize   uint64
		pageToken  string

		responseChatroommessages []*chatmessagechatroom.WebhookMessage
		expectRes                string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatroommessages?chatroom_id=5f0a2dd4-389f-11ed-9b81-5b36a877165b&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			uuid.FromStringOrNil("5f0a2dd4-389f-11ed-9b81-5b36a877165b"),
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatmessagechatroom.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("6402aab8-389b-11ed-b537-57de22d7f36f"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"6402aab8-389b-11ed-b537-57de22d7f36f","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatroommessages?chatroom_id=d5246de0-389f-11ed-9790-23d937d90529&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			uuid.FromStringOrNil("d5246de0-389f-11ed-9790-23d937d90529"),
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatmessagechatroom.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("d54e7a4a-389f-11ed-9891-97fc0cc84808"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("d575e648-389f-11ed-b67b-5b8f49688d39"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("d6687bb0-389f-11ed-94c7-5b68300fcff9"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"d54e7a4a-389f-11ed-9891-97fc0cc84808","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"d575e648-389f-11ed-b67b-5b8f49688d39","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"d6687bb0-389f-11ed-94c7-5b68300fcff9","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			mockSvc.EXPECT().ChatroommessageGetsByChatroomID(req.Context(), &tt.agent, tt.chatroomID, tt.pageSize, tt.pageToken).Return(tt.responseChatroommessages, nil)

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

		reqQuery          string
		chatroommessageID uuid.UUID

		responseChatroommessage *chatmessagechatroom.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatroommessages/0e2aab86-38a0-11ed-895e-c3af9cdbb491",
			uuid.FromStringOrNil("0e2aab86-38a0-11ed-895e-c3af9cdbb491"),

			&chatmessagechatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("0e2aab86-38a0-11ed-895e-c3af9cdbb491"),
			},

			`{"id":"0e2aab86-38a0-11ed-895e-c3af9cdbb491","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			mockSvc.EXPECT().ChatroommessageGet(req.Context(), &tt.agent, tt.chatroommessageID).Return(tt.responseChatroommessage, nil)

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

		reqQuery          string
		chatroommessageID uuid.UUID

		responseChatroommessage *chatmessagechatroom.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatroommessages/72c7cdee-38a0-11ed-a3b4-737e737b5977",
			uuid.FromStringOrNil("72c7cdee-38a0-11ed-a3b4-737e737b5977"),

			&chatmessagechatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("72c7cdee-38a0-11ed-a3b4-737e737b5977"),
			},

			`{"id":"72c7cdee-38a0-11ed-a3b4-737e737b5977","customer_id":"00000000-0000-0000-0000-000000000000","agent_id":"00000000-0000-0000-0000-000000000000","chatroom_id":"00000000-0000-0000-0000-000000000000","messagechat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			mockSvc.EXPECT().ChatroommessageDelete(req.Context(), &tt.agent, tt.chatroommessageID).Return(tt.responseChatroommessage, nil)

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
