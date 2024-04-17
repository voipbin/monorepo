package conversationaccounts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cvaccount "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_conversationAccountsGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		size  uint64
		token string

		responseAccounts []*cvaccount.WebhookMessage
		expectRes        *response.BodyConversationAccountsGET
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
			},
			"/v1.0/conversation_accounts?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			20,
			"2020-09-20 03:23:20.995000",

			[]*cvaccount.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6adce0da-004e-11ee-b74a-23da476139db"),
				},
			},
			&response.BodyConversationAccountsGET{
				Result: []*cvaccount.WebhookMessage{
					{
						ID: uuid.FromStringOrNil("6adce0da-004e-11ee-b74a-23da476139db"),
					},
				},
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

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationAccountGetsByCustomerID(req.Context(), &tt.agent, tt.size, tt.token).Return(tt.responseAccounts, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationAccountsPost(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		req request.BodyConversationAccountsPOST

		responseAccount *cvaccount.WebhookMessage
		expectRes       *cvaccount.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
			},
			target: "/v1.0/conversation_accounts",

			req: request.BodyConversationAccountsPOST{
				Type:   cvaccount.TypeLine,
				Name:   "test name",
				Detail: "test detail",
				Secret: "test secret",
				Token:  "test token",
			},

			responseAccount: &cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("6cc1b186-004f-11ee-91df-7f283f71f97a"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("6cc1b186-004f-11ee-91df-7f283f71f97a"),
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

			// create request
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationAccountCreate(req.Context(), &tt.agent, tt.req.Type, tt.req.Name, tt.req.Detail, tt.req.Secret, tt.req.Token).Return(tt.responseAccount, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationAccountsIDGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID

		expectRes *cvaccount.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("ab2f092e-004e-11ee-b834-b7077f22c1eb"),
			},
			"/v1.0/conversation_accounts/ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c",

			uuid.FromStringOrNil("ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c"),

			&cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("ab5a1bbe-004e-11ee-a22d-4f6e1c377a3c"),
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

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationAccountGet(req.Context(), &tt.agent, tt.id).Return(tt.expectRes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationAccountsIDPut(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		req request.BodyConversationAccountsIDPUT
		id  uuid.UUID

		responseAccount *cvaccount.WebhookMessage
		expectRes       *cvaccount.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
			},
			target: "/v1.0/conversation_accounts/009f2ac8-0050-11ee-b416-5f4fb9c7c682",

			req: request.BodyConversationAccountsIDPUT{
				Name:   "test name",
				Detail: "test detail",
				Secret: "test secret",
				Token:  "test token",
			},
			id: uuid.FromStringOrNil("009f2ac8-0050-11ee-b416-5f4fb9c7c682"),

			responseAccount: &cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("009f2ac8-0050-11ee-b416-5f4fb9c7c682"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("009f2ac8-0050-11ee-b416-5f4fb9c7c682"),
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

			// create request
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationAccountUpdate(req.Context(), &tt.agent, tt.id, tt.req.Name, tt.req.Detail, tt.req.Secret, tt.req.Token).Return(tt.responseAccount, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationAccountsIDDelete(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID

		responseAccount *cvaccount.WebhookMessage
		expectRes       *cvaccount.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
			},
			target: "/v1.0/conversation_accounts/31a54f8a-0050-11ee-aa7e-d3a80a493b8b",

			id: uuid.FromStringOrNil("31a54f8a-0050-11ee-aa7e-d3a80a493b8b"),

			responseAccount: &cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("31a54f8a-0050-11ee-aa7e-d3a80a493b8b"),
			},
			expectRes: &cvaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("31a54f8a-0050-11ee-aa7e-d3a80a493b8b"),
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

			// create request
			req, _ := http.NewRequest("DELETE", tt.target, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationAccountDelete(req.Context(), &tt.agent, tt.id).Return(tt.responseAccount, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}
