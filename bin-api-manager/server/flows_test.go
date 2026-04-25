package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostFlows(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseFlow *fmflow.WebhookMessage

		expectName             string
		expectDetail           string
		expectActions          []fmaction.Action
		expectOnCompleteFlowID uuid.UUID
		expectRes              string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/flows",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","actions":[{"type":"answer"}],"on_complete_flow_id":"15356302-cf93-11f0-82a4-4723dbd39b78"}`),

			responseFlow: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("264b18d4-82fa-11eb-919b-9f55a7f6ace1"),
				},
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			expectName:   "test name",
			expectDetail: "test detail",
			expectActions: []fmaction.Action{
				{
					Type: "answer",
				},
			},
			expectOnCompleteFlowID: uuid.FromStringOrNil("15356302-cf93-11f0-82a4-4723dbd39b78"),
			expectRes:              `{"id":"264b18d4-82fa-11eb-919b-9f55a7f6ace1","customer_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer","tm_execute":null}],"on_complete_flow_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().FlowCreate(req.Context(), tt.agent, tt.expectName, tt.expectDetail, tt.expectActions, tt.expectOnCompleteFlowID, true).Return(tt.responseFlow, nil)

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

func Test_GetFlows(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseFlows []*fmflow.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/flows?page_size=20&page_token=2020-09-20T03:23:20.995000Z",

			responseFlows: []*fmflow.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5af5346a-d92d-11ef-8c33-67a5ecb7e5e5"),
					},
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"5af5346a-d92d-11ef-8c33-67a5ecb7e5e5","customer_id":"00000000-0000-0000-0000-000000000000","on_complete_flow_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().FlowList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseFlows, nil)

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

func Test_GetFlowsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseFlow *fmflow.WebhookMessage

		expectFlowID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/flows/2375219e-0b87-11eb-90f9-036ec16f126b",

			responseFlow: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				},
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			expectFlowID: uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
			expectRes:    `{"id":"2375219e-0b87-11eb-90f9-036ec16f126b","customer_id":"00000000-0000-0000-0000-000000000000","name":"test name","detail":"test detail","actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer","tm_execute":null}],"on_complete_flow_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().FlowGet(req.Context(), tt.agent, tt.expectFlowID).Return(tt.responseFlow, nil)

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

func Test_PutFlowsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseFlow *fmflow.WebhookMessage

		expectFlowID           uuid.UUID
		expectName             string
		expectDetail           string
		expectActions          []fmaction.Action
		expectOnCompleteFlowID uuid.UUID
		expectRes              string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/flows/d213a09e-6790-11eb-8cea-bb3b333200ed",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","actions":[{"type":"answer"}],"on_complete_flow_id":"305ebdb8-cf93-11f0-9bd2-4386663663cd"}`),

			responseFlow: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
				},
			},

			expectFlowID: uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
			expectName:   "test name",
			expectDetail: "test detail",
			expectActions: []fmaction.Action{
				{
					Type: "answer",
				},
			},
			expectOnCompleteFlowID: uuid.FromStringOrNil("305ebdb8-cf93-11f0-9bd2-4386663663cd"),
			expectRes:              `{"id":"d213a09e-6790-11eb-8cea-bb3b333200ed","customer_id":"00000000-0000-0000-0000-000000000000","on_complete_flow_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().FlowUpdate(
				req.Context(),
				tt.agent,
				tt.expectFlowID,
				tt.expectName,
				tt.expectDetail,
				tt.expectActions,
				tt.expectOnCompleteFlowID,
			).Return(tt.responseFlow, nil)

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

// Test_flowsPOST_MissingAuthIdentity verifies PostFlows emits the
// canonical UNAUTHENTICATED / AUTHENTICATION_REQUIRED envelope when
// auth_identity is missing from the gin context.
func Test_flowsPOST_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodPost, "/flows",
		[]byte(`{"name":"test name","detail":"test detail","actions":[{"type":"answer"}]}`))
}

// Test_flowsPOST_InvalidJSONBody verifies PostFlows rejects malformed
// JSON with INVALID_ARGUMENT / INVALID_JSON_BODY.
func Test_flowsPOST_InvalidJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodPost, "/flows", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_JSON_BODY", commonoutline.ServiceNameAPIManager)
}

// Test_flowsIDGet_InvalidID verifies that a malformed UUID in the path
// triggers INVALID_ARGUMENT / INVALID_ID before the servicehandler is
// consulted.
func Test_flowsIDGet_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// "not-a-uuid" passes the path-shape check but uuid.FromStringOrNil
	// returns uuid.Nil, so the handler rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodGet, "/flows/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID", commonoutline.ServiceNameAPIManager)
}

// Test_flowsIDGet_ServiceError exercises the servicehandler-failure path
// through abortWithServiceError. The translator's substring fallback maps
// "flow not found" to NOT_FOUND / RESOURCE_NOT_FOUND.
func Test_flowsIDGet_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	flowID := uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodGet, "/flows/2375219e-0b87-11eb-90f9-036ec16f126b", nil)
	// The RequestID middleware augments the context, so match with gomock.Any().
	mockSvc.EXPECT().FlowGet(gomock.Any(), agent, flowID).Return(nil, fmt.Errorf("flow not found"))

	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND", commonoutline.ServiceNameAPIManager)
}

// Test_flowsIDDelete_MissingAuthIdentity exercises the
// auth-identity-missing branch of DeleteFlowsId.
func Test_flowsIDDelete_MissingAuthIdentity(t *testing.T) {
	assertMissingAuthIdentity(t, http.MethodDelete, "/flows/d466f900-67cb-11eb-b2ff-1f9adc48f842", nil)
}

func Test_DeleteFlowsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseFlow *fmflow.WebhookMessage

		expectFlowID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/flows/d466f900-67cb-11eb-b2ff-1f9adc48f842",

			responseFlow: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d466f900-67cb-11eb-b2ff-1f9adc48f842"),
				},
			},

			expectFlowID: uuid.FromStringOrNil("d466f900-67cb-11eb-b2ff-1f9adc48f842"),
			expectRes:    `{"id":"d466f900-67cb-11eb-b2ff-1f9adc48f842","customer_id":"00000000-0000-0000-0000-000000000000","on_complete_flow_id":"00000000-0000-0000-0000-000000000000","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().FlowDelete(req.Context(), tt.agent, tt.expectFlowID).Return(tt.responseFlow, nil)

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
