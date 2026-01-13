package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_conversationsGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseConversations []*cvconversation.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
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

			reqQuery: "/conversations?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",

			responseConversations: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("120bc6da-ed2e-11ec-839d-cb324c315bf3"),
					},
				},
			},
			expectRes: `{"result":[{"id":"120bc6da-ed2e-11ec-839d-cb324c315bf3","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}],"next_page_token":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken.Return(tt.responseConversations, nil)

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

func Test_conversationsIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseConversation *cvconversation.WebhookMessage

		expectConversationID uuid.UUID
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/conversations/4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d",

			responseConversation: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d"),
				},
			},

			expectConversationID: uuid.FromStringOrNil("4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d"),
			expectRes:            `{"id":"4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}`,
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

			// create request
			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationGet(req.Context(), &tt.agent, tt.expectConversationID.Return(tt.responseConversation, nil)

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

func Test_GetConversationsIdMessages(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseMessages []*cvmessage.WebhookMessage

		expectConversationID uuid.UUID
		expectPageSize       uint64
		expectPageToken      string
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/conversations/a09b01e0-ed2e-11ec-bdf1-8fa58d1092ad/messages?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseMessages: []*cvmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("120bc6da-ed2e-11ec-839d-cb324c315bf3"),
					},
				},
			},

			expectConversationID: uuid.FromStringOrNil("a09b01e0-ed2e-11ec-bdf1-8fa58d1092ad"),
			expectPageSize:       20,
			expectPageToken:      "2020-09-20 03:23:20.995000",
			expectRes:            `{"result":[{"id":"120bc6da-ed2e-11ec-839d-cb324c315bf3","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}],"next_page_token":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationMessageGetsByConversationID(req.Context(), &tt.agent, tt.expectConversationID, tt.expectPageSize, tt.expectPageToken.Return(tt.responseMessages, nil)

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

func Test_PostConversationsIdMessages(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseMessage *cvmessage.WebhookMessage

		expectConversationID uuid.UUID
		expectText           string
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages",
			reqBody:  []byte(`{"text":"hello world."}`),

			responseMessage: &cvmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44757534-ed2f-11ec-b41b-b36583f1d5a7"),
				},
			},

			expectConversationID: uuid.FromStringOrNil("5950b02c-ed2f-11ec-9093-d3dcc91a72fa"),
			expectText:           "hello world.",
			expectRes:            `{"id":"44757534-ed2f-11ec-b41b-b36583f1d5a7","customer_id":"00000000-0000-0000-0000-000000000000","conversation_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`,
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

			mockSvc.EXPECT().ConversationMessageSend(req.Context(), &tt.agent, tt.expectConversationID, tt.expectText, []cvmedia.Media{}.Return(tt.responseMessage, nil)

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

func Test_conversationsIDPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseConversation *cvconversation.WebhookMessage

		expectConversationID uuid.UUID
		expectFields         map[cvconversation.Field]any
		expectRes            string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/conversations/0e288b58-007d-11ee-b0ac-8be49d249ca9",
			reqBody:  []byte(`{"name":"test name","detail":"test detail"}`),

			responseConversation: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0e288b58-007d-11ee-b0ac-8be49d249ca9"),
				},
			},

			expectConversationID: uuid.FromStringOrNil("0e288b58-007d-11ee-b0ac-8be49d249ca9"),
			expectFields: map[cvconversation.Field]any{
				cvconversation.FieldName:   "test name",
				cvconversation.FieldDetail: "test detail",
			},
			expectRes: `{"id":"0e288b58-007d-11ee-b0ac-8be49d249ca9","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","self":{},"peer":{}}`,
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

			mockSvc.EXPECT().ConversationUpdate(req.Context(), &tt.agent, tt.expectConversationID, tt.expectFields.Return(tt.responseConversation, nil)

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
