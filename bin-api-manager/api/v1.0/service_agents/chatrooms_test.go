package service_agents

import (
	"bytes"
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_chatroomsGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery     string
		resChatrooms []*chatchatroom.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/chatrooms?page_token=2020-09-20%2003:23:20.995000&page_size=10",

			resChatrooms: []*chatchatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b7e14a4a-3bb9-11ef-bb03-3b706018eb7c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b8588a06-3bb9-11ef-81c3-43bb60ab1c00"),
					},
				},
			},

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"b7e14a4a-3bb9-11ef-bb03-3b706018eb7c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"b8588a06-3bb9-11ef-81c3-43bb60ab1c00","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			mockSvc.EXPECT().ServiceAgentChatroomGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.resChatrooms, nil)

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

func Test_chatroomsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyServiceAgentChatroomsPOST

		response *chatchatroom.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("35c9371e-3bbb-11ef-9ddd-93d954c59556"),
				},
			},

			"/v1.0/service_agents/chatrooms",
			request.BodyServiceAgentChatroomsPOST{
				ParticipantID: []uuid.UUID{
					uuid.FromStringOrNil("35c9371e-3bbb-11ef-9ddd-93d954c59556"),
					uuid.FromStringOrNil("3643d230-3bbb-11ef-a119-0b4c2a13ead0"),
				},
				Name:   "test name",
				Detail: "test detail",
			},

			&chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("367e5888-3bbb-11ef-a60b-0fecf1ef915a"),
				},
			},
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
			req.Header.Set("Content-type", "application/json")

			mockSvc.EXPECT().ServiceAgentChatroomCreate(
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

func Test_chatroomsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatroom *chatchatroom.WebhookMessage

		expectChatroomID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/chatrooms/100d4c94-3bbc-11ef-bfc4-0743f08f4c60",

			responseChatroom: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("100d4c94-3bbc-11ef-bfc4-0743f08f4c60"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("100d4c94-3bbc-11ef-bfc4-0743f08f4c60"),
			expectRes:        `{"id":"100d4c94-3bbc-11ef-bfc4-0743f08f4c60","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ServiceAgentChatroomGet(req.Context(), &tt.agent, tt.expectChatroomID).Return(tt.responseChatroom, nil)

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/service_agents/chatrooms/5b0bda76-3bbc-11ef-bf6d-a33053327539",
			uuid.FromStringOrNil("5b0bda76-3bbc-11ef-bf6d-a33053327539"),

			&chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5b0bda76-3bbc-11ef-bf6d-a33053327539"),
				},
			},

			`{"id":"5b0bda76-3bbc-11ef-bf6d-a33053327539","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentChatroomDelete(req.Context(), &tt.customer, tt.chatroomID).Return(tt.responseChatroom, nil)

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

		reqQuery string
		reqBody  request.BodyChatsIDPUT

		responseChat *chatchatroom.WebhookMessage

		expectChatroomID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/chatrooms/7ccc0438-3bbc-11ef-8096-ef2cd860ae7f",
			reqBody: request.BodyChatsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},

			responseChat: &chatchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7ccc0438-3bbc-11ef-8096-ef2cd860ae7f"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("7ccc0438-3bbc-11ef-8096-ef2cd860ae7f"),
			expectRes:        `{"id":"7ccc0438-3bbc-11ef-8096-ef2cd860ae7f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentChatroomUpdateBasicInfo(req.Context(), &tt.agent, tt.expectChatroomID, tt.reqBody.Name, tt.reqBody.Detail).Return(tt.responseChat, nil)

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
