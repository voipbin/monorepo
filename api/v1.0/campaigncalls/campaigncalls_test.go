package campaigncalls

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_campaigncallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.ParamCampaigncallsGET

		responseCampaigncalls []*cacampaigncall.WebhookMessage
		expectRes             string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			"/v1.0/campaigncalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamCampaigncallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*cacampaigncall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("13d06624-6e29-11ee-8c18-37f3708d43b9"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"13d06624-6e29-11ee-8c18-37f3708d43b9","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},

			"/v1.0/campaigncalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamCampaigncallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*cacampaigncall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("1402a4ea-6e29-11ee-a53c-1b448648df2e"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("142f6b10-6e29-11ee-b771-a7835a2bf8ef"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("212ed990-6e29-11ee-951e-9ffe2d340f93"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"1402a4ea-6e29-11ee-a53c-1b448648df2e","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":""},{"id":"142f6b10-6e29-11ee-b771-a7835a2bf8ef","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":""},{"id":"212ed990-6e29-11ee-951e-9ffe2d340f93","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().CampaigncallGets(req.Context(), &tt.agent, tt.reqBody.PageSize, tt.reqBody.PageToken).Return(tt.responseCampaigncalls, nil)

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

func Test_campaigncallsIDGET(t *testing.T) {

	tests := []struct {
		name           string
		agent          amagent.Agent
		campaigncallID uuid.UUID

		reqQuery string

		response *cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("897e611a-c870-11ec-9b81-a7b70b7cdaa1"),

			"/v1.0/campaigncalls/897e611a-c870-11ec-9b81-a7b70b7cdaa1",

			&cacampaigncall.WebhookMessage{
				ID: uuid.FromStringOrNil("897e611a-c870-11ec-9b81-a7b70b7cdaa1"),
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().CampaigncallGet(req.Context(), &tt.agent, tt.campaigncallID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaigncallsIDDELETE(t *testing.T) {

	tests := []struct {
		name           string
		agent          amagent.Agent
		campaigncallID uuid.UUID

		reqQuery string
		response *cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("afe97cd6-c870-11ec-b750-f3db7eda3a33"),

			"/v1.0/campaigncalls/afe97cd6-c870-11ec-b750-f3db7eda3a33",
			&cacampaigncall.WebhookMessage{
				ID: uuid.FromStringOrNil("afe97cd6-c870-11ec-b750-f3db7eda3a33"),
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().CampaigncallDelete(req.Context(), &tt.agent, tt.campaigncallID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
