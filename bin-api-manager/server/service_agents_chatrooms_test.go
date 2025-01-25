package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	chchatroom "monorepo/bin-chat-manager/models/chatroom"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsChatrooms(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatrooms []*chchatroom.WebhookMessage

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

			reqQuery: "/service_agents/chatrooms?page_token=2020-09-20%2003:23:20.995000&page_size=10",

			responseChatrooms: []*chchatroom.WebhookMessage{
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
			mockSvc.EXPECT().ServiceAgentChatroomGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseChatrooms, nil)

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

func Test_PostServiceAgentsChatrooms(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseChatroom *chchatroom.WebhookMessage

		expectParticipantIDs []uuid.UUID
		expectName           string
		expectDetail         string
		expectRes            string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("35c9371e-3bbb-11ef-9ddd-93d954c59556"),
				},
			},

			reqQuery: "/service_agents/chatrooms",
			reqBody:  []byte(`{"participant_ids":["35c9371e-3bbb-11ef-9ddd-93d954c59556","3643d230-3bbb-11ef-a119-0b4c2a13ead0"],"name":"test name","detail":"test detail"}`),

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("367e5888-3bbb-11ef-a60b-0fecf1ef915a"),
				},
			},

			expectParticipantIDs: []uuid.UUID{
				uuid.FromStringOrNil("35c9371e-3bbb-11ef-9ddd-93d954c59556"),
				uuid.FromStringOrNil("3643d230-3bbb-11ef-a119-0b4c2a13ead0"),
			},
			expectName:   "test name",
			expectDetail: "test detail",
			expectRes:    `{"id":"367e5888-3bbb-11ef-a60b-0fecf1ef915a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockSvc.EXPECT().ServiceAgentChatroomCreate(
				req.Context(),
				&tt.agent,
				tt.expectParticipantIDs,
				tt.expectName,
				tt.expectDetail,
			).Return(tt.responseChatroom, nil)

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

func Test_GetServiceAgentsChatroomsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatroom *chchatroom.WebhookMessage

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

			reqQuery: "/service_agents/chatrooms/100d4c94-3bbc-11ef-bfc4-0743f08f4c60",

			responseChatroom: &chchatroom.WebhookMessage{
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

func Test_DeleteServiceAgentsChatroomsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatroom *chchatroom.WebhookMessage

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

			reqQuery: "/service_agents/chatrooms/5b0bda76-3bbc-11ef-bf6d-a33053327539",

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5b0bda76-3bbc-11ef-bf6d-a33053327539"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("5b0bda76-3bbc-11ef-bf6d-a33053327539"),
			expectRes:        `{"id":"5b0bda76-3bbc-11ef-bf6d-a33053327539","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockSvc.EXPECT().ServiceAgentChatroomDelete(req.Context(), &tt.agent, tt.expectChatroomID).Return(tt.responseChatroom, nil)

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

func Test_PutServiceAgentsChatroomsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseChatroom *chchatroom.WebhookMessage

		expectChatroomID uuid.UUID
		expectName       string
		expectDetail     string
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/service_agents/chatrooms/7ccc0438-3bbc-11ef-8096-ef2cd860ae7f",
			reqBody:  []byte(`{"name":"test name","detail":"test detail"}`),

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7ccc0438-3bbc-11ef-8096-ef2cd860ae7f"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("7ccc0438-3bbc-11ef-8096-ef2cd860ae7f"),
			expectName:       "test name",
			expectDetail:     "test detail",
			expectRes:        `{"id":"7ccc0438-3bbc-11ef-8096-ef2cd860ae7f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentChatroomUpdateBasicInfo(req.Context(), &tt.agent, tt.expectChatroomID, tt.expectName, tt.expectDetail).Return(tt.responseChatroom, nil)

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
