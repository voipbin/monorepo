package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	cmrecording "monorepo/bin-call-manager/models/recording"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CallsPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqBody []byte

		responseCalls      []*cmcall.WebhookMessage
		responseGroupcalls []*cmgroupcall.WebhookMessage

		expectFlowID       uuid.UUID
		expectActions      []fmaction.Action
		expectSource       *commonaddress.Address
		expectDestinations []commonaddress.Address
		expectRes          string
	}

	tests := []test{
		{
			name: "full data",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqBody: []byte(`{"source":{"type":"sip","target":"source@test.voipbin.net"},"destinations":[{"type":"sip","target":"destination@test.voipbin.net"}],"flow_id":"f0f80af2-d7c8-11ef-bc6a-03858a6b220f","actions":[{"type":"answer"}]}`),

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

			expectFlowID: uuid.FromStringOrNil("f0f80af2-d7c8-11ef-bc6a-03858a6b220f"),
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			expectSource: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "source@test.voipbin.net",
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
					Target: "destination@test.voipbin.net",
				},
			},
			expectRes: `{"calls":[{"id":"98b963ac-8df9-11ec-b26b-031d30ff93df","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null}}],"groupcalls":[{"id":"37d675b5-83b2-4a78-8ed4-fe680ec41060","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}]}`,
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

			req, _ := http.NewRequest("POST", "/calls", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallCreate(req.Context(), &tt.agent, tt.expectFlowID, tt.expectActions, tt.expectSource, tt.expectDestinations).Return(tt.responseCalls, tt.responseGroupcalls, nil)

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
		name  string
		agent amagent.Agent

		reqQuery string

		responseCalls []*cmcall.WebhookMessage

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

			reqQuery: "/calls?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null},"tm_create":"2020-09-20T03:23:21.995Z"}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 items",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseCalls: []*cmcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("668e6ee6-f989-11ea-abca-bf1ca885b142"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5d8167e0-f989-11ea-8b34-2b0a03c78fc5"),
					},
					TMCreate: timePtr("2020-09-20T03:23:22.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("61c6626a-f989-11ea-abbf-97944933fee9"),
					},
					TMCreate: timePtr("2020-09-20T03:23:23.995000Z"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"668e6ee6-f989-11ea-abca-bf1ca885b142","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null},"tm_create":"2020-09-20T03:23:21.995Z"},{"id":"5d8167e0-f989-11ea-8b34-2b0a03c78fc5","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null},"tm_create":"2020-09-20T03:23:22.995Z"},{"id":"61c6626a-f989-11ea-abbf-97944933fee9","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null},"tm_create":"2020-09-20T03:23:23.995Z"}],"next_page_token":"2020-09-20T03:23:23.995000Z"}`,
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

			mockSvc.EXPECT().CallList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseCalls, nil)

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
		name  string
		agent amagent.Agent

		reqQuery string

		responseCall *cmcall.WebhookMessage

		expectRes string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/395518ca-830a-11eb-badc-b3582bc51917",

			responseCall: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("395518ca-830a-11eb-badc-b3582bc51917"),
				},
				TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
			},

			expectRes: `{"id":"395518ca-830a-11eb-badc-b3582bc51917","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null},"tm_create":"2020-09-20T03:23:21.995Z"}`,
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

			mockSvc.EXPECT().CallGet(req.Context(), &tt.agent, tt.responseCall.ID).Return(tt.responseCall, nil)
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

