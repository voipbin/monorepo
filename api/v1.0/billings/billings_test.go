package billings

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"

	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_billingsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		req      request.ParamBillingsGET
		reqQuery string

		responseBillings []*bmbilling.WebhookMessage
		expectRes        string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			request.ParamBillingsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			"/v1.0/billings?page_size=10&page_token=2020-09-20T03:23:20.995000",

			[]*bmbilling.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("30984a42-11ea-11ee-b5d2-93d4f8db3dca"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"30984a42-11ea-11ee-b5d2-93d4f8db3dca","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_per_unit":0,"cost_total":0,"billing_unit_count":0,"tm_billing_start":"","tm_billing_end":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			request.ParamBillingsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			"/v1.0/billings?page_size=10&page_token=2020-09-20T03:23:20.995000",

			[]*bmbilling.WebhookMessage{
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
			`{"result":[{"id":"30caab9a-11ea-11ee-8f18-5735018f9df2","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_per_unit":0,"cost_total":0,"billing_unit_count":0,"tm_billing_start":"","tm_billing_end":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"30f86f76-11ea-11ee-ae51-ef177df11436","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_per_unit":0,"cost_total":0,"billing_unit_count":0,"tm_billing_start":"","tm_billing_end":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"312228ca-11ea-11ee-9004-eb6099103496","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","cost_per_unit":0,"cost_total":0,"billing_unit_count":0,"tm_billing_start":"","tm_billing_end":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			// reqQuery := fmt.Sprintf("/v1.0/billing?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			mockSvc.EXPECT().BillingGets(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.responseBillings, nil)

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
