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

func Test_chatroomsPOST(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/chatrooms",
			reqBody:  []byte(`{"participant_ids":["2a2ec0ba-8004-11ec-aea5-439829c92a7c","a5d27d76-bc05-11ee-b3db-0333d9946bda"], "name":"test name","detail":"test detail"}`),

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a5ffbb60-bc05-11ee-af79-e73be92a78fe"),
				},
			},

			expectParticipantIDs: []uuid.UUID{
				uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				uuid.FromStringOrNil("a5d27d76-bc05-11ee-b3db-0333d9946bda"),
			},
			expectName:   "test name",
			expectDetail: "test detail",
			expectRes:    `{"id":"a5ffbb60-bc05-11ee-af79-e73be92a78fe","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroomCreate(
				req.Context(),
				&tt.agent,
				tt.expectParticipantIDs,
				tt.expectName,
				tt.expectDetail,
			.Return(tt.responseChatroom, nil)

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

func Test_chatroomsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseChatrooms []*chchatroom.WebhookMessage

		expectOwnerID   uuid.UUID
		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "all items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("65c83e84-8df5-11ee-ba8b-1700dcdfa8f2"),
				},
			},

			reqQuery: "/chatrooms?owner_id=f3974fc6-38a0-11ed-a40b-6fc6c6ec606e&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatrooms: []*chchatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f4037f8e-38a0-11ed-8424-5f0d424074aa"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectOwnerID:   uuid.FromStringOrNil("f3974fc6-38a0-11ed-a40b-6fc6c6ec606e"),
			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"f4037f8e-38a0-11ed-8424-5f0d424074aa","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("65c83e84-8df5-11ee-ba8b-1700dcdfa8f2"),
				},
			},

			reqQuery: "/chatrooms?owner_id=20c40fac-38a1-11ed-83d3-93d4a1e51688&page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseChatrooms: []*chchatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("20f10a66-38a1-11ed-bd54-d7b834668361"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("211bfb54-38a1-11ed-8d88-f34522ed8844"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("21454f0e-38a1-11ed-bce4-a7b8972af690"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectOwnerID:   uuid.FromStringOrNil("20c40fac-38a1-11ed-83d3-93d4a1e51688"),
			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"20f10a66-38a1-11ed-bd54-d7b834668361","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"211bfb54-38a1-11ed-8d88-f34522ed8844","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"21454f0e-38a1-11ed-bce4-a7b8972af690","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
		{
			name: "empty item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("65c83e84-8df5-11ee-ba8b-1700dcdfa8f2"),
				},
			},

			reqQuery: "/chatrooms",

			responseChatrooms: []*chchatroom.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("16fa73ec-dbc6-11ef-9c58-bbaa2ba3956b"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectOwnerID:   uuid.FromStringOrNil("65c83e84-8df5-11ee-ba8b-1700dcdfa8f2"),
			expectPageSize:  100,
			expectPageToken: "",
			expectRes:       `{"result":[{"id":"16fa73ec-dbc6-11ef-9c58-bbaa2ba3956b","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		}}

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

			mockSvc.EXPECT().ChatroomGetsByOwnerID(req.Context(), &tt.agent, tt.expectOwnerID, tt.expectPageSize, tt.expectPageToken.Return(tt.responseChatrooms, nil)

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

			reqQuery: "/chatrooms/5585fce6-38a1-11ed-93f2-c396374122fd",

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5585fce6-38a1-11ed-93f2-c396374122fd"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("5585fce6-38a1-11ed-93f2-c396374122fd"),
			expectRes:        `{"id":"5585fce6-38a1-11ed-93f2-c396374122fd","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroomGet(req.Context(), &tt.agent, tt.expectChatroomID.Return(tt.responseChatroom, nil)

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

			reqQuery: "/chatrooms/98799008-38a1-11ed-9406-6fa22402722c",

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("98799008-38a1-11ed-9406-6fa22402722c"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("98799008-38a1-11ed-9406-6fa22402722c"),
			expectRes:        `{"id":"98799008-38a1-11ed-9406-6fa22402722c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroomDelete(req.Context(), &tt.agent, tt.expectChatroomID.Return(tt.responseChatroom, nil)

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

			reqQuery: "/chatrooms/cba8e95e-bc64-11ee-9324-bb5c09ef083f",
			reqBody:  []byte(`{"name":"test name","detail":"test detail"}`),

			responseChatroom: &chchatroom.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cba8e95e-bc64-11ee-9324-bb5c09ef083f"),
				},
			},

			expectChatroomID: uuid.FromStringOrNil("cba8e95e-bc64-11ee-9324-bb5c09ef083f"),
			expectName:       "test name",
			expectDetail:     "test detail",
			expectRes:        `{"id":"cba8e95e-bc64-11ee-9324-bb5c09ef083f","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","type":"","chat_id":"00000000-0000-0000-0000-000000000000","room_owner_id":"00000000-0000-0000-0000-000000000000","participant_ids":null,"name":"","detail":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ChatroomUpdateBasicInfo(req.Context(), &tt.agent, tt.expectChatroomID, tt.expectName, tt.expectDetail.Return(tt.responseChatroom, nil)

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
