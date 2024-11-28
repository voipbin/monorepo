package outdials

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_outdialsGET(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		req         request.ParamOutdialsGET
		resOutdials []*omoutdial.WebhookMessage
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
			request.ParamOutdialsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*omoutdial.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("438f0ccc-c64a-11ec-9ac6-b729ca9f28bf"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"438f0ccc-c64a-11ec-9ac6-b729ca9f28bf","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.ParamOutdialsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*omoutdial.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("ad4ec08a-c64a-11ec-ad4d-2b9c85718834"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("b088244e-c64a-11ec-afb8-2f6ebf108ed8"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("c3e247e0-c64a-11ec-b415-c786d3fa957c"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"ad4ec08a-c64a-11ec-ad4d-2b9c85718834","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"b088244e-c64a-11ec-afb8-2f6ebf108ed8","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"c3e247e0-c64a-11ec-b415-c786d3fa957c","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			reqQuery := fmt.Sprintf("/v1.0/outdials?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().OutdialGetsByCustomerID(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.resOutdials, nil)

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

func Test_outdialsPOST(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		requestBody request.BodyOutdialsPOST

		response *omoutdial.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.BodyOutdialsPOST{
				CampaignID: uuid.FromStringOrNil("5770a50e-1a94-45fc-9ba1-79064573cf06"),
				Name:       "test name",
				Detail:     "test detail",
				Data:       "test data",
			},

			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("99b197a5-010e-4f4e-b9fc-aae44e241ddb"),
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/outdials", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialCreate(req.Context(), &tt.agent, tt.requestBody.CampaignID, tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Data).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDGET(t *testing.T) {

	tests := []struct {
		name      string
		agent     amagent.Agent
		outdialID uuid.UUID

		response *omoutdial.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("3bb463ed-aa6e-4b64-9fc5-b1fc62096b67"),

			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("3bb463ed-aa6e-4b64-9fc5-b1fc62096b67"),
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

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/outdials/%s", tt.outdialID), nil)
			mockSvc.EXPECT().OutdialGet(req.Context(), &tt.agent, tt.outdialID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDDELETE(t *testing.T) {

	tests := []struct {
		name      string
		agent     amagent.Agent
		outdialID uuid.UUID

		response *omoutdial.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("37877e5d-ebe3-492d-be8c-54d62e98b4db"),
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("37877e5d-ebe3-492d-be8c-54d62e98b4db"),
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

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/outdials/%s", tt.outdialID), nil)
			mockSvc.EXPECT().OutdialDelete(req.Context(), &tt.agent, tt.outdialID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		outdialID   uuid.UUID
		requestBody request.BodyOutdialsIDPUT
		response    *omoutdial.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("38114323-144a-499c-bfde-f3d8af114a7a"),
			request.BodyOutdialsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("38114323-144a-499c-bfde-f3d8af114a7a"),
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", "/v1.0/outdials/"+tt.outdialID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialUpdateBasicInfo(req.Context(), &tt.agent, tt.outdialID, tt.requestBody.Name, tt.requestBody.Detail).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDCampaignIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		outdialID   uuid.UUID
		requestBody request.BodyOutdialsIDCampaignIDPUT
		response    *omoutdial.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("607ea822-121d-4a52-ad3f-3f5320445ec8"),
			request.BodyOutdialsIDCampaignIDPUT{
				CampaignID: uuid.FromStringOrNil("caad42fb-8266-4a24-be3f-9963ba14a20a"),
			},
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("607ea822-121d-4a52-ad3f-3f5320445ec8"),
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", "/v1.0/outdials/"+tt.outdialID.String()+"/campaign_id", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialUpdateCampaignID(req.Context(), &tt.agent, tt.outdialID, tt.requestBody.CampaignID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDDataPUT(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		outdialID   uuid.UUID
		requestBody request.BodyOutdialsIDDataPUT
		response    *omoutdial.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("056eefbd-8afa-402a-8c9e-681040ec8803"),
			request.BodyOutdialsIDDataPUT{
				Data: "test data",
			},
			&omoutdial.WebhookMessage{
				ID: uuid.FromStringOrNil("056eefbd-8afa-402a-8c9e-681040ec8803"),
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", "/v1.0/outdials/"+tt.outdialID.String()+"/data", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialUpdateData(req.Context(), &tt.agent, tt.outdialID, tt.requestBody.Data).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDTargetsPOST(t *testing.T) {

	tests := []struct {
		name      string
		agent     amagent.Agent
		outdialID uuid.UUID

		requestBody request.BodyOutdialsIDTargetsPOST

		response *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("726d6b88-2028-44fe-a415-a58067d98acf"),
			request.BodyOutdialsIDTargetsPOST{
				Name:   "test name",
				Detail: "test detail",
				Data:   "test data",
				Destination0: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination1: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination2: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
				Destination3: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000004",
				},
				Destination4: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000005",
				},
			},

			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("e3097653-4c68-4915-add3-78b12a4ba151"),
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

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/outdials/"+tt.outdialID.String()+"/targets", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialtargetCreate(req.Context(), &tt.agent, tt.outdialID, tt.requestBody.Name, tt.requestBody.Detail, tt.requestBody.Data, tt.requestBody.Destination0, tt.requestBody.Destination1, tt.requestBody.Destination2, tt.requestBody.Destination3, tt.requestBody.Destination4).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDTargetsIDGET(t *testing.T) {

	tests := []struct {
		name            string
		agent           amagent.Agent
		outdialID       uuid.UUID
		outdialtargetID uuid.UUID

		response *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("112950f8-e3d3-4585-b858-125a59f8f51f"),
			uuid.FromStringOrNil("86a52dde-c523-11ec-a8b0-53d9628a5d7f"),

			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("86a52dde-c523-11ec-a8b0-53d9628a5d7f"),
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

			req, _ := http.NewRequest("GET", "/v1.0/outdials/"+tt.outdialID.String()+"/targets/"+tt.outdialtargetID.String(), nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialtargetGet(req.Context(), &tt.agent, tt.outdialID, tt.outdialtargetID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDTargetsIDDELETE(t *testing.T) {

	tests := []struct {
		name            string
		agent           amagent.Agent
		outdialID       uuid.UUID
		outdialtargetID uuid.UUID

		response *omoutdialtarget.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("112950f8-e3d3-4585-b858-125a59f8f51f"),
			uuid.FromStringOrNil("0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e"),

			&omoutdialtarget.WebhookMessage{
				ID: uuid.FromStringOrNil("0adb2487-eea7-4ec9-bb7f-b2b2aa5af49e"),
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

			req, _ := http.NewRequest("DELETE", "/v1.0/outdials/"+tt.outdialID.String()+"/targets/"+tt.outdialtargetID.String(), nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutdialtargetDelete(req.Context(), &tt.agent, tt.outdialID, tt.outdialtargetID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outdialsIDTargetGET(t *testing.T) {
	type test struct {
		name string

		agent     amagent.Agent
		reqQuery  string
		reqBody   request.ParamOutdialsIDTargetsGET
		outdialID uuid.UUID

		resOutdialtargets []*omoutdialtarget.WebhookMessage
		expectRes         string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			"/v1.0/outdials/fe7a06b6-c82c-11ec-89fd-f741623099f0/targets?page_size=10&page_token=2020-09-20%2003:23:21.995000",
			request.ParamOutdialsIDTargetsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:21.995000",
				},
			},
			uuid.FromStringOrNil("fe7a06b6-c82c-11ec-89fd-f741623099f0"),

			[]*omoutdialtarget.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("80fcacd4-c82c-11ec-b008-67e3b5299bec"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"80fcacd4-c82c-11ec-b008-67e3b5299bec","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			"/v1.0/outdials/33d8b93c-c82e-11ec-b630-f304b7d48448/targets?page_size=15&page_token=2020-09-20%2003:23:21.995000",
			request.ParamOutdialsIDTargetsGET{
				Pagination: request.Pagination{
					PageSize:  15,
					PageToken: "2020-09-20 03:23:21.995000",
				},
			},
			uuid.FromStringOrNil("33d8b93c-c82e-11ec-b630-f304b7d48448"),

			[]*omoutdialtarget.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("340757d8-c82e-11ec-92ef-235422080f76"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("34353180-c82e-11ec-b8f2-87eaa2dc5a1b"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("61f53c3c-c82e-11ec-ba3d-f387359c8014"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"340757d8-c82e-11ec-92ef-235422080f76","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"34353180-c82e-11ec-b8f2-87eaa2dc5a1b","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"61f53c3c-c82e-11ec-ba3d-f387359c8014","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockSvc.EXPECT().OutdialtargetGetsByOutdialID(req.Context(), &tt.agent, tt.outdialID, tt.reqBody.PageSize, tt.reqBody.PageToken).Return(tt.resOutdialtargets, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body.String())
			}
		})
	}
}
