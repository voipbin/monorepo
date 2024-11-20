package calls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	fmaction "monorepo/bin-flow-manager/models/action"

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

func Test_CallsPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent
		req   request.BodyCallsPOST

		responseCalls      []*cmcall.WebhookMessage
		responseGroupcalls []*cmgroupcall.WebhookMessage
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
			req: request.BodyCallsPOST{
				Source: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "source@test.voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeSIP,
						Target: "destination@test.voipbin.net",
					},
				},
				Actions: []fmaction.Action{},
			},

			responseCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("98b963ac-8df9-11ec-b26b-031d30ff93df"),
					},
				},
			},
			responseGroupcalls: []*cmgroupcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("37d675b5-83b2-4a78-8ed4-fe680ec41060"),
					},
				},
			},

			expectRes: `{"calls":[{"id":"98b963ac-8df9-11ec-b26b-031d30ff93df","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}],"groupcalls":[{"id":"37d675b5-83b2-4a78-8ed4-fe680ec41060","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","status":"","flow_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"answer_groupcall_id":"00000000-0000-0000-0000-000000000000","groupcall_ids":null,"call_count":0,"groupcall_count":0,"tm_create":"","tm_update":"","tm_delete":""}]}`,
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

			req, _ := http.NewRequest("POST", "/v1.0/calls", bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallCreate(req.Context(), &tt.agent, tt.req.FlowID, tt.req.Actions, &tt.req.Source, tt.req.Destinations).Return(tt.responseCalls, tt.responseGroupcalls, nil)

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

func Test_CallsGET(t *testing.T) {

	type test struct {
		name      string
		agent     amagent.Agent
		req       request.ParamCallsGET
		resCalls  []*cmcall.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},
			request.ParamCallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},
			request.ParamCallsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20T03:23:20.995000",
				},
			},
			[]*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("668e6ee6-f989-11ea-abca-bf1ca885b142"),
					},
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5d8167e0-f989-11ea-8b34-2b0a03c78fc5"),
					},
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("61c6626a-f989-11ea-abbf-97944933fee9"),
					},
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"668e6ee6-f989-11ea-abca-bf1ca885b142","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"5d8167e0-f989-11ea-8b34-2b0a03c78fc5","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"61c6626a-f989-11ea-abbf-97944933fee9","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			reqQuery := fmt.Sprintf("/v1.0/calls?page_size=%d&page_token=%s", tt.req.PageSize, tt.req.PageToken)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CallGets(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.resCalls, nil)

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

