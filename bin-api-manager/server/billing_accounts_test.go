package server

import (
	"bytes"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	bmaccount "monorepo/bin-billing-manager/models/account"
	bmallowance "monorepo/bin-billing-manager/models/allowance"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

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
			expectRes: `{"id":"602eb6b4-11eb-11ee-b79f-03124621dcc4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance":0,"payment_type":"","payment_method":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			expectRes:              `{"id":"8d1d01bc-4cdd-11ee-a22f-03714037d3db","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance":0,"payment_type":"","payment_method":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			expectRes:              `{"id":"64461024-4cdf-11ee-be1f-e7111eb57d28","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance":0,"payment_type":"","payment_method":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			expectRes:              `{"id":"605eae78-11eb-11ee-b8d3-6fd8da9d9879","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance":0,"payment_type":"","payment_method":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			expectRes:              `{"id":"e4e38ff6-11eb-11ee-879b-cb22a78168e4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","balance":0,"payment_type":"","payment_method":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

func Test_GetBillingAccountsIdAllowances(t *testing.T) {
	tmCreate := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	cycleStart := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	cycleEnd := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		billingAccountID uuid.UUID

		responseAllowances []*bmallowance.WebhookMessage
		expectRes          string
	}

	tests := []test{
		{
			name: "normal with results and pagination",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
				},
			},

			reqQuery:         "/billing_accounts/602eb6b4-11eb-11ee-b79f-03124621dcc4/allowances",
			billingAccountID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),

			responseAllowances: []*bmallowance.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a1b2c3d4-1234-5678-9abc-def012345678"),
						CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					},
					AccountID:   uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),
					CycleStart:  &cycleStart,
					CycleEnd:    &cycleEnd,
					TokensTotal: 10000,
					TokensUsed:  3500,
					TMCreate:    &tmCreate,
				},
			},
			expectRes: `{"result":[{"id":"a1b2c3d4-1234-5678-9abc-def012345678","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"602eb6b4-11eb-11ee-b79f-03124621dcc4","cycle_start":"2026-02-15T00:00:00Z","cycle_end":"2026-03-15T00:00:00Z","tokens_total":10000,"tokens_used":3500,"tm_create":"2026-02-15T00:00:00Z","tm_update":null,"tm_delete":null}],"next_page_token":"2026-02-15T00:00:00.000000Z"}`,
		},
		{
			name: "empty results",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23443698-11eb-11ee-93d2-83107308dab3"),
				},
			},

			reqQuery:         "/billing_accounts/602eb6b4-11eb-11ee-b79f-03124621dcc4/allowances",
			billingAccountID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),

			responseAllowances: []*bmallowance.WebhookMessage{},
			expectRes:          `{"result":[],"next_page_token":""}`,
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
			mockSvc.EXPECT().BillingAccountAllowancesGet(req.Context(), &tt.agent, tt.billingAccountID, uint64(10), "").Return(tt.responseAllowances, nil)

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
