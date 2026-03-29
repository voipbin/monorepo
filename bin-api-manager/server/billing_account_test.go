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

func TestGetBillingAccount(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

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

			reqQuery: "/billing_account",

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("602eb6b4-11eb-11ee-b79f-03124621dcc4"),
				},
			},
			expectRes: `{"id":"602eb6b4-11eb-11ee-b79f-03124621dcc4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","plan_status":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().BillingAccountSelfGet(req.Context(), &tt.agent).Return(tt.responseBillingAccount, nil)

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

func TestPutBillingAccount(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseBillingAccount *bmaccount.WebhookMessage

		expectName   string
		expectDetail string
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_account",
			reqBody:  []byte(`{"name":"update name","detail":"update detail"}`),

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d1d01bc-4cdd-11ee-a22f-03714037d3db"),
				},
			},

			expectName:   "update name",
			expectDetail: "update detail",
			expectRes:    `{"id":"8d1d01bc-4cdd-11ee-a22f-03714037d3db","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","plan_status":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().BillingAccountSelfUpdateBasicInfo(req.Context(), &tt.agent, tt.expectName, tt.expectDetail).Return(tt.responseBillingAccount, nil)

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

func TestPutBillingAccountPaymentInfo(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseBillingAccount *bmaccount.WebhookMessage

		expectPaymentType   bmaccount.PaymentType
		expectPaymentMethod bmaccount.PaymentMethod
		expectRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billing_account/payment_info",
			reqBody:  []byte(`{"payment_type":"prepaid","payment_method":"credit card"}`),

			responseBillingAccount: &bmaccount.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("64461024-4cdf-11ee-be1f-e7111eb57d28"),
				},
			},

			expectPaymentType:   bmaccount.PaymentTypePrepaid,
			expectPaymentMethod: bmaccount.PaymentMethodCreditCard,
			expectRes:           `{"id":"64461024-4cdf-11ee-be1f-e7111eb57d28","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","plan_type":"","plan_status":"","balance_credit":0,"balance_token":0,"payment_type":"","payment_method":"","tm_last_topup":null,"tm_next_topup":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().BillingAccountSelfUpdatePaymentInfo(req.Context(), &tt.agent, tt.expectPaymentType, tt.expectPaymentMethod).Return(tt.responseBillingAccount, nil)

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
