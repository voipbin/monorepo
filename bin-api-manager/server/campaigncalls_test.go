package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_campaigncallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCampaigncalls []*cacampaigncall.WebhookMessage

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

			reqQuery: "/campaigncalls?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseCampaigncalls: []*cacampaigncall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("13d06624-6e29-11ee-8c18-37f3708d43b9"),
					},
					TMCreate: "2020-09-20T03:23:21.995000Z",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"13d06624-6e29-11ee-8c18-37f3708d43b9","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:21.995000Z","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/campaigncalls?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseCampaigncalls: []*cacampaigncall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1402a4ea-6e29-11ee-a53c-1b448648df2e"),
					},
					TMCreate: "2020-09-20T03:23:21.995000Z",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("142f6b10-6e29-11ee-b771-a7835a2bf8ef"),
					},
					TMCreate: "2020-09-20T03:23:22.995000Z",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("212ed990-6e29-11ee-951e-9ffe2d340f93"),
					},
					TMCreate: "2020-09-20T03:23:23.995000Z",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"1402a4ea-6e29-11ee-a53c-1b448648df2e","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:21.995000Z","tm_update":"","tm_delete":""},{"id":"142f6b10-6e29-11ee-b771-a7835a2bf8ef","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:22.995000Z","tm_update":"","tm_delete":""},{"id":"212ed990-6e29-11ee-951e-9ffe2d340f93","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:23.995000Z","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000Z"}`,
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

			mockSvc.EXPECT().CampaigncallList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCampaigncalls, nil)

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
		name  string
		agent amagent.Agent

		reqQuery string

		responseCampaigncall *cacampaigncall.WebhookMessage

		expectCampaigncallID uuid.UUID
		expectRes            string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigncalls/897e611a-c870-11ec-9b81-a7b70b7cdaa1",

			responseCampaigncall: &cacampaigncall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("897e611a-c870-11ec-9b81-a7b70b7cdaa1"),
				},
			},

			expectCampaigncallID: uuid.FromStringOrNil("897e611a-c870-11ec-9b81-a7b70b7cdaa1"),
			expectRes:            `{"id":"897e611a-c870-11ec-9b81-a7b70b7cdaa1","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaigncallGet(req.Context(), &tt.agent, tt.expectCampaigncallID).Return(tt.responseCampaigncall, nil)

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

func Test_campaigncallsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery             string
		responseCampaigncall *cacampaigncall.WebhookMessage

		expectCampaigncallID uuid.UUID
		expectRes            string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigncalls/afe97cd6-c870-11ec-b750-f3db7eda3a33",
			responseCampaigncall: &cacampaigncall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("afe97cd6-c870-11ec-b750-f3db7eda3a33"),
				},
			},

			expectCampaigncallID: uuid.FromStringOrNil("afe97cd6-c870-11ec-b750-f3db7eda3a33"),
			expectRes:            `{"id":"afe97cd6-c870-11ec-b750-f3db7eda3a33","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaigncallDelete(req.Context(), &tt.agent, tt.expectCampaigncallID).Return(tt.responseCampaigncall, nil)

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
