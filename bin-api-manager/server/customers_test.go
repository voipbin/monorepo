package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_customersPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectName          string
		expectDetail        string
		expectEmail         string
		expectPhoneNumber   string
		expectAddress       string
		expectWebhookMethod cscustomer.WebhookMethod
		expectWebhookURI    string
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			reqQuery: "/customers",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("271353a8-83f3-11ec-9386-8be19d563155"),
			},

			expectName:          "test name",
			expectDetail:        "test detail",
			expectEmail:         "test@test.com",
			expectPhoneNumber:   "+821100000001",
			expectAddress:       "somewhere",
			expectWebhookMethod: cscustomer.WebhookMethodPost,
			expectWebhookURI:    "test.com",
			expectRes:           `{"id":"271353a8-83f3-11ec-9386-8be19d563155","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().CustomerCreate(
				req.Context(),
				&tt.agent,
				tt.expectName,
				tt.expectDetail,
				tt.expectEmail,
				tt.expectPhoneNumber,
				tt.expectAddress,
				tt.expectWebhookMethod,
				tt.expectWebhookURI,
			).Return(tt.responseCustomer, nil)

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

func Test_customersGet(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCustomers []*cscustomer.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectFilters   map[string]string
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/customers?page_size=20&page_token=2020-09-20T03:23:20.995000Z",

			responseCustomers: []*cscustomer.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("52bac7ec-83f4-11ec-a083-c3cf3f92a2e3"),
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectFilters: map[string]string{
				"deleted": "false",
			},
			expectRes: `{"result":[{"id":"52bac7ec-83f4-11ec-a083-c3cf3f92a2e3","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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

			mockSvc.EXPECT().CustomerList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken, tt.expectFilters).Return(tt.responseCustomers, nil)

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

func Test_customersIDGet(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCustomer *cscustomer.WebhookMessage

		expectCustomerID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			reqQuery: "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},

			expectCustomerID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			expectRes:        `{"id":"d98ed7ec-83f7-11ec-8b43-e7de0184974f","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().CustomerGet(req.Context(), &tt.agent, tt.expectCustomerID).Return(tt.responseCustomer, nil)

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

func Test_customersIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectCustomerID    uuid.UUID
		expectName          string
		expectDetail        string
		expectEmail         string
		expectPhoneNumber   string
		expectAddress       string
		expectWebhookMethod cscustomer.WebhookMethod
		expectWebhookURI    string
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			reqQuery: "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",
			reqBody:  []byte(`{"name":"new name","detail":"new detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},

			expectCustomerID:    uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			expectName:          "new name",
			expectDetail:        "new detail",
			expectEmail:         "test@test.com",
			expectPhoneNumber:   "+821100000001",
			expectAddress:       "somewhere",
			expectWebhookMethod: cscustomer.WebhookMethodPost,
			expectWebhookURI:    "test.com",
			expectRes:           `{"id":"d98ed7ec-83f7-11ec-8b43-e7de0184974f","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().CustomerUpdate(req.Context(), &tt.agent, tt.expectCustomerID, tt.expectName, tt.expectDetail, tt.expectEmail, tt.expectPhoneNumber, tt.expectAddress, tt.expectWebhookMethod, tt.expectWebhookURI).Return(tt.responseCustomer, nil)

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

func Test_customersIDDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCustomer *cscustomer.WebhookMessage

		expectCustomerID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			reqQuery: "/customers/d98ed7ec-83f7-11ec-8b43-e7de0184974f",

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			},

			expectCustomerID: uuid.FromStringOrNil("d98ed7ec-83f7-11ec-8b43-e7de0184974f"),
			expectRes:        `{"id":"d98ed7ec-83f7-11ec-8b43-e7de0184974f","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().CustomerDelete(req.Context(), &tt.agent, tt.expectCustomerID).Return(tt.responseCustomer, nil)

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

func Test_customersIDBillingAccountIDPut(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectedCustomerID       uuid.UUID
		expectedBillingAccountID uuid.UUID
		expectedRes              string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},

			reqQuery: "/customers/cc876058-1773-11ee-9694-136fe246dd34/billing_account_id",
			reqBody:  []byte(`{"billing_account_id":"ccc776b6-1773-11ee-bea5-d78345c015af"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			},

			expectedCustomerID:       uuid.FromStringOrNil("cc876058-1773-11ee-9694-136fe246dd34"),
			expectedBillingAccountID: uuid.FromStringOrNil("ccc776b6-1773-11ee-bea5-d78345c015af"),
			expectedRes:              `{"id":"cc876058-1773-11ee-9694-136fe246dd34","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().CustomerUpdateBillingAccountID(req.Context(), &tt.agent, tt.expectedCustomerID, tt.expectedBillingAccountID).Return(tt.responseCustomer, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}
