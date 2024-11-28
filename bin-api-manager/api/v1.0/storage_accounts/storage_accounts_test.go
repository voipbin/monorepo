package storage_accounts

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	smaccount "monorepo/bin-storage-manager/models/account"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_storageAccountsGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		size  uint64
		token string

		responseAccounts []*smaccount.WebhookMessage
		expectRes        *response.BodyStorageAccountsGET
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},
			"/v1.0/storage_accounts?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			20,
			"2020-09-20 03:23:20.995000",

			[]*smaccount.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6adce0da-004e-11ee-b74a-23da476139db"),
				},
			},
			&response.BodyStorageAccountsGET{
				Result: []*smaccount.WebhookMessage{
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

			mockSvc.EXPECT().StorageAccountGets(req.Context(), &tt.agent, tt.size, tt.token).Return(tt.responseAccounts, nil)

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

func Test_storageAccountsPost(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		req request.BodyStorageAccountsPOST

		responseAccount *smaccount.WebhookMessage
		expectRes       *smaccount.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},
			target: "/v1.0/storage_accounts",

			req: request.BodyStorageAccountsPOST{
				CustomerID: uuid.FromStringOrNil("a77397a6-1bef-11ef-bb4f-d76f9d478e32"),
			},

			responseAccount: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("ae58a520-1bef-11ef-afdc-571791bb0855"),
			},
			expectRes: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("ae58a520-1bef-11ef-afdc-571791bb0855"),
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

			mockSvc.EXPECT().StorageAccountCreate(req.Context(), &tt.agent, tt.req.CustomerID).Return(tt.responseAccount, nil)

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

func Test_storageAccountsIDGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID

		expectRes *smaccount.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab2f092e-004e-11ee-b834-b7077f22c1eb"),
				},
			},
			"/v1.0/storage_accounts/c85cf9d0-1bef-11ef-a736-e75259c323b2",

			uuid.FromStringOrNil("c85cf9d0-1bef-11ef-a736-e75259c323b2"),

			&smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("c85cf9d0-1bef-11ef-a736-e75259c323b2"),
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

			mockSvc.EXPECT().StorageAccountGet(req.Context(), &tt.agent, tt.id).Return(tt.expectRes, nil)

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

func Test_storageAccountsIDDelete(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID

		responseAccount *smaccount.WebhookMessage
		expectRes       *smaccount.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},
			target: "/v1.0/storage_accounts/c88754b4-1bef-11ef-b6d2-0b09724bcbc3",

			id: uuid.FromStringOrNil("c88754b4-1bef-11ef-b6d2-0b09724bcbc3"),

			responseAccount: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("c88754b4-1bef-11ef-b6d2-0b09724bcbc3"),
			},
			expectRes: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("c88754b4-1bef-11ef-b6d2-0b09724bcbc3"),
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

			mockSvc.EXPECT().StorageAccountDelete(req.Context(), &tt.agent, tt.id).Return(tt.responseAccount, nil)

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