func Test_callsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCall *cmcall.WebhookMessage

		expectCallID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/calls/72709904-719c-11ed-94f7-b78b75ad5dce",

			responseCall: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72709904-719c-11ed-94f7-b78b75ad5dce"),
				},
			},

			expectCallID: uuid.FromStringOrNil("72709904-719c-11ed-94f7-b78b75ad5dce"),
			expectRes:    `{"id":"72709904-719c-11ed-94f7-b78b75ad5dce","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null}}`,
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
			mockSvc.EXPECT().CallDelete(req.Context(), &tt.agent, tt.expectCallID).Return(tt.responseCall, nil)

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

		responseCall *cmcall.WebhookMessage

		expectCallID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/calls/09b9bf4c-8927-11ed-b16c-5719373564c9/hangup",

			responseCall: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("09b9bf4c-8927-11ed-b16c-5719373564c9"),
				},
			},

			expectCallID: uuid.FromStringOrNil("09b9bf4c-8927-11ed-b16c-5719373564c9"),
			expectRes:    `{"id":"09b9bf4c-8927-11ed-b16c-5719373564c9","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","recording_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","source":{},"destination":{},"action":{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","tm_execute":null}}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			mockSvc.EXPECT().CallHangup(req.Context(), &tt.agent, tt.expectCallID).Return(tt.responseCall, nil)

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
		reqBody  []byte

		expectCallID   uuid.UUID
		expectText     string
		expectGender   string
		expectLanguage string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/ed229366-a4b7-11ed-bfe7-b38647d68a3d/talk",
			reqBody:  []byte(`{"text":"hello world","gender":"female","language":"en-US"}`),

			expectCallID:   uuid.FromStringOrNil("ed229366-a4b7-11ed-bfe7-b38647d68a3d"),
			expectText:     "hello world",
			expectGender:   "female",
			expectLanguage: "en-US",
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallTalk(req.Context(), &tt.agent, tt.expectCallID, tt.expectText, tt.expectGender, tt.expectLanguage).Return(nil)

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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/eb763a4a-cf0f-11ed-a989-8fbebcdb62c2/hold",

			expectCallID: uuid.FromStringOrNil("eb763a4a-cf0f-11ed-a989-8fbebcdb62c2"),
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/ebbb2f06-cf0f-11ed-be2c-27600beaf155/hold",

			expectCallID: uuid.FromStringOrNil("ebbb2f06-cf0f-11ed-be2c-27600beaf155"),
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

		reqQuery string
		reqBody  []byte

		expectCallID    uuid.UUID
		expectDirection cmcall.MuteDirection
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/ebeb87b4-cf0f-11ed-bd36-5f06aa4155f5/mute",
			reqBody:  []byte(`{"direction":"both"}`),

			expectCallID:    uuid.FromStringOrNil("ebeb87b4-cf0f-11ed-bd36-5f06aa4155f5"),
			expectDirection: cmcall.MuteDirectionBoth,
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/97b7fadc-cf10-11ed-b07a-7b718bb9eef9/mute",
			reqBody:  []byte(`{"direction":"both"}`),

			expectCallID:    uuid.FromStringOrNil("97b7fadc-cf10-11ed-b07a-7b718bb9eef9"),
			expectDirection: cmcall.MuteDirectionBoth,
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/4c72ec78-d13e-11ed-b853-cff593bdd1af/moh",

			expectCallID: uuid.FromStringOrNil("4c72ec78-d13e-11ed-b853-cff593bdd1af"),
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/4cb3b1ae-d13e-11ed-a27d-f78da612d3c4/moh",

			expectCallID: uuid.FromStringOrNil("4cb3b1ae-d13e-11ed-a27d-f78da612d3c4"),
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/836320ae-d13e-11ed-9b0c-efff68751c5a/silence",

			expectCallID: uuid.FromStringOrNil("836320ae-d13e-11ed-9b0c-efff68751c5a"),
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/839a7b62-d13e-11ed-9448-e71729a96494/silence",

			expectCallID: uuid.FromStringOrNil("839a7b62-d13e-11ed-9448-e71729a96494"),
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
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
				},
			},

			reqQuery: "/calls/906c71fe-e922-11ee-808c-a721a8e44e90/media_stream?encapsulation=rtp",

			expectCallID:        uuid.FromStringOrNil("906c71fe-e922-11ee-808c-a721a8e44e90"),
			expectEncapsulation: "rtp",
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
			c, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

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

func Test_PostCallsIdRecordingStart(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseCall *cmcall.WebhookMessage

		expectedCallID      uuid.UUID
		expectedFormat      cmrecording.Format
		epxectEndOfSilence  int
		expectedEndOfKey    string
		expectedDuration    int
		expectedOnEndFlowID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d30e7702-0567-11f0-89b1-1ff587d35570"),
					CustomerID: uuid.FromStringOrNil("d33296b4-0567-11f0-8801-3f3e7d9612e6"),
				},
			},

			reqQuery: "/calls/d30e7702-0567-11f0-89b1-1ff587d35570/recording_start",
			reqBody:  []byte(`{"format":"wav","end_of_silence":10,"end_of_key":"1","duration":600,"on_end_flow_id":"d3578cb2-0567-11f0-82cf-5362e575afc8"}`),

			responseCall: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d30e7702-0567-11f0-89b1-1ff587d35570"),
				},
			},

			expectedCallID:      uuid.FromStringOrNil("d30e7702-0567-11f0-89b1-1ff587d35570"),
			expectedFormat:      cmrecording.FormatWAV,
			epxectEndOfSilence:  10,
			expectedEndOfKey:    "1",
			expectedDuration:    600,
			expectedOnEndFlowID: uuid.FromStringOrNil("d3578cb2-0567-11f0-82cf-5362e575afc8"),
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallRecordingStart(
				req.Context(),
				&tt.agent,
				tt.expectedCallID,
				tt.expectedFormat,
				tt.epxectEndOfSilence,
				tt.expectedEndOfKey,
				tt.expectedDuration,
				tt.expectedOnEndFlowID,
			).Return(tt.responseCall, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_PostCallsIdRecordingStop(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseCall *cmcall.WebhookMessage

		expectedCallID uuid.UUID
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee49e0c2-0569-11f0-83fc-b32453293d26"),
				},
			},

			reqQuery: "/calls/ee49e0c2-0569-11f0-83fc-b32453293d26/recording_stop",

			responseCall: &cmcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee49e0c2-0569-11f0-83fc-b32453293d26"),
				},
			},

			expectedCallID: uuid.FromStringOrNil("ee49e0c2-0569-11f0-83fc-b32453293d26"),
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().CallRecordingStop(req.Context(), &tt.agent, tt.expectedCallID).Return(tt.responseCall, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
