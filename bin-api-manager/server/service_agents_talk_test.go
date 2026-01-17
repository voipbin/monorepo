package server

import (
	"bytes"
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_talksGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTalks []*tkchat.WebhookMessage

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

			reqQuery: "/service_agents/talk_chats?page_token=2020-09-20%2003:23:20.995000&page_size=10",

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

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"83d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"direct"},{"id":"84caa752-3ed7-11ef-a428-7bbe6c050b77","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"group"}],"next_page_token":""}`,
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
			mockSvc.EXPECT().ServiceAgentTalkList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseTalks, nil)

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
		agent amagent.Agent

		reqQuery string

		responseTalk *tkchat.WebhookMessage
		expectTalkID uuid.UUID
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/service_agents/talk_chats/e66d1da0-3ed7-11ef-9208-4bcc069917a1",

			responseTalk: &tkchat.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type:     tkchat.TypeDirect,
				TMCreate: "2020-09-20T03:23:21.995000",
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkGet(req.Context(), &tt.agent, tt.expectTalkID).Return(tt.responseTalk, nil)
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
		agent amagent.Agent

		reqQuery string
		reqBody  string

		responseTalk *tkchat.WebhookMessage

		expectType tkchat.Type
		expectRes  string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
			},

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkCreate(req.Context(), &tt.agent, tt.expectType).Return(tt.responseTalk, nil)

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
		agent amagent.Agent

		reqQuery string

		responseTalk *tkchat.WebhookMessage
		expectTalkID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkDelete(req.Context(), &tt.agent, tt.expectTalkID).Return(tt.responseTalk, nil)
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
func Test_talksIDParticipantsGET(t *testing.T) {
	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseParticipants []*tkparticipant.WebhookMessage
		expectTalkID         uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
					TMJoined: "2020-09-20T03:23:21.995000",
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkParticipantList(req.Context(), &tt.agent, tt.expectTalkID).Return(tt.responseParticipants, nil)
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
		agent amagent.Agent

		reqQuery string
		reqBody  string

		responseParticipant *tkparticipant.WebhookMessage

		expectTalkID    uuid.UUID
		expectOwnerType string
		expectOwnerID   uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkParticipantCreate(req.Context(), &tt.agent, tt.expectTalkID, tt.expectOwnerType, tt.expectOwnerID).Return(tt.responseParticipant, nil)

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
		agent amagent.Agent

		reqQuery string

		responseParticipant  *tkparticipant.WebhookMessage
		expectTalkID         uuid.UUID
		expectParticipantID  uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkParticipantDelete(req.Context(), &tt.agent, tt.expectTalkID, tt.expectParticipantID).Return(tt.responseParticipant, nil)
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
		agent amagent.Agent

		reqQuery string

		responseMessages []*tkmessage.WebhookMessage

		expectChatID    uuid.UUID
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

			reqQuery: "/service_agents/talk_messages?chat_id=e66d1da0-3ed7-11ef-9208-4bcc069917a1&page_token=2020-09-20%2003:23:20.995000&page_size=10",

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
		expectChatID:    uuid.FromStringOrNil("e66d1da0-3ed7-11ef-9208-4bcc069917a1"),

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"93d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","type":"normal","text":"Hello","medias":null,"metadata":{"reactions":null},"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			mockSvc.EXPECT().ServiceAgentTalkMessageList(req.Context(), &tt.agent, tt.expectChatID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseMessages, nil)

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
		agent amagent.Agent

		reqQuery string
		reqBody  string

		responseMessage *tkmessage.WebhookMessage

		expectChatID   uuid.UUID
		expectParentID *uuid.UUID
		expectType     tkmessage.Type
		expectText     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
			},

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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkMessageCreate(req.Context(), &tt.agent, tt.expectChatID, tt.expectParentID, tt.expectType, tt.expectText).Return(tt.responseMessage, nil)

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
		agent amagent.Agent

		reqQuery string

		responseMessage *tkmessage.WebhookMessage
		expectMessageID uuid.UUID
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
				TMCreate: "2020-09-20T03:23:21.995000",
			},

			expectMessageID: uuid.FromStringOrNil("93d48228-3ed7-11ef-a9ca-070e7ba46a55"),
			expectRes:       `{"id":"93d48228-3ed7-11ef-a9ca-070e7ba46a55","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"cdb5213a-8003-11ec-84ca-9fa226fcda9f","chat_id":"e66d1da0-3ed7-11ef-9208-4bcc069917a1","type":"normal","text":"Hello","medias":null,"metadata":{"reactions":null},"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().ServiceAgentTalkMessageGet(req.Context(), &tt.agent, tt.expectMessageID).Return(tt.responseMessage, nil)
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
		agent amagent.Agent

		reqQuery string

		responseMessage *tkmessage.WebhookMessage
		expectMessageID uuid.UUID
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)

			mockSvc.EXPECT().ServiceAgentTalkMessageDelete(req.Context(), &tt.agent, tt.expectMessageID).Return(tt.responseMessage, nil)
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
		agent amagent.Agent

		reqQuery string
		reqBody  string

		responseMessage *tkmessage.WebhookMessage

		expectMessageID uuid.UUID
		expectEmoji     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentTalkMessageReactionCreate(req.Context(), &tt.agent, tt.expectMessageID, tt.expectEmoji).Return(tt.responseMessage, nil)

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
