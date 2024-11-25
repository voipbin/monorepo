package accesskeys

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_accesskeysPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyAccesskeysPOST

		response *csaccesskey.WebhookMessage

		expectedRes string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/accesskeys",
			reqBody: request.BodyAccesskeysPOST{
				Name:   "test name",
				Detail: "test detail",
				Expire: 86400000,
			},

			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("18e018ae-ab4e-11ef-8be3-9b5666a5e592"),
			},

			expectedRes: `{"id":"18e018ae-ab4e-11ef-8be3-9b5666a5e592","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AccesskeyCreate(
				req.Context(),
				&tt.agent,
				tt.reqBody.Name,
				tt.reqBody.Detail,
				tt.reqBody.Expire,
			).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_accesskeysGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery       string
		reqBody        request.ParamAccesskeysGET
		resAccesskeys  []*csaccesskey.WebhookMessage
		expectedFilter map[string]string
		expectedRes    string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("9b7a30c4-ab4e-11ef-9068-1b1141edabd3"),
				},
			},

			"/v1.0/accesskeys?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamAccesskeysGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*csaccesskey.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			map[string]string{
				"customer_id": "9b7a30c4-ab4e-11ef-9068-1b1141edabd3",
				"deleted":     "false",
			},
			`{"result":[{"id":"3bc539bc-c68b-11ec-b41f-0776699e7467","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20T03:23:21.995000"}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("cb8453ee-ab4e-11ef-9a1c-dfc505495abd"),
				},
			},

			"/v1.0/accesskeys?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamAccesskeysGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},
			[]*csaccesskey.WebhookMessage{
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
			map[string]string{
				"customer_id": "cb8453ee-ab4e-11ef-9a1c-dfc505495abd",
				"deleted":     "false",
			},
			`{"result":[{"id":"3bfa9cc4-c68b-11ec-a1cf-5fffd85773bb","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20T03:23:21.995000"},{"id":"3c2648d8-c68b-11ec-a47f-7bfbe26dbdcf","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20T03:23:22.995000"},{"id":"3c4d9a1e-c68b-11ec-8b46-5f282fd0eb19","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20T03:23:23.995000"}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().AccesskeyGets(req.Context(), &tt.agent, tt.reqBody.PageSize, tt.reqBody.PageToken).Return(tt.resAccesskeys, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_accesskeysIDGet(t *testing.T) {

	tests := []struct {
		name        string
		agent       amagent.Agent
		accesskeyID uuid.UUID

		reqQuery string

		responseAccesskey *csaccesskey.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("14a2e40a-ab4f-11ef-a837-63a93a15cd69"),

			"/v1.0/accesskeys/14a2e40a-ab4f-11ef-a837-63a93a15cd69",

			&csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("14a2e40a-ab4f-11ef-a837-63a93a15cd69"),
			},

			`{"id":"14a2e40a-ab4f-11ef-a837-63a93a15cd69","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			mockSvc.EXPECT().AccesskeyGet(req.Context(), &tt.agent, tt.accesskeyID).Return(tt.responseAccesskey, nil)

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

func Test_accesskeysIDDelete(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery   string
		campaignID uuid.UUID

		responseCampaign *csaccesskey.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("35ffcfdc-ab4f-11ef-a110-e348ca351ef1"),
				},
			},

			"/v1.0/accesskeys/3629227e-ab4f-11ef-bcdb-ebc17777d865",
			uuid.FromStringOrNil("3629227e-ab4f-11ef-bcdb-ebc17777d865"),

			&csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("3629227e-ab4f-11ef-bcdb-ebc17777d865"),
			},

			`{"id":"3629227e-ab4f-11ef-bcdb-ebc17777d865","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			mockSvc.EXPECT().AccesskeyDelete(req.Context(), &tt.agent, tt.campaignID).Return(tt.responseCampaign, nil)
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

func Test_accesskeysIDPUT(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		accesskeyID uuid.UUID

		reqBody  request.BodyAccesskeysIDPUT
		response *csaccesskey.WebhookMessage
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e"),
				},
			},

			reqQuery:    "/v1.0/accesskeys/a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e",
			accesskeyID: uuid.FromStringOrNil("a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e"),

			reqBody: request.BodyAccesskeysIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e"),
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
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AccesskeyUpdate(req.Context(), &tt.agent, tt.accesskeyID, tt.reqBody.Name, tt.reqBody.Detail).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
