package activeflows

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_activeflowsPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		req      request.BodyActiveflowsPOST

		responseActiveflow *fmactiveflow.WebhookMessage
		expectRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/v1.0/activeflows",
			req: request.BodyActiveflowsPOST{
				ID:     uuid.FromStringOrNil("06f74fac-c82f-11ee-bbfd-33ef6d9071f8"),
				FlowID: uuid.FromStringOrNil("073130b4-c82f-11ee-b396-6392e3537393"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
			},

			responseActiveflow: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("06f74fac-c82f-11ee-bbfd-33ef6d9071f8"),
			},
			expectRes: `{"id":"06f74fac-c82f-11ee-bbfd-33ef6d9071f8","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ActiveflowCreate(req.Context(), &tt.agent, tt.req.ID, tt.req.FlowID, tt.req.Actions).Return(tt.responseActiveflow, nil)

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

func Test_ActiveflowsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseActiveflows []*fmactiveflow.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44b678b0-8def-11ee-8112-23635969596b"),
				},
			},

			reqQuery: "/v1.0/activeflows?page_token=2020-09-20%2003:23:20.995000&page_size=10",
			responseActiveflows: []*fmactiveflow.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/v1.0/activeflows?page_token=2020-09-20%2003:23:20.995000&page_size=10",

			responseActiveflows: []*fmactiveflow.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("e4862b74-cb2b-11ed-ba51-afe7fc321764"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("e4b04fe4-cb2b-11ed-b3ac-3f2dfd3a3a7a"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("e4dd0886-cb2b-11ed-b7e6-e7e930615845"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"e4862b74-cb2b-11ed-ba51-afe7fc321764","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"e4b04fe4-cb2b-11ed-b3ac-3f2dfd3a3a7a","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"e4dd0886-cb2b-11ed-b7e6-e7e930615845","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","status":"","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","current_action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"forward_action_id":"00000000-0000-0000-0000-000000000000","executed_actions":null,"tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().ActiveflowGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseActiveflows, nil)

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

func Test_ActiveflowsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		resActiveflow *fmactiveflow.WebhookMessage

		expectActiveflowID uuid.UUID
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

			expectActiveflowID: uuid.FromStringOrNil("31c3088a-cb2c-11ed-b323-2b5e8c1da422"),
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

			mockSvc.EXPECT().ActiveflowGet(req.Context(), &tt.agent, tt.expectActiveflowID).Return(tt.resActiveflow, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.resActiveflow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_activeflowsIDDELETE(t *testing.T) {

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

func Test_activeflowsIDStopPOST(t *testing.T) {

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
