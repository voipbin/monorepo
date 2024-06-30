package chats

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	chatchat "monorepo/bin-chat-manager/models/chat"

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

func Test_chatsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyChatsPOST

		response *chatchat.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats",
			request.BodyChatsPOST{
				Type:    chatchat.TypeNormal,
				OwnerID: uuid.FromStringOrNil("b044e2a8-38a3-11ed-8427-dfcc31e2715d"),
				ParticipantID: []uuid.UUID{
					uuid.FromStringOrNil("b044e2a8-38a3-11ed-8427-dfcc31e2715d"),
					uuid.FromStringOrNil("b93d4e5e-38a3-11ed-ab61-4352c92e28e8"),
				},
				Name:   "test name",
				Detail: "test detail",
			},

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("daddd81c-38a3-11ed-91bd-3b29b0d8de2d"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatCreate(
				req.Context(),
				&tt.agent,
				tt.reqBody.Type,
				tt.reqBody.OwnerID,
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

func Test_chatsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery  string
		pageSize  uint64
		pageToken string

		responseChats []*chatchat.WebhookMessage
		expectRes     string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatchat.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("370254c4-38a4-11ed-b2a5-37ed1141a85e"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"370254c4-38a4-11ed-b2a5-37ed1141a85e","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*chatchat.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("37280f2a-38a4-11ed-86cf-43452b441d43"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("37505e8a-38a4-11ed-b987-bb8a17321104"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("3775e664-38a4-11ed-82d4-e30616d2700c"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"37280f2a-38a4-11ed-86cf-43452b441d43","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"37505e8a-38a4-11ed-b987-bb8a17321104","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"3775e664-38a4-11ed-82d4-e30616d2700c","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ChatGetsByCustomerID(req.Context(), &tt.agent, tt.pageSize, tt.pageToken).Return(tt.responseChats, nil)

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

func Test_chatsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		chatID   uuid.UUID

		responseChatmessage *chatchat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats/66f4fc5e-38a4-11ed-9898-4fe958a93f32",
			uuid.FromStringOrNil("66f4fc5e-38a4-11ed-9898-4fe958a93f32"),

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("66f4fc5e-38a4-11ed-9898-4fe958a93f32"),
			},

			`{"id":"66f4fc5e-38a4-11ed-9898-4fe958a93f32","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatGet(req.Context(), &tt.agent, tt.chatID).Return(tt.responseChatmessage, nil)

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

func Test_chatsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		chatID   uuid.UUID

		responseChatmessage *chatchat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats/bb04bc12-38a4-11ed-83a7-2f3469ac49d8",
			uuid.FromStringOrNil("bb04bc12-38a4-11ed-83a7-2f3469ac49d8"),

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("bb04bc12-38a4-11ed-83a7-2f3469ac49d8"),
			},

			`{"id":"bb04bc12-38a4-11ed-83a7-2f3469ac49d8","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatDelete(req.Context(), &tt.agent, tt.chatID).Return(tt.responseChatmessage, nil)

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

func Test_chatsIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		chatID   uuid.UUID

		reqBody request.BodyChatsIDPUT

		responseChat *chatchat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats/493c3ece-38a5-11ed-a068-3b268867512b",
			uuid.FromStringOrNil("493c3ece-38a5-11ed-a068-3b268867512b"),

			request.BodyChatsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("493c3ece-38a5-11ed-a068-3b268867512b"),
			},

			`{"id":"493c3ece-38a5-11ed-a068-3b268867512b","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatUpdateBasicInfo(req.Context(), &tt.agent, tt.chatID, tt.reqBody.Name, tt.reqBody.Detail).Return(tt.responseChat, nil)

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

func Test_chatsIDRoomOwnerIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		chatID   uuid.UUID

		reqBody request.BodyChatsIDRoomOwnerIDPUT

		responseChat *chatchat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats/71e7f444-38a5-11ed-ada5-7b100414c281/room_owner_id",
			uuid.FromStringOrNil("71e7f444-38a5-11ed-ada5-7b100414c281"),

			request.BodyChatsIDRoomOwnerIDPUT{
				RoomOwnerID: uuid.FromStringOrNil("720f35a4-38a5-11ed-8d75-b7dc3e2589d1"),
			},

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("71e7f444-38a5-11ed-ada5-7b100414c281"),
			},

			`{"id":"71e7f444-38a5-11ed-ada5-7b100414c281","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatUpdateRoomOwnerID(req.Context(), &tt.agent, tt.chatID, tt.reqBody.RoomOwnerID).Return(tt.responseChat, nil)

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

func Test_chatsIDParticipantIDsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		chatID   uuid.UUID

		reqBody request.BodyChatsIDParticipantIDsPOST

		responseChat *chatchat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats/9e543ff6-38a5-11ed-9e73-6f0b324904e7/participant_ids",
			uuid.FromStringOrNil("9e543ff6-38a5-11ed-9e73-6f0b324904e7"),

			request.BodyChatsIDParticipantIDsPOST{
				ParticipantID: uuid.FromStringOrNil("9e7e032c-38a5-11ed-96c4-0f8ebc10e7f4"),
			},

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("9e543ff6-38a5-11ed-9e73-6f0b324904e7"),
			},

			`{"id":"9e543ff6-38a5-11ed-9e73-6f0b324904e7","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatAddParticipantID(req.Context(), &tt.agent, tt.chatID, tt.reqBody.ParticipantID).Return(tt.responseChat, nil)

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

func Test_chatsIDParticipantIDsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		chatID        uuid.UUID
		participantID uuid.UUID

		responseChat *chatchat.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chats/c8a77ab6-38a5-11ed-b33f-137843123dff/participant_ids/c8d869fa-38a5-11ed-9fa4-0ba3841352ee",
			uuid.FromStringOrNil("c8a77ab6-38a5-11ed-b33f-137843123dff"),
			uuid.FromStringOrNil("c8d869fa-38a5-11ed-9fa4-0ba3841352ee"),

			&chatchat.WebhookMessage{
				ID: uuid.FromStringOrNil("c8a77ab6-38a5-11ed-b33f-137843123dff"),
			},

			`{"id":"c8a77ab6-38a5-11ed-b33f-137843123dff","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatRemoveParticipantID(req.Context(), &tt.agent, tt.chatID, tt.participantID).Return(tt.responseChat, nil)

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
