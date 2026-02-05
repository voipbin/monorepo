package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	cvaccount "monorepo/bin-conversation-manager/models/account"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_conversationAccountsGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAccounts []*cvaccount.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},

			reqQuery: "/conversation_accounts?page_size=20&page_token=2020-09-20T03:23:20.995000Z",

			responseAccounts: []*cvaccount.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6adce0da-004e-11ee-b74a-23da476139db"),
					},
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"6adce0da-004e-11ee-b74a-23da476139db","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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

			mockSvc.EXPECT().ConversationAccountGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseAccounts, nil)

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

func Test_conversationAccountsPost(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAccount *cvaccount.WebhookMessage

		expectType   cvaccount.Type
		expectName   string
		expectDetail string
		expectSecret string
		expectToken  string
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},

			reqQuery: "/conversation_accounts",
			reqBody:  []byte(`{"type":"line","name":"test name","detail":"test detail","secret":"test secret","token":"test token"}`),

			responseAccount: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6cc1b186-004f-11ee-91df-7f283f71f97a"),
				},
			},

			expectType:   cvaccount.TypeLine,
			expectName:   "test name",
			expectDetail: "test detail",
			expectSecret: "test secret",
			expectToken:  "test token",
			expectRes:    `{"id":"6cc1b186-004f-11ee-91df-7f283f71f97a","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ConversationAccountCreate(req.Context(), &tt.agent, tt.expectType, tt.expectName, tt.expectDetail, tt.expectSecret, tt.expectToken).Return(tt.responseAccount, nil)

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

func Test_conversationAccountsIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseConversationAccount *cvaccount.WebhookMessage

		expectConversationAccountID uuid.UUID
		expectRes                   string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab2f092e-004e-11ee-b834-b7077f22c1eb"),
				},
			},

			reqQuery: "/conversation_accounts/ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c",

			responseConversationAccount: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c"),
				},
			},

			expectConversationAccountID: uuid.FromStringOrNil("ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c"),
			expectRes:                   `{"id":"ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ConversationAccountGet(req.Context(), &tt.agent, tt.expectConversationAccountID).Return(tt.responseConversationAccount, nil)

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

func Test_conversationAccountsIDPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseConversationAccount *cvaccount.WebhookMessage

		expectConversationAccountID uuid.UUID
		expectFields                map[cvaccount.Field]any
		expectRes                   string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},

			reqQuery: "/conversation_accounts/009f2ac8-0050-11ee-b416-5f4fb9c7c682",
			reqBody:  []byte(`{"name":"test name","secret":"test secret","token":"test token"}`),

			responseConversationAccount: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("009f2ac8-0050-11ee-b416-5f4fb9c7c682"),
				},
			},

			expectConversationAccountID: uuid.FromStringOrNil("009f2ac8-0050-11ee-b416-5f4fb9c7c682"),
			expectFields: map[cvaccount.Field]any{
				cvaccount.FieldName:   "test name",
				cvaccount.FieldSecret: "test secret",
				cvaccount.FieldToken:  "test token",
			},
			expectRes: `{"id":"009f2ac8-0050-11ee-b416-5f4fb9c7c682","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().ConversationAccountUpdate(req.Context(), &tt.agent, tt.expectConversationAccountID, tt.expectFields).Return(tt.responseConversationAccount, nil)

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

func Test_conversationAccountsIDDelete(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseConversationAccount *cvaccount.WebhookMessage

		expectConversationAccountID uuid.UUID
		expectRes                   string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},

			reqQuery: "/conversation_accounts/31a54f8a-0050-11ee-aa7e-d3a80a493b8b",

			responseConversationAccount: &cvaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31a54f8a-0050-11ee-aa7e-d3a80a493b8b"),
				},
			},

			expectConversationAccountID: uuid.FromStringOrNil("31a54f8a-0050-11ee-aa7e-d3a80a493b8b"),
			expectRes:                   `{"id":"31a54f8a-0050-11ee-aa7e-d3a80a493b8b","customer_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationAccountDelete(req.Context(), &tt.agent, tt.expectConversationAccountID).Return(tt.responseConversationAccount, nil)

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
