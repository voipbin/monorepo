package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_billingsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseBillings []*bmbilling.WebhookMessage

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

			reqQuery: "/billings?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseBillings: []*bmbilling.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30984a42-11ea-11ee-b5d2-93d4f8db3dca"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"30984a42-11ea-11ee-b5d2-93d4f8db3dca","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","transaction_type":"","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_type":"","usage_duration":0,"billable_units":0,"rate_token_per_unit":0,"rate_credit_per_unit":0,"amount_token":0,"amount_credit":0,"balance_token_snapshot":0,"balance_credit_snapshot":0,"idempotency_key":"00000000-0000-0000-0000-000000000000","tm_billing_start":null,"tm_billing_end":null,"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/billings?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseBillings: []*bmbilling.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30caab9a-11ea-11ee-8f18-5735018f9df2"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30f86f76-11ea-11ee-ae51-ef177df11436"),
					},
					TMCreate: timePtr("2020-09-20T03:23:22.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("312228ca-11ea-11ee-9004-eb6099103496"),
					},
					TMCreate: timePtr("2020-09-20T03:23:23.995000Z"),
				},
			},
			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"30caab9a-11ea-11ee-8f18-5735018f9df2","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","transaction_type":"","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_type":"","usage_duration":0,"billable_units":0,"rate_token_per_unit":0,"rate_credit_per_unit":0,"amount_token":0,"amount_credit":0,"balance_token_snapshot":0,"balance_credit_snapshot":0,"idempotency_key":"00000000-0000-0000-0000-000000000000","tm_billing_start":null,"tm_billing_end":null,"tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null},{"id":"30f86f76-11ea-11ee-ae51-ef177df11436","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","transaction_type":"","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_type":"","usage_duration":0,"billable_units":0,"rate_token_per_unit":0,"rate_credit_per_unit":0,"amount_token":0,"amount_credit":0,"balance_token_snapshot":0,"balance_credit_snapshot":0,"idempotency_key":"00000000-0000-0000-0000-000000000000","tm_billing_start":null,"tm_billing_end":null,"tm_create":"2020-09-20T03:23:22.995Z","tm_update":null,"tm_delete":null},{"id":"312228ca-11ea-11ee-9004-eb6099103496","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","transaction_type":"","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_type":"","usage_duration":0,"billable_units":0,"rate_token_per_unit":0,"rate_credit_per_unit":0,"amount_token":0,"amount_credit":0,"balance_token_snapshot":0,"balance_credit_snapshot":0,"idempotency_key":"00000000-0000-0000-0000-000000000000","tm_billing_start":null,"tm_billing_end":null,"tm_create":"2020-09-20T03:23:23.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:23.995000Z"}`,
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

			mockSvc.EXPECT().BillingList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseBillings, nil)

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
