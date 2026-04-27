package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_talksGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseTalks []*tkchat.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/talk_chats?page_token=2020-09-20T03:23:20.995000Z&page_size=10",

			responseTalks: []*tkchat.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("83d48228-3ed7-11ef-a9ca-070e7ba46a55"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Type: tkchat.TypeDirect,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("84caa752-3ed7-11ef-a428-7bbe6c050b77"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Type: tkchat.TypeGroup,
				},
			},

			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"83d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"direct","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"84caa752-3ed7-11ef-a428-7bbe6c050b77","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"group","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentTalkChatList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseTalks, nil)

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

func Test_talksIDGET(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseTalk *tkchat.WebhookMessage
		expectTalkID uuid.UUID
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1",

			responseTalk: &tkchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type:     tkchat.TypeDirect,
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectTalkID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectRes:    `{"id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"direct"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkChatGet(req.Context(), tt.agent, tt.expectTalkID).Return(tt.responseTalk, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseTalk)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talksPOST(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseTalk *tkchat.WebhookMessage

		expectType tkchat.Type
		expectRes  string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
			}),

			reqQuery: "/service_agents/talk_chats",
			reqBody:  `{"type":"direct"}`,

			responseTalk: &tkchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("83d48228-3ed7-11ef-a9ca-070e7ba46a55"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type: tkchat.TypeDirect,
			},

			expectType: tkchat.TypeDirect,
			expectRes:  `{"id":"83d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"direct"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkChatCreate(req.Context(), tt.agent, tt.expectType, "", "", gomock.Any()).Return(tt.responseTalk, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseTalk)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talksIDDELETE(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseTalk *tkchat.WebhookMessage
		expectTalkID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1",

			responseTalk: &tkchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type: tkchat.TypeDirect,
			},

			expectTalkID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkChatDelete(req.Context(), tt.agent, tt.expectTalkID).Return(tt.responseTalk, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseTalk)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talksIDPUT(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseTalk *tkchat.WebhookMessage

		expectTalkID uuid.UUID
		expectName   *string
		expectDetail *string
	}{
		{
			name: "update name only",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			reqBody:  `{"name":"Updated Name"}`,

			responseTalk: &tkchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type: tkchat.TypeDirect,
				Name: "Updated Name",
			},

			expectTalkID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectName:   ptrString("Updated Name"),
			expectDetail: nil,
		},
		{
			name: "update both name and detail",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1",
			reqBody:  `{"name":"New Name","detail":"New Detail"}`,

			responseTalk: &tkchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type:   tkchat.TypeDirect,
				Name:   "New Name",
				Detail: "New Detail",
			},

			expectTalkID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectName:   ptrString("New Name"),
			expectDetail: ptrString("New Detail"),
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkChatUpdate(req.Context(), tt.agent, tt.expectTalkID, tt.expectName, tt.expectDetail).Return(tt.responseTalk, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseTalk)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func ptrString(s string) *string {
	return &s
}

func Test_talksIDParticipantsGET(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseParticipants []*tkparticipant.WebhookMessage
		expectTalkID         uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1/participants",

			responseParticipants: []*tkparticipant.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
					},
					ChatID:   uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					TMJoined: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectTalkID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkParticipantList(req.Context(), tt.agent, tt.expectTalkID).Return(tt.responseParticipants, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseParticipants)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talksIDParticipantsPOST(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseParticipant *tkparticipant.WebhookMessage

		expectTalkID    uuid.UUID
		expectOwnerType string
		expectOwnerID   uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1/participants",
			reqBody:  `{"owner_type":"agent","owner_id":"cdb5213a-8003-11ec-84ca-9fa226fcda9f"}`,

			responseParticipant: &tkparticipant.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			},

			expectTalkID:    uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectOwnerType: "agent",
			expectOwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkParticipantCreate(req.Context(), tt.agent, tt.expectTalkID, tt.expectOwnerType, tt.expectOwnerID).Return(tt.responseParticipant, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseParticipant)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talksIDParticipantsIDDELETE(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseParticipant *tkparticipant.WebhookMessage
		expectTalkID        uuid.UUID
		expectParticipantID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1/participants/f66d1da0-3ed7-11ef-9208-4bcc069917a2",

			responseParticipant: &tkparticipant.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			},

			expectTalkID:        uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectParticipantID: uuid.FromStringOrNil("f66d1da0-3ed7-11ef-9208-4bcc069917a2"),
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkParticipantDelete(req.Context(), tt.agent, tt.expectTalkID, tt.expectParticipantID).Return(tt.responseParticipant, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseParticipant)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talkMessagesGET(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseMessages []*tkmessage.WebhookMessage

		expectChatID    uuid.UUID
		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/talk_messages?chat_id=e66d1da0-3ed7-11ef-9208-4bcc069917a1&page_token=2020-09-20T03:23:20.995000Z&page_size=10",

			responseMessages: []*tkmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					},
					ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					Type:   tkmessage.TypeNormal,
					Text:   "Hello",
				},
			},
			expectChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),

			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"93d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","type":"normal","text":"Hello","medias":null,"metadata":{"reactions":null},"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentTalkMessageList(req.Context(), tt.agent, tt.expectChatID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseMessages, nil)

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

