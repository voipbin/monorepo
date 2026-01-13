package server

import (
	"bytes"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_customerGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCustomer *cscustomer.WebhookMessage

		expectedCustomerID uuid.UUID
		expectedRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			reqQuery: "/customer",

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
			},

			expectedCustomerID: uuid.FromStringOrNil("e25f1af8-c44f-11ef-9d46-bfaf61e659c2"),
			expectedRes:        `{"id":"e25f1af8-c44f-11ef-9d46-bfaf61e659c2","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CustomerGet(req.Context(), &tt.agent, tt.expectedCustomerID).Return(tt.responseCustomer, nil)

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

func Test_customerPut(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCustomer *cscustomer.WebhookMessage

		expectedCustomerID    uuid.UUID
		expectecName          string
		expectedDetail        string
		expectedEmail         string
		expectedPhoneNumber   string
		expectedAddress       string
		expectedWebhookMethod cscustomer.WebhookMethod
		expectedWebhookURI    string
		expectedRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4afd144c-c451-11ef-a8d8-6fd67202355e"),
					CustomerID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			reqQuery: "/customer",
			reqBody:  []byte(`{"name":"new name","detail":"new detail","email":"test@test.com","phone_number":"+821100000001","address":"somewhere","webhook_method":"POST","webhook_uri":"test.com"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
			},

			expectedCustomerID:    uuid.FromStringOrNil("4b7dcc68-c451-11ef-a289-33cbfe065115"),
			expectecName:          "new name",
			expectedDetail:        "new detail",
			expectedEmail:         "test@test.com",
			expectedPhoneNumber:   "+821100000001",
			expectedAddress:       "somewhere",
			expectedWebhookMethod: cscustomer.WebhookMethodPost,
			expectedWebhookURI:    "test.com",
			expectedRes:           `{"id":"4b7dcc68-c451-11ef-a289-33cbfe065115","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CustomerUpdate(req.Context(), &tt.agent, tt.expectedCustomerID, tt.expectecName, tt.expectedDetail, tt.expectedEmail, tt.expectedPhoneNumber, tt.expectedAddress, tt.expectedWebhookMethod, tt.expectedWebhookURI).Return(tt.responseCustomer, nil)

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

func Test_customerBillingAccountIDPut(t *testing.T) {

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
					ID:         uuid.FromStringOrNil("23ad14fa-c514-11ef-a03b-af3d499fdf18"),
					CustomerID: uuid.FromStringOrNil("2422306e-c514-11ef-a89d-2f0585ee15f9"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			reqQuery: "/customer/billing_account_id",
			reqBody:  []byte(`{"billing_account_id":"245bc55e-c514-11ef-85d3-23d66dfc487a"}`),

			responseCustomer: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("2422306e-c514-11ef-a89d-2f0585ee15f9"),
			},

			expectedCustomerID:       uuid.FromStringOrNil("2422306e-c514-11ef-a89d-2f0585ee15f9"),
			expectedBillingAccountID: uuid.FromStringOrNil("245bc55e-c514-11ef-85d3-23d66dfc487a"),
			expectedRes:              `{"id":"2422306e-c514-11ef-a89d-2f0585ee15f9","billing_account_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest(http.MethodPut, tt.reqQuery, bytes.NewBuffer(tt.reqBody))
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
