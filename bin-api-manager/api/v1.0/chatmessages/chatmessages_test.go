package chatmessages

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_chatmessagesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyChatmessagesPOST

		response *chatmessagechat.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/chatmessages",
			request.BodyChatmessagesPOST{
				ChatID: uuid.FromStringOrNil("ce2f5fd6-389a-11ed-b2f2-732e87938cc1"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Type:   chatmessagechat.TypeNormal,
				Text:   "test text",
				Medias: []media.Media{},
			},

			&chatmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("df61b3d0-389a-11ed-ab0f-6b738df69dbe"),
				},
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

			mockSvc.EXPECT().ChatmessageCreate(
				req.Context(),
				&tt.agent,
				tt.reqBody.ChatID,
				tt.reqBody.Source,
				tt.reqBody.Type,
				tt.reqBody.Text,
				tt.reqBody.Medias,
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

		reqQuery  string
		chatID    uuid.UUID
		pageSize  uint64
		pageToken string

		responseChatmessages []*chatmessagechat.WebhookMessage
		expectRes            string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/chatmessages?chat_id=63bdfa08-389b-11ed-9410-f3f0330b6445&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			uuid.FromStringOrNil("63bdfa08-389b-11ed-9410-f3f0330b6445"),
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatmessagechat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6402aab8-389b-11ed-b537-57de22d7f36f"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"6402aab8-389b-11ed-b537-57de22d7f36f","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/chatmessages?chat_id=44d163ac-389e-11ed-aba8-9bb6e419cfc3&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			uuid.FromStringOrNil("44d163ac-389e-11ed-aba8-9bb6e419cfc3"),
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatmessagechat.WebhookMessage{
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
			`{"result":[{"id":"57f0ff4c-389e-11ed-9694-7b4ee5f2ad11","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"5819a442-389e-11ed-bdff-177962843d00","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"583d596e-389e-11ed-84ff-83124069fc30","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ChatmessageGetsByChatID(req.Context(), &tt.agent, tt.chatID, tt.pageSize, tt.pageToken).Return(tt.responseChatmessages, nil)

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

		reqQuery      string
		chatmessageID uuid.UUID

		responseChatmessage *chatmessagechat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/chatmessages/97c8071e-389e-11ed-b5ac-c3e9dbd9d066",
			uuid.FromStringOrNil("97c8071e-389e-11ed-b5ac-c3e9dbd9d066"),

			&chatmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("97c8071e-389e-11ed-b5ac-c3e9dbd9d066"),
				},
			},

			`{"id":"97c8071e-389e-11ed-b5ac-c3e9dbd9d066","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatmessageGet(req.Context(), &tt.agent, tt.chatmessageID).Return(tt.responseChatmessage, nil)

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

		reqQuery      string
		chatmessageID uuid.UUID

		responseChatmessage *chatmessagechat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/chatmessages/fa4c5552-389e-11ed-adca-eb6b8dfe8032",
			uuid.FromStringOrNil("fa4c5552-389e-11ed-adca-eb6b8dfe8032"),

			&chatmessagechat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa4c5552-389e-11ed-adca-eb6b8dfe8032"),
				},
			},

			`{"id":"fa4c5552-389e-11ed-adca-eb6b8dfe8032","customer_id":"00000000-0000-0000-0000-000000000000","chat_id":"00000000-0000-0000-0000-000000000000","source":null,"type":"","text":"","medias":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatmessageDelete(req.Context(), &tt.agent, tt.chatmessageID).Return(tt.responseChatmessage, nil)

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