func Test_talkMessagesPOST(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseMessage *tkmessage.WebhookMessage

		expectChatID   uuid.UUID
		expectParentID *uuid.UUID
		expectType     tkmessage.Type
		expectText     string
		expectMedias   []tkmessage.Media
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
			}),

			reqQuery: "/service_agents/talk_messages",
			reqBody:  `{"chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","type":"normal","text":"Hello"}`,

			responseMessage: &tkmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
				Type:   tkmessage.TypeNormal,
				Text:   "Hello",
			},

			expectChatID:   uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
			expectParentID: nil,
			expectType:     tkmessage.TypeNormal,
			expectText:     "Hello",
			expectMedias:   nil,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkMessageCreate(req.Context(), tt.agent, tt.expectChatID, tt.expectParentID, tt.expectType, tt.expectText, tt.expectMedias).Return(tt.responseMessage, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseMessage)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talkMessagesIDGET(t *testing.T) {
	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseMessage *tkmessage.WebhookMessage
		expectMessageID uuid.UUID
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_messages/93d48228-3ed7-11ef-a9ca-070e7ba46a55",

			responseMessage: &tkmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID:   uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
				Type:     tkmessage.TypeNormal,
				Text:     "Hello",
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectMessageID: uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
			expectRes:       `{"id":"93d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"cdb5213a-8003-11ec-84ca-9fa226fcda9f","chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","type":"normal","text":"Hello","medias":null,"metadata":{"reactions":null},"tm_create":"2020-09-20T03:23:21.995000Z","tm_update":"","tm_delete":""}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkMessageGet(req.Context(), tt.agent, tt.expectMessageID).Return(tt.responseMessage, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseMessage)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talkMessagesIDDELETE(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseMessage *tkmessage.WebhookMessage
		expectMessageID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_messages/93d48228-3ed7-11ef-a9ca-070e7ba46a55",

			responseMessage: &tkmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
				Type:   tkmessage.TypeNormal,
				Text:   "Hello",
			},

			expectMessageID: uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkMessageDelete(req.Context(), tt.agent, tt.expectMessageID).Return(tt.responseMessage, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseMessage)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_talkMessagesIDReactionsPOST(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  string

		responseMessage *tkmessage.WebhookMessage

		expectMessageID uuid.UUID
		expectEmoji     string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			}),

			reqQuery: "/service_agents/talk_messages/93d48228-3ed7-11ef-a9ca-070e7ba46a55/reactions",
			reqBody:  `{"emoji":"👍"}`,

			responseMessage: &tkmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				ChatID: uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
				Type:   tkmessage.TypeNormal,
				Text:   "Hello",
			},

			expectMessageID: uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
			expectEmoji:     "👍",
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkMessageReactionCreate(req.Context(), tt.agent, tt.expectMessageID, tt.expectEmoji).Return(tt.responseMessage, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.responseMessage)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

// Test_serviceAgentsTalkChatsPost_MissingAuthIdentity verifies
// PostServiceAgentsTalkChats emits the canonical UNAUTHENTICATED /
// AUTHENTICATION_REQUIRED envelope when auth_identity is missing from
// the gin context.
func Test_serviceAgentsTalkChatsPost_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPost, "/service_agents/talk_chats",
		[]byte(`{"type":"direct"}`))
}

// Test_serviceAgentsTalkChatsPost_InvalidJSONBody verifies
// PostServiceAgentsTalkChats rejects a malformed JSON body with
// INVALID_ARGUMENT / INVALID_JSON_BODY before the servicehandler is
// consulted.
func Test_serviceAgentsTalkChatsPost_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodPost, "/service_agents/talk_chats", bytes.NewBufferString(`{not-json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY", commonoutline.ServiceNameAPIManager)
}

// Test_serviceAgentsTalkChatsIDPut_InvalidID verifies
// PutServiceAgentsTalkChatsId rejects a malformed UUID in the path with
// INVALID_ARGUMENT / INVALID_ID before the servicehandler is consulted.
func Test_serviceAgentsTalkChatsIDPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodPut, "/service_agents/talk_chats/not-a-uuid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID", commonoutline.ServiceNameAPIManager)
}

// Test_serviceAgentsTalkChatsIDParticipantsParticipantIDDelete_InvalidParticipantID
// verifies DeleteServiceAgentsTalkChatsIdParticipantsParticipantId
// returns INVALID_ARGUMENT / INVALID_ID when the parent chat id is a
// valid UUID but the nested participant_id is malformed. Exercises the
// dual-ID validation path with a distinguishing message.
func Test_serviceAgentsTalkChatsIDParticipantsParticipantIDDelete_InvalidParticipantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodDelete, "/service_agents/talk_chats/83d48228-3ed7-11ef-a9ca-070e7ba46a55/participants/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID", commonoutline.ServiceNameAPIManager)
}

// Test_serviceAgentsTalkMessagesPost_MissingAuthIdentity verifies
// PostServiceAgentsTalkMessages emits the canonical UNAUTHENTICATED /
// AUTHENTICATION_REQUIRED envelope when auth_identity is missing from
// the gin context.
func Test_serviceAgentsTalkMessagesPost_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPost, "/service_agents/talk_messages",
		[]byte(`{"chat_id":"83d48228-3ed7-11ef-a9ca-070e7ba46a55","type":"normal","text":"hello"}`))
}
