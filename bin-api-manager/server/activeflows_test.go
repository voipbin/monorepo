package server

import (
	"bytes"
	"encoding/json"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/models/flow_manager"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetActiveflows(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery       string
		resActiveflows []*fmactiveflow.WebhookMessage

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

			reqQuery: "/v1.0/activeflows",
			resActiveflows: []*fmactiveflow.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("c9f6c460-d3aa-11ef-9062-db42ef820bc8"),
				},
			},

			expectedPageSize:  100,
			expectedPageToken: "",
			expectedRes:       `{"result":[{"id":"c9f6c460-d3aa-11ef-9062-db42ef820bc8","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
		},
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("9b7a30c4-ab4e-11ef-9068-1b1141edabd3"),
				},
			},

			reqQuery: "/v1.0/activeflows?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			resActiveflows: []*fmactiveflow.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("ca5324bc-d3aa-11ef-b3a2-5f9b2297d0b5"),
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"ca5324bc-d3aa-11ef-b3a2-5f9b2297d0b5","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("cb8453ee-ab4e-11ef-9a1c-dfc505495abd"),
				},
			},

			reqQuery: "/v1.0/activeflows?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			resActiveflows: []*fmactiveflow.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("ca814ff4-d3aa-11ef-b654-4356ff1e24b8"),
				},
				{
					ID: uuid.FromStringOrNil("caaf28de-d3aa-11ef-acab-8364561636de"),
				},
				{
					ID: uuid.FromStringOrNil("d80bf73c-d3aa-11ef-9e3b-5327fa6fb18b"),
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20 03:23:20.995000",
			expectedRes:       `{"result":[{"id":"ca814ff4-d3aa-11ef-b654-4356ff1e24b8","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"caaf28de-d3aa-11ef-acab-8364561636de","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"d80bf73c-d3aa-11ef-9e3b-5327fa6fb18b","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			mockSvc.EXPECT().ActiveflowGets(req.Context(), &tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.resActiveflows, nil)

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

func Test_PostActiveflows(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  openapi_server.PostActiveflowsJSONBody

		response *fmactiveflow.WebhookMessage

		expectedActions []fmaction.Action
		expectedID      uuid.UUID
		expectedFlowID  uuid.UUID
		expectedRes     string
	}{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/activeflows",
			reqBody: openapi_server.PostActiveflowsJSONBody{
				Actions: &[]flow_manager.Action{
					{
						Id: "692de0d6-d3ab-11ef-a2cd-07af60d8bb91",
					},
				},
				FlowId: stringPtr("8917167e-d3ab-11ef-b322-b36809068d12"),
				Id:     stringPtr("88eaacce-d3ab-11ef-ac99-23f970b154a2"),
			},

			response: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("893ebb34-d3ab-11ef-90e4-f31b0ef8762a"),
			},

			expectedActions: []fmaction.Action{
				{
					ID: uuid.FromStringOrNil("692de0d6-d3ab-11ef-a2cd-07af60d8bb91"),
				},
			},
			expectedID:     uuid.FromStringOrNil("88eaacce-d3ab-11ef-ac99-23f970b154a2"),
			expectedFlowID: uuid.FromStringOrNil("8917167e-d3ab-11ef-b322-b36809068d12"),
			expectedRes:    `{"id":"893ebb34-d3ab-11ef-90e4-f31b0ef8762a","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
		},
		{
			name: "empty",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/activeflows",
			reqBody:  openapi_server.PostActiveflowsJSONBody{},

			response: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("8969817a-d3ab-11ef-b01b-c358a80962d3"),
			},

			expectedActions: []fmaction.Action{},
			expectedID:      uuid.Nil,
			expectedFlowID:  uuid.Nil,
			expectedRes:     `{"id":"8969817a-d3ab-11ef-b01b-c358a80962d3","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().ActiveflowCreate(
				req.Context(),
				&tt.agent,
				tt.expectedID,
				tt.expectedFlowID,
				tt.expectedActions,
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

func Test_GetActiveflowsId(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		resActiveflow *fmactiveflow.WebhookMessage

		expectedActiveflowID uuid.UUID
		expectedRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/v1.0/activeflows/31c3088a-cb2c-11ed-b323-2b5e8c1da422",
			resActiveflow: &fmactiveflow.WebhookMessage{
				ID:       uuid.FromStringOrNil("31c3088a-cb2c-11ed-b323-2b5e8c1da422"),
				TMCreate: "2020-09-20 03:23:21.995000",
			},

			expectedActiveflowID: uuid.FromStringOrNil("31c3088a-cb2c-11ed-b323-2b5e8c1da422"),
			expectedRes:          `{"id":"31c3088a-cb2c-11ed-b323-2b5e8c1da422","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"2020-09-20 03:23:21.995000","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().ActiveflowGet(req.Context(), &tt.agent, tt.expectedActiveflowID).Return(tt.resActiveflow, nil)
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

func Test_DeleteActiveflowsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		callID   uuid.UUID

		responseActiveflow *fmactiveflow.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/activeflows/8abf67b2-cb2c-11ed-997d-4ff8509599f7",
			uuid.FromStringOrNil("8abf67b2-cb2c-11ed-997d-4ff8509599f7"),

			&fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("8abf67b2-cb2c-11ed-997d-4ff8509599f7"),
			},

			`{"id":"8abf67b2-cb2c-11ed-997d-4ff8509599f7","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().ActiveflowDelete(req.Context(), &tt.agent, tt.callID).Return(tt.responseActiveflow, nil)

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

func Test_PostActiveflowsIdStop(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery     string
		activeflowID uuid.UUID

		responseActiveflow *fmactiveflow.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery:     "/v1.0/activeflows/da10d24c-cb2c-11ed-be08-1fca5d4747f4/stop",
			activeflowID: uuid.FromStringOrNil("da10d24c-cb2c-11ed-be08-1fca5d4747f4"),

			responseActiveflow: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("da10d24c-cb2c-11ed-be08-1fca5d4747f4"),
			},

			expectRes: `{"id":"da10d24c-cb2c-11ed-be08-1fca5d4747f4","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().ActiveflowStop(req.Context(), &tt.agent, tt.activeflowID).Return(tt.responseActiveflow, nil)

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
