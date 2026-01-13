package server

import (
	"bytes"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostBillingAccounts(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqBody []byte

		responsBillingAccount *bmaccount.WebhookMessage

		expectedName          string
		expectedDetail        string
		expectedPaymentType   bmaccount.PaymentType
		expectedPaymentMethod bmaccount.PaymentMethod
		expectRes             string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqBody: []byte(`{"name":"test name","detail":"test detail","payment_type":"prepaid","payment_method":"credit card"}`),

			responsBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa87d0ce-11e6-11ee-9d0a-ef3f8b33f9d9"),
				},
			},

			expectedName:          "test name",
			expectedDetail:        "test detail",
			expectedPaymentType:   bmaccount.PaymentTypePrepaid,
			expectedPaymentMethod: bmaccount.PaymentMethodCreditCard,
			expectRes:             `{"id":"fa87d0ce-11e6-11ee-9d0a-ef3f8b33f9d9","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", "/billing_accounts", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().BillingAccountCreate(req.Context(), &tt.agent, tt.expectedName, tt.expectedDetail, tt.expectedPaymentType, tt.expectedPaymentMethod).Return(tt.responsBillingAccount, nil)

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

func Test_GetBillingAccounts(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		resBillingAccounts []*bmaccount.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_accounts?page_size=10&page_token=2020-09-20T03:23:20.995000",

			resBillingAccounts: []*bmaccount.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30984a42-11ea-11ee-b5d2-93d4f8db3dca"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000",
			expectRes:       `{"result":[{"id":"30984a42-11ea-11ee-b5d2-93d4f8db3dca","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_accounts?page_size=10&page_token=2020-09-20T03:23:20.995000",

			resBillingAccounts: []*bmaccount.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30caab9a-11ea-11ee-8f18-5735018f9df2"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30f86f76-11ea-11ee-ae51-ef177df11436"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("312228ca-11ea-11ee-9004-eb6099103496"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000",
			expectRes:       `{"result":[{"id":"30caab9a-11ea-11ee-8f18-5735018f9df2","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"30f86f76-11ea-11ee-ae51-ef177df11436","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"312228ca-11ea-11ee-9004-eb6099103496","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().BillingAccountGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.resBillingAccounts, nil)

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

func Test_DeleteBillingAccountsId(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseBillingAccount *bmaccount.WebhookMessage
		expectBillingAccountID uuid.UUID
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_accounts/9738a486-11ea-11ee-b56b-db24f5d2c81a",

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9738a486-11ea-11ee-b56b-db24f5d2c81a"),
				},
			},
			expectBillingAccountID: uuid.FromStringOrNil("9738a486-11ea-11ee-b56b-db24f5d2c81a"),
			expectRes:              `{"id":"9738a486-11ea-11ee-b56b-db24f5d2c81a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().BillingAccountDelete(req.Context(), &tt.agent, tt.expectBillingAccountID).Return(tt.responseBillingAccount, nil)

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

func Test_GetBillingAccountsId(t *testing.T) {

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
				},
			},

			reqQuery:         "/billing_accounts/602eb6b4-11eb-11ee-b79f-03124621dcc4",
			billingAccountID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),
				},
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

func Test_PutBillingAccountsId(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responsBillingAccount *bmaccount.WebhookMessage

		expectBillingAccountID uuid.UUID
		expectName             string
		expectDetail           string
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_accounts/8d1d01bc-4cdd-11ee-a22f-03714037d3db",
			reqBody:  []byte(`{"name":"update name","detail":"update detail"}`),

			responsBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d1d01bc-4cdd-11ee-a22f-03714037d3db"),
				},
			},

			expectBillingAccountID: uuid.FromStringOrNil("8d1d01bc-4cdd-11ee-a22f-03714037d3db"),
			expectName:             "update name",
			expectDetail:           "update detail",
			expectRes:              `{"id":"8d1d01bc-4cdd-11ee-a22f-03714037d3db","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().BillingAccountUpdateBasicInfo(req.Context(), &tt.agent, tt.expectBillingAccountID, tt.expectName, tt.expectDetail).Return(tt.responsBillingAccount, nil)

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

func Test_PutBillingAccountsIdPaymentInfo(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responsBillingAccount *bmaccount.WebhookMessage

		expectBillingAccountID uuid.UUID
		expectPaymentType      bmaccount.PaymentType
		expectPaymentMethod    bmaccount.PaymentMethod
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_accounts/64461024-4cdf-11ee-be1f-e7111eb57d28/payment_info",
			reqBody:  []byte(`{"payment_type":"prepaid","payment_method":"credit card"}`),

			responsBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("64461024-4cdf-11ee-be1f-e7111eb57d28"),
				},
			},

			expectBillingAccountID: uuid.FromStringOrNil("64461024-4cdf-11ee-be1f-e7111eb57d28"),
			expectPaymentType:      bmaccount.PaymentTypePrepaid,
			expectPaymentMethod:    bmaccount.PaymentMethodCreditCard,
			expectRes:              `{"id":"64461024-4cdf-11ee-be1f-e7111eb57d28","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().BillingAccountUpdatePaymentInfo(req.Context(), &tt.agent, tt.expectBillingAccountID, tt.expectPaymentType, tt.expectPaymentMethod).Return(tt.responsBillingAccount, nil)

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

func Test_PostBillingAccountsIdBalanceAddForce(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseBillingAccount *bmaccount.WebhookMessage

		expectBillingAccountID uuid.UUID
		expectBalance          float32
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
				},
			},

			reqQuery: "/billing_accounts/605eae78-11eb-11ee-b8d3-6fd8da9d9879/balance_add_force",
			reqBody:  []byte(`{"balance": 20.9}`),

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("605eae78-11eb-11ee-b8d3-6fd8da9d9879"),
				},
			},

			expectBillingAccountID: uuid.FromStringOrNil("605eae78-11eb-11ee-b8d3-6fd8da9d9879"),
			expectBalance:          20.9,
			expectRes:              `{"id":"605eae78-11eb-11ee-b8d3-6fd8da9d9879","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().BillingAccountAddBalanceForce(req.Context(), &tt.agent, tt.expectBillingAccountID, tt.expectBalance).Return(tt.responseBillingAccount, nil)

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

func Test_PostBillingAccountsIdBalanceSubtractForce(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseBillingAccount *bmaccount.WebhookMessage

		expectBillingAccountID uuid.UUID
		expectBalance          float32
		expectRes              string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
				},
			},

			reqQuery: "/billing_accounts/e4e38ff6-11eb-11ee-879b-cb22a78168e4/balance_subtract_force",
			reqBody:  []byte(`{"balance": 20.9}`),

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e4e38ff6-11eb-11ee-879b-cb22a78168e4"),
				},
			},

			expectBillingAccountID: uuid.FromStringOrNil("e4e38ff6-11eb-11ee-879b-cb22a78168e4"),
			expectBalance:          20.9,
			expectRes:              `{"id":"e4e38ff6-11eb-11ee-879b-cb22a78168e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","balance":0,"payment_type":"","payment_method":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().BillingAccountSubtractBalanceForce(req.Context(), &tt.agent, tt.expectBillingAccountID, tt.expectBalance).Return(tt.responseBillingAccount, nil)

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
