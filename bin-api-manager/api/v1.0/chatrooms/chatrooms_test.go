package chatrooms

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	chatchatroom "monorepo/bin-chat-manager/models/chatroom"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
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
		reqBody  request.BodyChatroomsPOST

		response *chatchatroom.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatrooms",
			request.BodyChatroomsPOST{
				ParticipantID: []uuid.UUID{
					uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					uuid.FromStringOrNil("a5d27d76-bc05-11ee-b3db-0333d9946bda"),
				},
				Name:   "test name",
				Detail: "test detail",
			},

			&chatchatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("a5ffbb60-bc05-11ee-af79-e73be92a78fe"),
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

			mockSvc.EXPECT().ChatroomCreate(
				req.Context(),
				&tt.agent,
				tt.reqBody.ParticipantID,
				tt.reqBody.Name,
				tt.reqBody.Detail,
			).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_chatroomsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery  string
		ownerID   uuid.UUID
		pageSize  uint64
		pageToken string

		responseChatrooms []*chatchatroom.WebhookMessage
		expectRes         string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("65c83e84-8df5-11ee-ba8b-1700dcdfa8f2"),
			},

			"/v1.0/chatrooms?owner_id=f3974fc6-38a0-11ed-a40b-6fc6c6ec606e&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			uuid.FromStringOrNil("f3974fc6-38a0-11ed-a40b-6fc6c6ec606e"),
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatchatroom.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("f4037f8e-38a0-11ed-8424-5f0d424074aa"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"f4037f8e-38a0-11ed-8424-5f0d424074aa","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("65c83e84-8df5-11ee-ba8b-1700dcdfa8f2"),
			},

			"/v1.0/chatrooms?owner_id=20c40fac-38a1-11ed-83d3-93d4a1e51688&page_size=10&page_token=2020-09-20%2003:23:20.995000",
			uuid.FromStringOrNil("20c40fac-38a1-11ed-83d3-93d4a1e51688"),
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatchatroom.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("20f10a66-38a1-11ed-bd54-d7b834668361"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("211bfb54-38a1-11ed-8d88-f34522ed8844"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("21454f0e-38a1-11ed-bce4-a7b8972af690"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"20f10a66-38a1-11ed-bd54-d7b834668361","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"211bfb54-38a1-11ed-8d88-f34522ed8844","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"21454f0e-38a1-11ed-bce4-a7b8972af690","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ChatroomGetsByOwnerID(req.Context(), &tt.agent, tt.ownerID, tt.pageSize, tt.pageToken).Return(tt.responseChatrooms, nil)

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

func Test_chatroomsIDGET(t *testing.T) {

	tests := []struct {
		name     string
		customer amagent.Agent

		reqQuery          string
		chatroommessageID uuid.UUID

		responseChatroom *chatchatroom.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatrooms/5585fce6-38a1-11ed-93f2-c396374122fd",
			uuid.FromStringOrNil("5585fce6-38a1-11ed-93f2-c396374122fd"),

			&chatchatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("5585fce6-38a1-11ed-93f2-c396374122fd"),
			},

			`{"id":"5585fce6-38a1-11ed-93f2-c396374122fd","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ChatroomGet(req.Context(), &tt.customer, tt.chatroommessageID).Return(tt.responseChatroom, nil)

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

func Test_chatroomsIDDELETE(t *testing.T) {

	tests := []struct {
		name     string
		customer amagent.Agent

		reqQuery   string
		chatroomID uuid.UUID

		responseChatroom *chatchatroom.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatrooms/98799008-38a1-11ed-9406-6fa22402722c",
			uuid.FromStringOrNil("98799008-38a1-11ed-9406-6fa22402722c"),

			&chatchatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("98799008-38a1-11ed-9406-6fa22402722c"),
			},

			`{"id":"98799008-38a1-11ed-9406-6fa22402722c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ChatroomDelete(req.Context(), &tt.customer, tt.chatroomID).Return(tt.responseChatroom, nil)

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

func Test_chatroomsIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery   string
		chatroomID uuid.UUID

		reqBody request.BodyChatsIDPUT

		responseChat *chatchatroom.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatrooms/cba8e95e-bc64-11ee-9324-bb5c09ef083f",
			uuid.FromStringOrNil("cba8e95e-bc64-11ee-9324-bb5c09ef083f"),

			request.BodyChatsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},

			&chatchatroom.WebhookMessage{
				ID: uuid.FromStringOrNil("cba8e95e-bc64-11ee-9324-bb5c09ef083f"),
			},

			`{"id":"cba8e95e-bc64-11ee-9324-bb5c09ef083f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatroomUpdateBasicInfo(req.Context(), &tt.agent, tt.chatroomID, tt.reqBody.Name, tt.reqBody.Detail).Return(tt.responseChat, nil)

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
