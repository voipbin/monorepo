package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	smaccount "monorepo/bin-storage-manager/models/account"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_storageAccountsGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAccounts []*smaccount.WebhookMessage

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

			reqQuery: "/storage_accounts?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",

			responseAccounts: []*smaccount.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6adce0da-004e-11ee-b74a-23da476139db"),
				},
			},
			expectRes: `{"result":[{"id":"6adce0da-004e-11ee-b74a-23da476139db","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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

			mockSvc.EXPECT().StorageAccountGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken.Return(tt.responseAccounts, nil)

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

func Test_storageAccountsPost(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAccount *smaccount.WebhookMessage

		expectCustomerID uuid.UUID
		expectRes        string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},

			reqQuery: "/storage_accounts",
			reqBody:  []byte(`{"customer_id":"a77397a6-1bef-11ef-bb4f-d76f9d478e32"}`),

			responseAccount: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("ae58a520-1bef-11ef-afdc-571791bb0855"),
			},

			expectCustomerID: uuid.FromStringOrNil("a77397a6-1bef-11ef-bb4f-d76f9d478e32"),
			expectRes:        `{"id":"ae58a520-1bef-11ef-afdc-571791bb0855","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().StorageAccountCreate(req.Context(), &tt.agent, tt.expectCustomerID.Return(tt.responseAccount, nil)

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

func Test_storageAccountsIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseStorageAccount *smaccount.WebhookMessage

		expectStorageAccountID uuid.UUID
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab2f092e-004e-11ee-b834-b7077f22c1eb"),
				},
			},

			reqQuery: "/storage_accounts/c85cf9d0-1bef-11ef-a736-e75259c323b2",

			expectStorageAccountID: uuid.FromStringOrNil("c85cf9d0-1bef-11ef-a736-e75259c323b2"),

			responseStorageAccount: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("c85cf9d0-1bef-11ef-a736-e75259c323b2"),
			},
			expectRes: `{"id":"c85cf9d0-1bef-11ef-a736-e75259c323b2","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().StorageAccountGet(req.Context(), &tt.agent, tt.expectStorageAccountID.Return(tt.responseStorageAccount, nil)

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

func Test_storageAccountsIDDelete(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAccount        *smaccount.WebhookMessage
		expectStorageAccountID uuid.UUID
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("59d63b06-004e-11ee-b272-731775b3fdc8"),
				},
			},

			reqQuery: "/storage_accounts/c88754b4-1bef-11ef-b6d2-0b09724bcbc3",

			responseAccount: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("c88754b4-1bef-11ef-b6d2-0b09724bcbc3"),
			},

			expectStorageAccountID: uuid.FromStringOrNil("c88754b4-1bef-11ef-b6d2-0b09724bcbc3"),
			expectRes:              `{"id":"c88754b4-1bef-11ef-b6d2-0b09724bcbc3","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().StorageAccountDelete(req.Context(), &tt.agent, tt.expectStorageAccountID.Return(tt.responseAccount, nil)

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