func Test_CallsIDGET(t *testing.T) {

	type test struct {
		name    string
		agent   amagent.Agent
		resCall *cmcall.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},
			&cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
				},
				TMCreate: "2020-09-20T03:23:21.995000",
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

			reqQuery := fmt.Sprintf("/v1.0/calls/%s", tt.resCall.ID)
			req, _ := http.NewRequest("GET", reqQuery, nil)

			mockSvc.EXPECT().CallGet(req.Context(), &tt.agent, tt.resCall.ID).Return(tt.resCall, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.resCall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_callsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		callID   uuid.UUID

		responseCall *cmcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/calls/72709904-719c-11ed-94f7-b78b75ad5dce",
			uuid.FromStringOrNil("72709904-719c-11ed-94f7-b78b75ad5dce"),

			&cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72709904-719c-11ed-94f7-b78b75ad5dce"),
				},
			},

			`{"id":"72709904-719c-11ed-94f7-b78b75ad5dce","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CallDelete(req.Context(), &tt.agent, tt.callID).Return(tt.responseCall, nil)

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

func Test_callsIDHangupPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		callID   uuid.UUID

		responseCall *cmcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/calls/09b9bf4c-8927-11ed-b16c-5719373564c9/hangup",
			uuid.FromStringOrNil("09b9bf4c-8927-11ed-b16c-5719373564c9"),

			&cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("09b9bf4c-8927-11ed-b16c-5719373564c9"),
				},
			},

			`{"id":"09b9bf4c-8927-11ed-b16c-5719373564c9","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","type":"","master_call_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":null,"recording_id":"00000000-0000-0000-0000-000000000000","recording_ids":null,"groupcall_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"destination":{"type":"","target":"","target_name":"","name":"","detail":""},"status":"","action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":""},"direction":"","mute_direction":"","hangup_by":"","hangup_reason":"","tm_progressing":"","tm_ringing":"","tm_hangup":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().CallHangup(req.Context(), &tt.agent, tt.callID).Return(tt.responseCall, nil)

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

func Test_CallsIDTalkPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyCallsIDTalkPOST

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/ed229366-a4b7-11ed-bfe7-b38647d68a3d/talk",
			request.BodyCallsIDTalkPOST{
				Text:     "hello world",
				Gender:   "female",
				Language: "en-US",
			},

			uuid.FromStringOrNil("ed229366-a4b7-11ed-bfe7-b38647d68a3d"),
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
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallTalk(req.Context(), &tt.agent, tt.expectCallID, tt.reqBody.Text, tt.reqBody.Gender, tt.reqBody.Language).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDHoldPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/eb763a4a-cf0f-11ed-a989-8fbebcdb62c2/hold",

			uuid.FromStringOrNil("eb763a4a-cf0f-11ed-a989-8fbebcdb62c2"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallHoldOn(req.Context(), &tt.agent, tt.expectCallID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDHoldDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/ebbb2f06-cf0f-11ed-be2c-27600beaf155/hold",

			uuid.FromStringOrNil("ebbb2f06-cf0f-11ed-be2c-27600beaf155"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallHoldOff(req.Context(), &tt.agent, tt.expectCallID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDMutePOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqBody  []byte
		reqQuery string

		expectCallID    uuid.UUID
		expectDirection cmcall.MuteDirection
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			[]byte(`{"direction":"both"}`),
			"/v1.0/calls/ebeb87b4-cf0f-11ed-bd36-5f06aa4155f5/mute",

			uuid.FromStringOrNil("ebeb87b4-cf0f-11ed-bd36-5f06aa4155f5"),
			cmcall.MuteDirectionBoth,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallMuteOn(req.Context(), &tt.agent, tt.expectCallID, tt.expectDirection).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDMuteDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		expectCallID    uuid.UUID
		expectDirection cmcall.MuteDirection
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/97b7fadc-cf10-11ed-b07a-7b718bb9eef9/mute",
			[]byte(`{"direction":"both"}`),

			uuid.FromStringOrNil("97b7fadc-cf10-11ed-b07a-7b718bb9eef9"),
			cmcall.MuteDirectionBoth,
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallMuteOff(req.Context(), &tt.agent, tt.expectCallID, tt.expectDirection).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDMOHPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/4c72ec78-d13e-11ed-b853-cff593bdd1af/moh",

			uuid.FromStringOrNil("4c72ec78-d13e-11ed-b853-cff593bdd1af"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallMOHOn(req.Context(), &tt.agent, tt.expectCallID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDMOHDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/4cb3b1ae-d13e-11ed-a27d-f78da612d3c4/moh",

			uuid.FromStringOrNil("4cb3b1ae-d13e-11ed-a27d-f78da612d3c4"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallMOHOff(req.Context(), &tt.agent, tt.expectCallID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDSilencePOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/836320ae-d13e-11ed-9b0c-efff68751c5a/silence",

			uuid.FromStringOrNil("836320ae-d13e-11ed-9b0c-efff68751c5a"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallSilenceOn(req.Context(), &tt.agent, tt.expectCallID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_CallsIDSilenceDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/839a7b62-d13e-11ed-9448-e71729a96494/silence",

			uuid.FromStringOrNil("839a7b62-d13e-11ed-9448-e71729a96494"),
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallSilenceOff(req.Context(), &tt.agent, tt.expectCallID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_callsIDMediaStreamGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectCallID        uuid.UUID
		expectEncapsulation string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			"/v1.0/calls/906c71fe-e922-11ee-808c-a721a8e44e90/media_stream?encapsulation=rtp",

			uuid.FromStringOrNil("906c71fe-e922-11ee-808c-a721a8e44e90"),
			"rtp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallMediaStreamStart(req.Context(), &tt.agent, tt.expectCallID, tt.expectEncapsulation, c.Writer, req).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
