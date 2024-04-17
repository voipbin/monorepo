package billingaccounts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_billingAccountsPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent
		req   request.BodyBillingAccountsPOST

		responsBillingAccount *bmaccount.WebhookMessage
		expectRes             string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID:         uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			req: request.BodyBillingAccountsPOST{
				Name:          "test name",
				Detail:        "test detail",
				PaymentType:   bmaccount.PaymentTypePrepaid,
				PaymentMethod: bmaccount.PaymentMethodCreditCard,
			},

			responsBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("fa87d0ce-11e6-11ee-9d0a-ef3f8b33f9d9"),
			},

			expectRes: `{"id":"fa87d0ce-11e6-11ee-9d0a-ef3f8b33f9d9","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/billing_accounts", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().BillingAccountCreate(req.Context(), &tt.agent, tt.req.Name, tt.req.Detail, tt.req.PaymentType, tt.req.PaymentMethod).Return(tt.responsBillingAccount, nil)

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

func Test_billingaccountsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent
		req   request.ParamBillingAccountsGET

		resBillingAccounts []*bmaccount.WebhookMessage
		expectRes          string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			request.ParamBillingAccountsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*bmaccount.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("30984a42-11ea-11ee-b5d2-93d4f8db3dca"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"30984a42-11ea-11ee-b5d2-93d4f8db3dca","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			request.ParamBillingAccountsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*bmaccount.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("30caab9a-11ea-11ee-8f18-5735018f9df2"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("30f86f76-11ea-11ee-ae51-ef177df11436"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("312228ca-11ea-11ee-9004-eb6099103496"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"30caab9a-11ea-11ee-8f18-5735018f9df2","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"30f86f76-11ea-11ee-ae51-ef177df11436","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"312228ca-11ea-11ee-9004-eb6099103496","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			reqQuery := fmt.Sprintf("/v1.0/billing_accounts?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().BillingAccountGets(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.resBillingAccounts, nil)

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

func Test_billingAccountsIDDelete(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.WebhookMessage
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			reqQuery:         "/v1.0/billing_accounts/9738a486-11ea-11ee-b56b-db24f5d2c81a",
			billingAccountID: uuid.FromStringOrNil("9738a486-11ea-11ee-b56b-db24f5d2c81a"),

			responseBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("9738a486-11ea-11ee-b56b-db24f5d2c81a"),
			},
			expectRes: `{"id":"9738a486-11ea-11ee-b56b-db24f5d2c81a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().BillingAccountDelete(req.Context(), &tt.agent, tt.billingAccountID).Return(tt.responseBillingAccount, nil)

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

func Test_billingAccountsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		billingAccountID uuid.UUID

		responseBillingAccount *bmaccount.WebhookMessage
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
			},

			reqQuery:         "/v1.0/billing_accounts/602eb6b4-11eb-11ee-b79f-03124621dcc4",
			billingAccountID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),

			responseBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),
			},
			expectRes: `{"id":"602eb6b4-11eb-11ee-b79f-03124621dcc4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().BillingAccountGet(req.Context(), &tt.agent, tt.billingAccountID).Return(tt.responseBillingAccount, nil)

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

func Test_billingAccountsIDPUT(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		reqBody          request.BodyBillingAccountsIDPUT
		billingAccountID uuid.UUID

		responsBillingAccount *bmaccount.WebhookMessage
		expectRes             string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			reqQuery: "/v1.0/billing_accounts/8d1d01bc-4cdd-11ee-a22f-03714037d3db",
			reqBody: request.BodyBillingAccountsIDPUT{
				Name:   "update name",
				Detail: "update detail",
			},
			billingAccountID: uuid.FromStringOrNil("8d1d01bc-4cdd-11ee-a22f-03714037d3db"),

			responsBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("8d1d01bc-4cdd-11ee-a22f-03714037d3db"),
			},

			expectRes: `{"id":"8d1d01bc-4cdd-11ee-a22f-03714037d3db","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().BillingAccountUpdateBasicInfo(req.Context(), &tt.agent, tt.billingAccountID, tt.reqBody.Name, tt.reqBody.Detail).Return(tt.responsBillingAccount, nil)

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

func Test_billingAccountsIDPaymentInfoPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		reqBody          request.BodyBillingAccountsIDPaymentInfoPUT
		billingAccountID uuid.UUID

		responsBillingAccount *bmaccount.WebhookMessage
		expectRes             string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			reqQuery: "/v1.0/billing_accounts/64461024-4cdf-11ee-be1f-e7111eb57d28/payment_info",
			reqBody: request.BodyBillingAccountsIDPaymentInfoPUT{
				PaymentType:   bmaccount.PaymentTypePrepaid,
				PaymentMethod: bmaccount.PaymentMethodNone,
			},
			billingAccountID: uuid.FromStringOrNil("64461024-4cdf-11ee-be1f-e7111eb57d28"),

			responsBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("64461024-4cdf-11ee-be1f-e7111eb57d28"),
			},

			expectRes: `{"id":"64461024-4cdf-11ee-be1f-e7111eb57d28","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().BillingAccountUpdatePaymentInfo(req.Context(), &tt.agent, tt.billingAccountID, tt.reqBody.PaymentType, tt.reqBody.PaymentMethod).Return(tt.responsBillingAccount, nil)

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

func Test_billingAccountsIDBalanceAddForcePOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		reqBody          request.BodyBillingAccountsIDBalanceAddForcePOST
		billingAccountID uuid.UUID
		balance          float32

		responseBillingAccount *bmaccount.WebhookMessage
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
			},

			reqQuery: "/v1.0/billing_accounts/605eae78-11eb-11ee-b8d3-6fd8da9d9879/balance_add_force",
			reqBody: request.BodyBillingAccountsIDBalanceAddForcePOST{
				Balance: 20.9,
			},
			billingAccountID: uuid.FromStringOrNil("605eae78-11eb-11ee-b8d3-6fd8da9d9879"),
			balance:          20.9,

			responseBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("605eae78-11eb-11ee-b8d3-6fd8da9d9879"),
			},
			expectRes: `{"id":"605eae78-11eb-11ee-b8d3-6fd8da9d9879","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			mockSvc.EXPECT().BillingAccountAddBalanceForce(req.Context(), &tt.agent, tt.billingAccountID, tt.balance).Return(tt.responseBillingAccount, nil)

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

func Test_billingAccountsIDSubtractAddForcePOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		reqBody          request.BodyBillingAccountsIDBalanceAddForcePOST
		billingAccountID uuid.UUID
		balance          float32

		responseBillingAccount *bmaccount.WebhookMessage
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
			},

			reqQuery: "/v1.0/billing_accounts/e4e38ff6-11eb-11ee-879b-cb22a78168e4/balance_subtract_force",
			reqBody: request.BodyBillingAccountsIDBalanceAddForcePOST{
				Balance: 20.9,
			},
			billingAccountID: uuid.FromStringOrNil("e4e38ff6-11eb-11ee-879b-cb22a78168e4"),
			balance:          20.9,

			responseBillingAccount: &bmaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("e4e38ff6-11eb-11ee-879b-cb22a78168e4"),
			},
			expectRes: `{"id":"e4e38ff6-11eb-11ee-879b-cb22a78168e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			mockSvc.EXPECT().BillingAccountSubtractBalanceForce(req.Context(), &tt.agent, tt.billingAccountID, tt.balance).Return(tt.responseBillingAccount, nil)

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
