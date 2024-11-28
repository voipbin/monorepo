package outplans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	caoutplan "monorepo/bin-campaign-manager/models/outplan"

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

func Test_outplansPOST(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		requestBody request.BodyOutplansPOST

		response *caoutplan.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.BodyOutplansPOST{
				Name:   "test name",
				Detail: "test detail",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialTimeout:  30000,
				TryInterval:  600000,
				MaxTryCount0: 5,
				MaxTryCount1: 5,
				MaxTryCount2: 5,
				MaxTryCount3: 5,
				MaxTryCount4: 5,
			},

			&caoutplan.WebhookMessage{
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
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/outplans", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutplanCreate(
				req.Context(),
				&tt.agent,
				tt.requestBody.Name,
				tt.requestBody.Detail,
				tt.requestBody.Source,
				tt.requestBody.DialTimeout,
				tt.requestBody.TryInterval,
				tt.requestBody.MaxTryCount0,
				tt.requestBody.MaxTryCount1,
				tt.requestBody.MaxTryCount2,
				tt.requestBody.MaxTryCount3,
				tt.requestBody.MaxTryCount4,
			).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outplansGET(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		req         request.ParamOutplansGET
		resOutplans []*caoutplan.WebhookMessage
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
			request.ParamOutplansGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*caoutplan.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("891dceb2-c64b-11ec-ad40-4f3b7ab8bd4e"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"891dceb2-c64b-11ec-ad40-4f3b7ab8bd4e","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.ParamOutplansGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*caoutplan.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("b85b50fa-c64b-11ec-a17f-fb6cd8c28a0d"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("b88bd6f8-c64b-11ec-a895-0f50245da5a9"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("b8c11570-c64b-11ec-82f7-abb0350c1d7d"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"b85b50fa-c64b-11ec-a17f-fb6cd8c28a0d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"b88bd6f8-c64b-11ec-a895-0f50245da5a9","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"b8c11570-c64b-11ec-82f7-abb0350c1d7d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			reqQuery := fmt.Sprintf("/v1.0/outplans?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().OutplanGetsByCustomerID(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.resOutplans, nil)

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

func Test_outplansIDGET(t *testing.T) {

	tests := []struct {
		name      string
		agent     amagent.Agent
		outplanID uuid.UUID

		response *caoutplan.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("1b27088c-c64c-11ec-b7df-b37c8b4c4c13"),

			&caoutplan.WebhookMessage{
				ID: uuid.FromStringOrNil("1b27088c-c64c-11ec-b7df-b37c8b4c4c13"),
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

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/outplans/%s", tt.outplanID), nil)
			mockSvc.EXPECT().OutplanGet(req.Context(), &tt.agent, tt.outplanID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outplansIDDELETE(t *testing.T) {

	tests := []struct {
		name      string
		agent     amagent.Agent
		outplanID uuid.UUID

		response *caoutplan.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("3b58765e-c64c-11ec-a2c1-03acafdff2d7"),
			&caoutplan.WebhookMessage{
				ID: uuid.FromStringOrNil("3b58765e-c64c-11ec-a2c1-03acafdff2d7"),
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

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/outplans/%s", tt.outplanID), nil)
			mockSvc.EXPECT().OutplanDelete(req.Context(), &tt.agent, tt.outplanID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outplansIDPUT(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		outplanID   uuid.UUID
		requestBody request.BodyOutplansIDPUT
		response    *caoutplan.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("5ad57130-c64c-11ec-b131-a787ac641f8a"),
			request.BodyOutplansIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			&caoutplan.WebhookMessage{
				ID: uuid.FromStringOrNil("5ad57130-c64c-11ec-b131-a787ac641f8a"),
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

			req, _ := http.NewRequest("PUT", "/v1.0/outplans/"+tt.outplanID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutplanUpdateBasicInfo(req.Context(), &tt.agent, tt.outplanID, tt.requestBody.Name, tt.requestBody.Detail).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_outplansIDDialInfoPUT(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		outplanID   uuid.UUID
		requestBody request.BodyOutplansIDDialInfoPUT
		response    *caoutplan.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("d94e07e8-c64c-11ec-9e9d-8b700336c5ef"),
			request.BodyOutplansIDDialInfoPUT{
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialTimeout:  30000,
				TryInterval:  600000,
				MaxTryCount0: 5,
				MaxTryCount1: 5,
				MaxTryCount2: 5,
				MaxTryCount3: 5,
				MaxTryCount4: 5,
			},
			&caoutplan.WebhookMessage{
				ID: uuid.FromStringOrNil("d94e07e8-c64c-11ec-9e9d-8b700336c5ef"),
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

			req, _ := http.NewRequest("PUT", "/v1.0/outplans/"+tt.outplanID.String()+"/dial_info", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().OutplanUpdateDialInfo(
				req.Context(),
				&tt.agent,
				tt.outplanID,
				tt.requestBody.Source,
				tt.requestBody.DialTimeout,
				tt.requestBody.TryInterval,
				tt.requestBody.MaxTryCount0,
				tt.requestBody.MaxTryCount1,
				tt.requestBody.MaxTryCount2,
				tt.requestBody.MaxTryCount3,
				tt.requestBody.MaxTryCount4,
			).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
