package server

import (
	"bytes"
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetAccesskeys(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		resAccesskeys []*csaccesskey.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "empty request",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("9b7a30c4-ab4e-11ef-9068-1b1141edabd3"),
				},
			},

			reqQuery: "/v1.0/accesskeys",
			resAccesskeys: []*csaccesskey.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"),
					TMCreate: "2020-09-20 03:23:21.995000",
				},
			},

			expectedPageSize:  100,
			expectedPageToken: "",
			expectedRes:       `{"result":[{"id":"3bc539bc-c68b-11ec-b41f-0776699e7467","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20 03:23:21.995000"}],"next_page_token":"2020-09-20 03:23:21.995000"}`,
		},
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("9b7a30c4-ab4e-11ef-9068-1b1141edabd3"),
				},
			},

			reqQuery: "/v1.0/accesskeys?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			resAccesskeys: []*csaccesskey.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467"),
					TMCreate: "2020-09-20 03:23:21.995000",
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"3bc539bc-c68b-11ec-b41f-0776699e7467","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20 03:23:21.995000"}],"next_page_token":"2020-09-20 03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("cb8453ee-ab4e-11ef-9a1c-dfc505495abd"),
				},
			},

			reqQuery: "/v1.0/accesskeys?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			resAccesskeys: []*csaccesskey.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("3bfa9cc4-c68b-11ec-a1cf-5fffd85773bb"),
					TMCreate: "2020-09-20 03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("3c2648d8-c68b-11ec-a47f-7bfbe26dbdcf"),
					TMCreate: "2020-09-20 03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("3c4d9a1e-c68b-11ec-8b46-5f282fd0eb19"),
					TMCreate: "2020-09-20 03:23:23.995000",
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"3bfa9cc4-c68b-11ec-a1cf-5fffd85773bb","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20 03:23:21.995000"},{"id":"3c2648d8-c68b-11ec-a47f-7bfbe26dbdcf","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20 03:23:22.995000"},{"id":"3c4d9a1e-c68b-11ec-8b46-5f282fd0eb19","customer_id":"00000000-0000-0000-0000-000000000000","token":"","tm_create":"2020-09-20 03:23:23.995000"}],"next_page_token":"2020-09-20 03:23:23.995000"}`,
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
			v1 := r.RouterGroup.Group("v1.0")
			openapi_server.RegisterHandlers(v1, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AccesskeyGets(req.Context(), &tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.resAccesskeys, nil)

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

func Test_PostAccesskeys(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  openapi_server.PostAccesskeysJSONBody

		response *csaccesskey.WebhookMessage

		expectedName   string
		expectedDetail string
		expectedExpire int32
		expectedRes    string
	}{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/accesskeys",
			reqBody: openapi_server.PostAccesskeysJSONBody{
				Name:   stringPtr("test name"),
				Detail: stringPtr("test detail"),
				Expire: intPtr(86400000),
			},

			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("18e018ae-ab4e-11ef-8be3-9b5666a5e592"),
			},

			expectedName:   "test name",
			expectedDetail: "test detail",
			expectedExpire: 86400000,
			expectedRes:    `{"id":"18e018ae-ab4e-11ef-8be3-9b5666a5e592","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
		},
		{
			name: "empty",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/accesskeys",
			reqBody:  openapi_server.PostAccesskeysJSONBody{},

			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("4684c4c0-d363-11ef-acc4-f31a48cce971"),
			},

			expectedName:   "",
			expectedDetail: "",
			expectedExpire: 0,
			expectedRes:    `{"id":"4684c4c0-d363-11ef-acc4-f31a48cce971","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			v1 := r.RouterGroup.Group("v1.0")
			openapi_server.RegisterHandlers(v1, h)

			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AccesskeyCreate(
				req.Context(),
				&tt.agent,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedExpire,
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

func Test_GetAccesskeysId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		response *csaccesskey.WebhookMessage

		expectedAccesskeyID uuid.UUID
		expectRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/accesskeys/14a2e40a-ab4f-11ef-a837-63a93a15cd69",

			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("14a2e40a-ab4f-11ef-a837-63a93a15cd69"),
			},

			expectedAccesskeyID: uuid.FromStringOrNil("14a2e40a-ab4f-11ef-a837-63a93a15cd69"),
			expectRes:           `{"id":"14a2e40a-ab4f-11ef-a837-63a93a15cd69","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			v1 := r.RouterGroup.Group("v1.0")
			openapi_server.RegisterHandlers(v1, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().AccesskeyGet(req.Context(), &tt.agent, tt.expectedAccesskeyID).Return(tt.response, nil)

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

func Test_DeleteAccesskeysId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		response *csaccesskey.WebhookMessage

		expectAccesskeyID uuid.UUID
		expectRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("35ffcfdc-ab4f-11ef-a110-e348ca351ef1"),
				},
			},

			reqQuery: "/v1.0/accesskeys/3629227e-ab4f-11ef-bcdb-ebc17777d865",

			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("3629227e-ab4f-11ef-bcdb-ebc17777d865"),
			},

			expectAccesskeyID: uuid.FromStringOrNil("3629227e-ab4f-11ef-bcdb-ebc17777d865"),
			expectRes:         `{"id":"3629227e-ab4f-11ef-bcdb-ebc17777d865","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			v1 := r.RouterGroup.Group("v1.0")
			openapi_server.RegisterHandlers(v1, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().AccesskeyDelete(req.Context(), &tt.agent, tt.expectAccesskeyID).Return(tt.response, nil)
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

func Test_PutAccesskeysId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		reqBody  request.BodyAccesskeysIDPUT
		response *csaccesskey.WebhookMessage

		expectedAccesskeyID uuid.UUID
		expectedName        string
		expectedDetail      string
		expectedRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e"),
				},
			},

			reqQuery: "/v1.0/accesskeys/a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e",

			reqBody: request.BodyAccesskeysIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},
			response: &csaccesskey.WebhookMessage{
				ID: uuid.FromStringOrNil("a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e"),
			},

			expectedAccesskeyID: uuid.FromStringOrNil("a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e"),
			expectedName:        "test name",
			expectedDetail:      "test detail",
			expectedRes:         `{"id":"a1a28ef0-ab4f-11ef-bf8c-4f4d983fb85e","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`,
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
			v1 := r.RouterGroup.Group("v1.0")
			openapi_server.RegisterHandlers(v1, h)

			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().AccesskeyUpdate(req.Context(), &tt.agent, tt.expectedAccesskeyID, tt.expectedName, tt.expectedDetail).Return(tt.response, nil)

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
