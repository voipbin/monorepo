package campaigns

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
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_campaignsPOST(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		requestBody request.BodyCampaignsPOST

		response *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			request.BodyCampaignsPOST{
				Name:         "test name",
				Detail:       "test detail",
				Type:         cacampaign.TypeCall,
				ServiceLevel: 100,
				EndHandle:    cacampaign.EndHandleStop,
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				OutplanID:      uuid.FromStringOrNil("a1380082-c68a-11ec-9fa9-d7588fa9c904"),
				OutdialID:      uuid.FromStringOrNil("a16d488c-c68a-11ec-8252-375e8f888c2f"),
				QueueID:        uuid.FromStringOrNil("a19393ca-c68a-11ec-a78d-a7110df02eb3"),
				NextCampaignID: uuid.FromStringOrNil("a1ba021c-c68a-11ec-b81e-f3e6f905293b"),
			},

			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("1e701ed2-c649-11ec-97e4-87f868a3e3a9"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().CampaignCreate(
				&tt.customer,
				tt.requestBody.Name,
				tt.requestBody.Detail,
				tt.requestBody.Type,
				tt.requestBody.ServiceLevel,
				tt.requestBody.EndHandle,
				tt.requestBody.Actions,
				tt.requestBody.OutplanID,
				tt.requestBody.OutdialID,
				tt.requestBody.QueueID,
				tt.requestBody.NextCampaignID,
			).Return(tt.response, nil)
			req, _ := http.NewRequest("POST", "/v1.0/campaigns", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaignsGET(t *testing.T) {

	type test struct {
		name        string
		customer    cscustomer.Customer
		req         request.ParamCampaignsGET
		resOutdials []*cacampaign.WebhookMessage
		expectRes   string
	}

	tests := []test{
		{
			"1 item",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			request.ParamCampaignsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cacampaign.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"3bc539bc-c68b-11ec-b41f-0776699e7467","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			request.ParamCampaignsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cacampaign.WebhookMessage{
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
			`{"result":[{"id":"3bfa9cc4-c68b-11ec-a1cf-5fffd85773bb","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"3c2648d8-c68b-11ec-a47f-7bfbe26dbdcf","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"3c4d9a1e-c68b-11ec-8b46-5f282fd0eb19","type":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			reqQuery := fmt.Sprintf("/v1.0/campaigns?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CampaignGetsByCustomerID(&tt.customer, tt.req.PageSize, tt.req.PageToken).Return(tt.resOutdials, nil)

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
		name       string
		customer   cscustomer.Customer
		campaignID uuid.UUID

		response *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("832bd31a-c68b-11ec-bcd0-7f66f70ae88d"),

			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("832bd31a-c68b-11ec-bcd0-7f66f70ae88d"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().CampaignGet(&tt.customer, tt.campaignID).Return(tt.response, nil)
			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/campaigns/%s", tt.campaignID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaignsIDDELETE(t *testing.T) {

	tests := []struct {
		name       string
		customer   cscustomer.Customer
		campaignID uuid.UUID

		response *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("aa1a055a-c68b-11ec-99c7-173b42898a47"),
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("aa1a055a-c68b-11ec-99c7-173b42898a47"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().CampaignDelete(&tt.customer, tt.campaignID).Return(tt.response, nil)
			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/campaigns/%s", tt.campaignID), nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaignsIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyCampaignsIDPUT
		response    *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("e2758bfe-c68b-11ec-a1d0-ff54494682b4"),
			request.BodyCampaignsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("e2758bfe-c68b-11ec-a1d0-ff54494682b4"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().CampaignUpdateBasicInfo(&tt.customer, tt.outdialID, tt.requestBody.Name, tt.requestBody.Detail).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/campaigns/"+tt.outdialID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaignsIDStatusPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyCampaignsIDStatusPUT
		response    *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("1bbc5316-c68c-11ec-a2cd-7b9fb7e1e855"),
			request.BodyCampaignsIDStatusPUT{
				Status: cacampaign.StatusRun,
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("1bbc5316-c68c-11ec-a2cd-7b9fb7e1e855"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().CampaignUpdateStatus(&tt.customer, tt.outdialID, tt.requestBody.Status).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/campaigns/"+tt.outdialID.String()+"/status", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaignsIDServiceLevelPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyCampaignsIDServiceLevelPUT
		response    *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("40460ace-c68c-11ec-9694-830803c448f7"),
			request.BodyCampaignsIDServiceLevelPUT{
				ServiceLevel: 100,
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("40460ace-c68c-11ec-9694-830803c448f7"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().CampaignUpdateServiceLevel(&tt.customer, tt.outdialID, tt.requestBody.ServiceLevel).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/campaigns/"+tt.outdialID.String()+"/service_level", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_campaignsIDActionsPUT(t *testing.T) {

	tests := []struct {
		name        string
		customer    cscustomer.Customer
		outdialID   uuid.UUID
		requestBody request.BodyCampaignsIDActionsPUT
		response    *cacampaign.WebhookMessage
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},
			uuid.FromStringOrNil("79027712-c68c-11ec-b75e-27bce33a22a8"),
			request.BodyCampaignsIDActionsPUT{
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("79027712-c68c-11ec-b75e-27bce33a22a8"),
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			mockSvc.EXPECT().CampaignUpdateActions(&tt.customer, tt.outdialID, tt.requestBody.Actions).Return(tt.response, nil)
			req, _ := http.NewRequest("PUT", "/v1.0/campaigns/"+tt.outdialID.String()+"/actions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
