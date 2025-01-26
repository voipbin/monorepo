package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_campaignsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		response *cacampaign.WebhookMessage

		expectName           string
		expectDetail         string
		expectType           cacampaign.Type
		expectServiceLevel   int
		expectEndHandle      cacampaign.EndHandle
		expectActions        []fmaction.Action
		expectOutplanID      uuid.UUID
		expectOutdialID      uuid.UUID
		expectQueueID        uuid.UUID
		expectNextCampaignID uuid.UUID
		expectRes            string
	}{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","type":"call","service_level":100,"end_handle":"stop","actions":[{"type":"answer"}],"outplan_id":"a1380082-c68a-11ec-9fa9-d7588fa9c904","outdial_id":"a16d488c-c68a-11ec-8252-375e8f888c2f","queue_id":"a19393ca-c68a-11ec-a78d-a7110df02eb3","next_campaign_id":"a1ba021c-c68a-11ec-b81e-f3e6f905293b"}`),

			response: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("1e701ed2-c649-11ec-97e4-87f868a3e3a9"),
			},

			expectName:         "test name",
			expectDetail:       "test detail",
			expectType:         cacampaign.TypeCall,
			expectServiceLevel: 100,
			expectEndHandle:    cacampaign.EndHandleStop,
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectOutplanID:      uuid.FromStringOrNil("a1380082-c68a-11ec-9fa9-d7588fa9c904"),
			expectOutdialID:      uuid.FromStringOrNil("a16d488c-c68a-11ec-8252-375e8f888c2f"),
			expectQueueID:        uuid.FromStringOrNil("a19393ca-c68a-11ec-a78d-a7110df02eb3"),
			expectNextCampaignID: uuid.FromStringOrNil("a1ba021c-c68a-11ec-b81e-f3e6f905293b"),
			expectRes:            `{"id":"1e701ed2-c649-11ec-97e4-87f868a3e3a9","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().CampaignCreate(
				req.Context(),
				&tt.agent,
				tt.expectName,
				tt.expectDetail,
				tt.expectType,
				tt.expectServiceLevel,
				tt.expectEndHandle,
				tt.expectActions,
				tt.expectOutplanID,
				tt.expectOutdialID,
				tt.expectQueueID,
				tt.expectNextCampaignID,
			).Return(tt.response, nil)

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

func Test_campaignsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCampaigns []*cacampaign.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseCampaigns: []*cacampaign.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"3bc539bc-c68b-11ec-b41f-0776699e7467","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns?page_size=10&page_token=2020-09-20%2003:23:20.995000",

			responseCampaigns: []*cacampaign.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bfa9cc4-c68b-11ec-a1cf-5fffd85773bb"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("3c2648d8-c68b-11ec-a47f-7bfbe26dbdcf"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("3c4d9a1e-c68b-11ec-8b46-5f282fd0eb19"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"3bfa9cc4-c68b-11ec-a1cf-5fffd85773bb","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"3c2648d8-c68b-11ec-a47f-7bfbe26dbdcf","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"3c4d9a1e-c68b-11ec-8b46-5f282fd0eb19","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().CampaignGetsByCustomerID(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCampaigns, nil)

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

func Test_campaignsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/832bd31a-c68b-11ec-bcd0-7f66f70ae88d",

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("832bd31a-c68b-11ec-bcd0-7f66f70ae88d"),
			},

			expectCampaignID: uuid.FromStringOrNil("832bd31a-c68b-11ec-bcd0-7f66f70ae88d"),
			expectRes:        `{"id":"832bd31a-c68b-11ec-bcd0-7f66f70ae88d","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignGet(req.Context(), &tt.agent, tt.expectCampaignID).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID uuid.UUID
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/aa1a055a-c68b-11ec-99c7-173b42898a47",

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("aa1a055a-c68b-11ec-99c7-173b42898a47"),
			},

			expectCampaignID: uuid.FromStringOrNil("aa1a055a-c68b-11ec-99c7-173b42898a47"),
			expectRes:        `{"id":"aa1a055a-c68b-11ec-99c7-173b42898a47","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignDelete(req.Context(), &tt.agent, tt.expectCampaignID).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID   uuid.UUID
		expectName         string
		expectDetail       string
		expectType         cacampaign.Type
		expectServiceLevel int
		expectEndHandle    cacampaign.EndHandle
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/e2758bfe-c68b-11ec-a1d0-ff54494682b4",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","type":"call","service_level":100,"end_handle":"continue"}`),

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("e2758bfe-c68b-11ec-a1d0-ff54494682b4"),
			},

			expectCampaignID:   uuid.FromStringOrNil("e2758bfe-c68b-11ec-a1d0-ff54494682b4"),
			expectName:         "test name",
			expectDetail:       "test detail",
			expectType:         cacampaign.TypeCall,
			expectServiceLevel: 100,
			expectEndHandle:    cacampaign.EndHandleContinue,
			expectRes:          `{"id":"e2758bfe-c68b-11ec-a1d0-ff54494682b4","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignUpdateBasicInfo(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectName, tt.expectDetail, tt.expectType, tt.expectServiceLevel, tt.expectEndHandle).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDStatusPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID uuid.UUID
		expectStatus     cacampaign.Status
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/1bbc5316-c68c-11ec-a2cd-7b9fb7e1e855/status",
			reqBody:  []byte(`{"status":"run"}`),

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("1bbc5316-c68c-11ec-a2cd-7b9fb7e1e855"),
			},

			expectCampaignID: uuid.FromStringOrNil("1bbc5316-c68c-11ec-a2cd-7b9fb7e1e855"),
			expectStatus:     cacampaign.StatusRun,
			expectRes:        `{"id":"1bbc5316-c68c-11ec-a2cd-7b9fb7e1e855","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignUpdateStatus(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectStatus).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDServiceLevelPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery         string
		reqBody          []byte
		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID   uuid.UUID
		expectServiceLevel int
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/40460ace-c68c-11ec-9694-830803c448f7/service_level",
			reqBody:  []byte(`{"service_level":100}`),
			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("40460ace-c68c-11ec-9694-830803c448f7"),
			},

			expectCampaignID:   uuid.FromStringOrNil("40460ace-c68c-11ec-9694-830803c448f7"),
			expectServiceLevel: 100,
			expectRes:          `{"id":"40460ace-c68c-11ec-9694-830803c448f7","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignUpdateServiceLevel(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectServiceLevel).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDActionsPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID uuid.UUID
		expectActions    []fmaction.Action
		expectRes        string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/79027712-c68c-11ec-b75e-27bce33a22a8/actions",
			reqBody:  []byte(`{"actions":[{"type":"answer"},{"type":"talk","option":{"text":"hello"}}]}`),

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("79027712-c68c-11ec-b75e-27bce33a22a8"),
			},

			expectCampaignID: uuid.FromStringOrNil("79027712-c68c-11ec-b75e-27bce33a22a8"),
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeTalk,
					Option: []byte(`{"text":"hello"}`),
				},
			},
			expectRes: `{"id":"79027712-c68c-11ec-b75e-27bce33a22a8","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignUpdateActions(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectActions).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDResourceInfoPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID     uuid.UUID
		expectOutplanID      uuid.UUID
		expectOutdialID      uuid.UUID
		expectQueueID        uuid.UUID
		expectNextCampaignID uuid.UUID
		expectRes            string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/campaigns/47a64a88-c6b7-11ec-973d-1f139c4db335/resource_info",
			reqBody:  []byte(`{"outplan_id":"60fbac4e-c6b7-11ec-869d-3bb7acd5d21a","outdial_id":"61276366-c6b7-11ec-9a5f-07c38e459ee5","queue_id":"614def2c-c6b7-11ec-be49-f350c18391d0","next_campaign_id":"2d21918e-7cd4-11ee-9f07-c3d4e266f6f6"}`),

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("47a64a88-c6b7-11ec-973d-1f139c4db335"),
			},

			expectCampaignID:     uuid.FromStringOrNil("47a64a88-c6b7-11ec-973d-1f139c4db335"),
			expectOutplanID:      uuid.FromStringOrNil("60fbac4e-c6b7-11ec-869d-3bb7acd5d21a"),
			expectOutdialID:      uuid.FromStringOrNil("61276366-c6b7-11ec-9a5f-07c38e459ee5"),
			expectQueueID:        uuid.FromStringOrNil("614def2c-c6b7-11ec-be49-f350c18391d0"),
			expectNextCampaignID: uuid.FromStringOrNil("2d21918e-7cd4-11ee-9f07-c3d4e266f6f6"),
			expectRes:            `{"id":"47a64a88-c6b7-11ec-973d-1f139c4db335","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignUpdateResourceInfo(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectOutplanID, tt.expectOutdialID, tt.expectQueueID, tt.expectNextCampaignID).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDNextCampaignIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCampaign *cacampaign.WebhookMessage

		expectCampaignID     uuid.UUID
		expectNextCampaignID uuid.UUID
		expectRes            string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			expectCampaignID: uuid.FromStringOrNil("a76dcb26-c6b7-11ec-b0dc-23d4f8625f83"),

			reqQuery: "/campaigns/a76dcb26-c6b7-11ec-b0dc-23d4f8625f83/next_campaign_id",
			reqBody:  []byte(`{"next_campaign_id":"b045bff6-c6b7-11ec-8d03-2f6187fcf80f"}`),

			responseCampaign: &cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("a76dcb26-c6b7-11ec-b0dc-23d4f8625f83"),
			},

			expectNextCampaignID: uuid.FromStringOrNil("b045bff6-c6b7-11ec-8d03-2f6187fcf80f"),
			expectRes:            `{"id":"a76dcb26-c6b7-11ec-b0dc-23d4f8625f83","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CampaignUpdateNextCampaignID(req.Context(), &tt.agent, tt.expectCampaignID, tt.expectNextCampaignID).Return(tt.responseCampaign, nil)

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

func Test_campaignsIDCampaigncallsGET(t *testing.T) {

	type test struct {
		name       string
		agent      amagent.Agent
		campaignID uuid.UUID

		reqQuery    string
		reqBody     request.ParamCampaignsIDCampaigncallsGET
		resOutdials []*cacampaigncall.WebhookMessage
		expectRes   string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("571e5aa6-c86e-11ec-a62f-d7989ff2e4dd"),

			"/campaigns/571e5aa6-c86e-11ec-a62f-d7989ff2e4dd/campaigncalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamCampaignsIDCampaigncallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},
			[]*cacampaigncall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"3bc539bc-c68b-11ec-b41f-0776699e7467","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("ef319a88-c86e-11ec-a8b2-abe87e962b9b"),

			"/campaigns/ef319a88-c86e-11ec-a8b2-abe87e962b9b/campaigncalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamCampaignsIDCampaigncallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},
			[]*cacampaigncall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("ef5da59c-c86e-11ec-95bf-b7309c164fc2"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("ef83ff26-c86e-11ec-bfae-d34d64f4c3a5"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("efab58fa-c86e-11ec-9fcb-4b7edd03d7cb"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"ef5da59c-c86e-11ec-95bf-b7309c164fc2","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"ef83ff26-c86e-11ec-bfae-d34d64f4c3a5","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"efab58fa-c86e-11ec-9fcb-4b7edd03d7cb","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().CampaigncallGetsByCampaignID(req.Context(), &tt.agent, tt.campaignID, tt.reqBody.PageSize, tt.reqBody.PageToken).Return(tt.resOutdials, nil)

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
