package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	chchat "monorepo/bin-chat-manager/models/chat"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_chatsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseChat *chchat.WebhookMessage

		expectName           string
		expectDetail         string
		expectType           chchat.Type
		expectOwnerID        uuid.UUID
		expectParticipantIDs []uuid.UUID
		expectRes            string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats",
			reqBody:  []byte(`{"type":"normal","owner_id":"b044e2a8-38a3-11ed-8427-dfcc31e2715d","participant_ids":["b044e2a8-38a3-11ed-8427-dfcc31e2715d","b93d4e5e-38a3-11ed-ab61-4352c92e28e8"],"name":"test name","detail":"test detail"}`),

			responseChat: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("daddd81c-38a3-11ed-91bd-3b29b0d8de2d"),
				},
			},

			expectName:    "test name",
			expectDetail:  "test detail",
			expectType:    chchat.TypeNormal,
			expectOwnerID: uuid.FromStringOrNil("b044e2a8-38a3-11ed-8427-dfcc31e2715d"),
			expectParticipantIDs: []uuid.UUID{
				uuid.FromStringOrNil("b044e2a8-38a3-11ed-8427-dfcc31e2715d"),
				uuid.FromStringOrNil("b93d4e5e-38a3-11ed-ab61-4352c92e28e8"),
			},
			expectRes: `{"id":"daddd81c-38a3-11ed-91bd-3b29b0d8de2d","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatCreate(
				req.Context(),
				&tt.agent,
				tt.expectType,
				tt.expectOwnerID,
				tt.expectParticipantIDs,
				tt.expectName,
				tt.expectDetail,
			).Return(tt.responseChat, nil)

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

func Test_chatsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChats []*chchat.WebhookMessage

		expectPageSize  uint64
		expectpageToken string
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

			reqQuery: "/chats?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChats: []*chchat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("370254c4-38a4-11ed-b2a5-37ed1141a85e"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectpageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"370254c4-38a4-11ed-b2a5-37ed1141a85e","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChats: []*chchat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("37280f2a-38a4-11ed-86cf-43452b441d43"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("37505e8a-38a4-11ed-b987-bb8a17321104"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3775e664-38a4-11ed-82d4-e30616d2700c"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectpageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"37280f2a-38a4-11ed-86cf-43452b441d43","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"37505e8a-38a4-11ed-b987-bb8a17321104","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"3775e664-38a4-11ed-82d4-e30616d2700c","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ChatGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectpageToken).Return(tt.responseChats, nil)

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

		responseChatmessage *chchat.WebhookMessage

		expectChatID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats/66f4fc5e-38a4-11ed-9898-4fe958a93f32",

			responseChatmessage: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66f4fc5e-38a4-11ed-9898-4fe958a93f32"),
				},
			},

			expectChatID: uuid.FromStringOrNil("66f4fc5e-38a4-11ed-9898-4fe958a93f32"),
			expectRes:    `{"id":"66f4fc5e-38a4-11ed-9898-4fe958a93f32","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatGet(req.Context(), &tt.agent, tt.expectChatID).Return(tt.responseChatmessage, nil)

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

		responseChat *chchat.WebhookMessage

		expectChatID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats/bb04bc12-38a4-11ed-83a7-2f3469ac49d8",

			responseChat: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bb04bc12-38a4-11ed-83a7-2f3469ac49d8"),
				},
			},

			expectChatID: uuid.FromStringOrNil("bb04bc12-38a4-11ed-83a7-2f3469ac49d8"),
			expectRes:    `{"id":"bb04bc12-38a4-11ed-83a7-2f3469ac49d8","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatDelete(req.Context(), &tt.agent, tt.expectChatID).Return(tt.responseChat, nil)

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
		reqBody  []byte

		responseChat *chchat.WebhookMessage

		expectChatID uuid.UUID
		expectName   string
		expectDetail string
		expectRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats/493c3ece-38a5-11ed-a068-3b268867512b",
			reqBody:  []byte(`{"name": "test name", "detail":"test detail"}`),

			responseChat: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("493c3ece-38a5-11ed-a068-3b268867512b"),
				},
			},

			expectChatID: uuid.FromStringOrNil("493c3ece-38a5-11ed-a068-3b268867512b"),
			expectName:   "test name",
			expectDetail: "test detail",
			expectRes:    `{"id":"493c3ece-38a5-11ed-a068-3b268867512b","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatUpdateBasicInfo(req.Context(), &tt.agent, tt.expectChatID, tt.expectName, tt.expectDetail).Return(tt.responseChat, nil)

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
		reqBody  []byte

		responseChat *chchat.WebhookMessage

		expectChatID      uuid.UUID
		expectRoomOwnerID uuid.UUID
		expectRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats/71e7f444-38a5-11ed-ada5-7b100414c281/room_owner_id",
			reqBody:  []byte(`{"room_owner_id":"720f35a4-38a5-11ed-8d75-b7dc3e2589d1"}`),

			responseChat: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("71e7f444-38a5-11ed-ada5-7b100414c281"),
				},
			},

			expectChatID:      uuid.FromStringOrNil("71e7f444-38a5-11ed-ada5-7b100414c281"),
			expectRoomOwnerID: uuid.FromStringOrNil("720f35a4-38a5-11ed-8d75-b7dc3e2589d1"),
			expectRes:         `{"id":"71e7f444-38a5-11ed-ada5-7b100414c281","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatUpdateRoomOwnerID(req.Context(), &tt.agent, tt.expectChatID, tt.expectRoomOwnerID).Return(tt.responseChat, nil)

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
		reqBody  []byte

		responseChat *chchat.WebhookMessage

		expectChatID        uuid.UUID
		expectParticipantID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats/9e543ff6-38a5-11ed-9e73-6f0b324904e7/participant_ids",
			reqBody:  []byte(`{"participant_id":"9e7e032c-38a5-11ed-96c4-0f8ebc10e7f4"}`),

			responseChat: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9e543ff6-38a5-11ed-9e73-6f0b324904e7"),
				},
			},

			expectChatID:        uuid.FromStringOrNil("9e543ff6-38a5-11ed-9e73-6f0b324904e7"),
			expectParticipantID: uuid.FromStringOrNil("9e7e032c-38a5-11ed-96c4-0f8ebc10e7f4"),
			expectRes:           `{"id":"9e543ff6-38a5-11ed-9e73-6f0b324904e7","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatAddParticipantID(req.Context(), &tt.agent, tt.expectChatID, tt.expectParticipantID).Return(tt.responseChat, nil)

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

		reqQuery string

		responseChat *chchat.WebhookMessage

		expectChatID        uuid.UUID
		expectParticipantID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chats/c8a77ab6-38a5-11ed-b33f-137843123dff/participant_ids/c8d869fa-38a5-11ed-9fa4-0ba3841352ee",

			responseChat: &chchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c8a77ab6-38a5-11ed-b33f-137843123dff"),
				},
			},

			expectChatID:        uuid.FromStringOrNil("c8a77ab6-38a5-11ed-b33f-137843123dff"),
			expectParticipantID: uuid.FromStringOrNil("c8d869fa-38a5-11ed-9fa4-0ba3841352ee"),
			expectRes:           `{"id":"c8a77ab6-38a5-11ed-b33f-137843123dff","customer_id":"00000000-0000-0000-0000-000000000000","type":"","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ChatRemoveParticipantID(req.Context(), &tt.agent, tt.expectChatID, tt.expectParticipantID).Return(tt.responseChat, nil)

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
