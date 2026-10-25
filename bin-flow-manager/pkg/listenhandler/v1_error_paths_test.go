package listenhandler

import (
	"encoding/json"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	pkgerrors "github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

const (
	testErrID  = "0d58c9cc-ccfd-11ec-8807-cb5ce3bc2a68"
	testErrID2 = "1e69dadd-ddfe-22fd-9918-dc6df4cd3b79"
)

var errSynthetic = errors.New("synthetic failure")

type errPathMocks struct {
	flow       *flowhandler.MockFlowHandler
	activeflow *activeflowhandler.MockActiveflowHandler
	variable   *variablehandler.MockVariableHandler
}

// Test_processRequest_invalidBody verifies that a malformed JSON body is rejected with 400
// before any business handler is called. No handler expectation is registered, so a call
// into a business handler fails the test through the gomock controller.
func Test_processRequest_invalidBody(t *testing.T) {
	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"activeflows get", sock.RequestMethodGet, "/v1/activeflows?page_size=10"},
		{"activeflows post", sock.RequestMethodPost, "/v1/activeflows"},
		{"activeflows next", sock.RequestMethodGet, "/v1/activeflows/" + testErrID + "/next"},
		{"activeflows forward action id", sock.RequestMethodPut, "/v1/activeflows/" + testErrID + "/forward_action_id"},
		{"activeflows continue", sock.RequestMethodPost, "/v1/activeflows/" + testErrID + "/continue"},
		{"activeflows add actions", sock.RequestMethodPost, "/v1/activeflows/" + testErrID + "/add_actions"},
		{"activeflows push actions", sock.RequestMethodPost, "/v1/activeflows/" + testErrID + "/push_actions"},
		{"activeflows service stop", sock.RequestMethodPost, "/v1/activeflows/" + testErrID + "/service_stop"},
		{"flows count by customer", sock.RequestMethodGet, "/v1/flows/count_by_customer"},
		{"flows post", sock.RequestMethodPost, "/v1/flows"},
		{"flows get", sock.RequestMethodGet, "/v1/flows?page_size=10"},
		{"flows id put", sock.RequestMethodPut, "/v1/flows/" + testErrID},
		{"flows id actions put", sock.RequestMethodPut, "/v1/flows/" + testErrID + "/actions"},
		{"variables set", sock.RequestMethodPost, "/v1/variables/" + testErrID + "/variables"},
		{"variables substitute", sock.RequestMethodPost, "/v1/variables/" + testErrID + "/substitute"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				sockHandler:       sockhandler.NewMockSockHandler(mc),
				flowHandler:       flowhandler.NewMockFlowHandler(mc),
				activeflowHandler: activeflowhandler.NewMockActiveflowHandler(mc),
				variableHandler:   variablehandler.NewMockVariableHandler(mc),
			}

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   tt.method,
				DataType: "application/json",
				Data:     []byte(`{"broken":`),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 400 {
				t.Errorf("Wrong status code. expect: 400, got: %d", res.StatusCode)
			}
			if len(res.Data) != 0 || res.DataType != "" {
				t.Errorf("Wrong body. expect: empty, got: %s (%s)", res.Data, res.DataType)
			}
		})
	}
}

// Test_processRequest_filterConversionFailure verifies the list endpoints answer 400
// directly when the filter map cannot be converted to the resource's field types.
func Test_processRequest_filterConversionFailure(t *testing.T) {
	tests := []struct {
		name string
		uri  string
	}{
		{"activeflows", "/v1/activeflows?page_size=10"},
		{"flows", "/v1/flows?page_size=10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{
				flowHandler:       flowhandler.NewMockFlowHandler(mc),
				activeflowHandler: activeflowhandler.NewMockActiveflowHandler(mc),
			}

			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"not-a-uuid"}`),
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("Wrong status code. expect: 400, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processRequest_handlerError verifies that an error from the business handler is mapped
// by processRequest: plain error 400, ErrNotFound (wrapped) 404, typed error with its own code.
func Test_processRequest_handlerError(t *testing.T) {
	notFound := pkgerrors.Wrap(dbhandler.ErrNotFound, "could not get the resource")
	typed := cerrors.NotFound(commonoutline.ServiceNameFlowManager, "FLOW_NOT_FOUND", "The flow was not found.")

	id := uuid.FromStringOrNil(testErrID)
	id2 := uuid.FromStringOrNil(testErrID2)

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		data   string

		setup func(m *errPathMocks, err error)
		err   error

		expectStatus int
	}{
		{
			name: "activeflows list", method: sock.RequestMethodGet, uri: "/v1/activeflows?page_size=10&page_token=tk", data: `{}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().List(gomock.Any(), "tk", uint64(10), gomock.Any()).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows create", method: sock.RequestMethodPost, uri: "/v1/activeflows", data: `{"id":"` + testErrID + `"}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().Create(gomock.Any(), id, uuid.Nil, gomock.Any(), uuid.Nil, uuid.Nil, uuid.Nil, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows get", method: sock.RequestMethodGet, uri: "/v1/activeflows/" + testErrID,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().Get(gomock.Any(), id).Return(nil, err)
			},
			err: notFound, expectStatus: 404,
		},
		{
			name: "activeflows delete", method: sock.RequestMethodDelete, uri: "/v1/activeflows/" + testErrID,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().Delete(gomock.Any(), id).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows next", method: sock.RequestMethodGet, uri: "/v1/activeflows/" + testErrID + "/next", data: `{"current_action_id":"` + testErrID2 + `"}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().ExecuteNextAction(gomock.Any(), id, id2).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows forward action id", method: sock.RequestMethodPut, uri: "/v1/activeflows/" + testErrID + "/forward_action_id", data: `{"forward_action_id":"` + testErrID2 + `","forward_now":true}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().SetForwardActionID(gomock.Any(), id, id2, true).Return(err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows stop", method: sock.RequestMethodPost, uri: "/v1/activeflows/" + testErrID + "/stop",
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().Stop(gomock.Any(), id).Return(nil, err)
			},
			err: notFound, expectStatus: 404,
		},
		{
			name: "activeflows push actions", method: sock.RequestMethodPost, uri: "/v1/activeflows/" + testErrID + "/push_actions", data: `{"actions":[{"type":"answer"}]}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().PushActions(gomock.Any(), id, []action.Action{{Type: action.TypeAnswer}}).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows add actions", method: sock.RequestMethodPost, uri: "/v1/activeflows/" + testErrID + "/add_actions", data: `{"actions":[{"type":"hangup"}]}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().AddActions(gomock.Any(), id, []action.Action{{Type: action.TypeHangup}}).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "activeflows service stop", method: sock.RequestMethodPost, uri: "/v1/activeflows/" + testErrID + "/service_stop", data: `{"service_id":"` + testErrID2 + `"}`,
			setup: func(m *errPathMocks, err error) {
				m.activeflow.EXPECT().ServiceStop(gomock.Any(), id, id2).Return(err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "flows create", method: sock.RequestMethodPost, uri: "/v1/flows", data: `{"name":"n1","detail":"d1"}`,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().Create(gomock.Any(), uuid.Nil, gomock.Any(), "n1", "d1", false, gomock.Any(), uuid.Nil).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "flows list", method: sock.RequestMethodGet, uri: "/v1/flows?page_size=5&page_token=tk", data: `{}`,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().List(gomock.Any(), "tk", uint64(5), gomock.Any()).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "flows get", method: sock.RequestMethodGet, uri: "/v1/flows/" + testErrID,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().Get(gomock.Any(), id).Return(nil, err)
			},
			err: notFound, expectStatus: 404,
		},
		{
			name: "flows update", method: sock.RequestMethodPut, uri: "/v1/flows/" + testErrID, data: `{"actions":[]}`,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().Update(gomock.Any(), id, nil, nil, gomock.Any(), nil).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "flows delete", method: sock.RequestMethodDelete, uri: "/v1/flows/" + testErrID,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().Delete(gomock.Any(), id).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "flows update actions", method: sock.RequestMethodPut, uri: "/v1/flows/" + testErrID + "/actions", data: `{"actions":[{"type":"answer"}]}`,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().UpdateActions(gomock.Any(), id, []action.Action{{Type: action.TypeAnswer}}).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "flows action get", method: sock.RequestMethodGet, uri: "/v1/flows/" + testErrID + "/actions/" + testErrID2,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().ActionGet(gomock.Any(), id, id2).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "variables get", method: sock.RequestMethodGet, uri: "/v1/variables/" + testErrID,
			setup: func(m *errPathMocks, err error) {
				m.variable.EXPECT().Get(gomock.Any(), id).Return(nil, err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "variables set", method: sock.RequestMethodPost, uri: "/v1/variables/" + testErrID + "/variables", data: `{"variables":{"k1":"v1"}}`,
			setup: func(m *errPathMocks, err error) {
				m.variable.EXPECT().SetVariable(gomock.Any(), id, map[string]string{"k1": "v1"}).Return(err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "variables delete key", method: sock.RequestMethodDelete, uri: "/v1/variables/" + testErrID + "/variables/key%201",
			setup: func(m *errPathMocks, err error) {
				m.variable.EXPECT().DeleteVariable(gomock.Any(), id, "key 1").Return(err)
			},
			err: errSynthetic, expectStatus: 400,
		},
		{
			name: "variables substitute", method: sock.RequestMethodPost, uri: "/v1/variables/" + testErrID + "/substitute", data: `{"data":"hello ${k1}"}`,
			setup: func(m *errPathMocks, err error) {
				m.variable.EXPECT().Substitute(gomock.Any(), id, "hello ${k1}").Return("", err)
			},
			err: notFound, expectStatus: 404,
		},
		{
			name: "flows count by customer plain error", method: sock.RequestMethodGet, uri: "/v1/flows/count_by_customer", data: `{"customer_id":"` + testErrID + `"}`,
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().CountByCustomerID(gomock.Any(), id).Return(0, err)
			},
			err: errSynthetic, expectStatus: 500,
		},
		{
			name: "flows direct hash regenerate not found", method: sock.RequestMethodPost, uri: "/v1/flows/" + testErrID + "/direct-hash-regenerate",
			setup: func(m *errPathMocks, err error) {
				m.flow.EXPECT().DirectHashRegenerate(gomock.Any(), id).Return(nil, err)
			},
			err: notFound, expectStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := &errPathMocks{
				flow:       flowhandler.NewMockFlowHandler(mc),
				activeflow: activeflowhandler.NewMockActiveflowHandler(mc),
				variable:   variablehandler.NewMockVariableHandler(mc),
			}
			h := &listenHandler{
				flowHandler:       m.flow,
				activeflowHandler: m.activeflow,
				variableHandler:   m.variable,
			}
			tt.setup(m, tt.err)

			var data []byte
			if tt.data != "" {
				data = []byte(tt.data)
			}
			res, err := h.processRequest(&sock.Request{
				URI:      tt.uri,
				Method:   tt.method,
				DataType: "application/json",
				Data:     data,
			})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status code. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if len(res.Data) != 0 || res.DataType != "" {
				t.Errorf("Wrong body. expect: empty, got: %s (%s)", res.Data, res.DataType)
			}
		})
	}

	// typed error keeps its own status code, data type and body
	t.Run("typed error passthrough", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockFlow := flowhandler.NewMockFlowHandler(mc)
		h := &listenHandler{flowHandler: mockFlow}
		mockFlow.EXPECT().Get(gomock.Any(), id).Return(nil, typed)

		res, err := h.processRequest(&sock.Request{
			URI:    "/v1/flows/" + testErrID,
			Method: sock.RequestMethodGet,
		})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.StatusCode != 404 {
			t.Errorf("Wrong status code. expect: 404, got: %d", res.StatusCode)
		}
		if res.DataType != "voipbin_error" {
			t.Errorf("Wrong data type. expect: voipbin_error, got: %s", res.DataType)
		}

		var body map[string]any
		if errUnmarshal := json.Unmarshal(res.Data, &body); errUnmarshal != nil {
			t.Fatalf("Could not decode the body. err: %v", errUnmarshal)
		}
		if body["reason"] != "FLOW_NOT_FOUND" || body["domain"] != "flow-manager" || body["message"] != "The flow was not found." {
			t.Errorf("Wrong body. got: %v", body)
		}
	})
}

// Test_v1_handlerError_keepsCause verifies the handlers return the lower handler's error
// unchanged or wrapped so the cause stays reachable with errors.Is.
func Test_v1_handlerError_keepsCause(t *testing.T) {
	id := uuid.FromStringOrNil(testErrID)

	tests := []struct {
		name  string
		setup func(m *errPathMocks)
		call  func(h *listenHandler) (*sock.Response, error)
	}{
		{
			name: "activeflows list wraps",
			setup: func(m *errPathMocks) {
				m.activeflow.EXPECT().List(gomock.Any(), "", uint64(0), gomock.Any()).Return(nil, errSynthetic)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsGet(t.Context(), &sock.Request{URI: "/v1/activeflows?", Method: sock.RequestMethodGet, Data: []byte(`{}`)})
			},
		},
		{
			name: "activeflows stop wraps",
			setup: func(m *errPathMocks) {
				m.activeflow.EXPECT().Stop(gomock.Any(), id).Return(nil, errSynthetic)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsIDStopPost(t.Context(), &sock.Request{URI: "/v1/activeflows/" + testErrID + "/stop", Method: sock.RequestMethodPost})
			},
		},
		{
			name: "activeflows service stop wraps",
			setup: func(m *errPathMocks) {
				m.activeflow.EXPECT().ServiceStop(gomock.Any(), id, uuid.Nil).Return(errSynthetic)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsIDServiceStopPost(t.Context(), &sock.Request{URI: "/v1/activeflows/" + testErrID + "/service_stop", Method: sock.RequestMethodPost, Data: []byte(`{}`)})
			},
		},
		{
			name: "flows get returns as is",
			setup: func(m *errPathMocks) {
				m.flow.EXPECT().Get(gomock.Any(), id).Return(nil, errSynthetic)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1FlowsIDGet(t.Context(), &sock.Request{URI: "/v1/flows/" + testErrID, Method: sock.RequestMethodGet})
			},
		},
		{
			name: "variables substitute returns as is",
			setup: func(m *errPathMocks) {
				m.variable.EXPECT().Substitute(gomock.Any(), id, "x").Return("", errSynthetic)
			},
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1VariablesIDSubstitutePost(t.Context(), &sock.Request{URI: "/v1/variables/" + testErrID + "/substitute", Method: sock.RequestMethodPost, Data: []byte(`{"data":"x"}`)})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := &errPathMocks{
				flow:       flowhandler.NewMockFlowHandler(mc),
				activeflow: activeflowhandler.NewMockActiveflowHandler(mc),
				variable:   variablehandler.NewMockVariableHandler(mc),
			}
			h := &listenHandler{flowHandler: m.flow, activeflowHandler: m.activeflow, variableHandler: m.variable}
			tt.setup(m)

			res, err := tt.call(h)
			if res != nil {
				t.Errorf("Wrong response. expect: nil, got: %v", res)
			}
			if !errors.Is(err, errSynthetic) {
				t.Errorf("Wrong error. expect the cause to be reachable, got: %v", err)
			}
		})
	}
}

func Test_Run_queueCreateFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errSynthetic)

	err := h.Run("test-queue", "delay")
	if !errors.Is(err, errSynthetic) {
		t.Errorf("Wrong error. expect the cause to be reachable, got: %v", err)
	}
}

func Test_NewListenHandler_wiring(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockFlow := flowhandler.NewMockFlowHandler(mc)
	mockActiveflow := activeflowhandler.NewMockActiveflowHandler(mc)
	mockVariable := variablehandler.NewMockVariableHandler(mc)

	res := NewListenHandler(mockSock, mockFlow, mockActiveflow, mockVariable)
	h, ok := res.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}

	if h.utilHandler == nil {
		t.Errorf("utilHandler is not set")
	}
	if h.sockHandler != mockSock || h.flowHandler != mockFlow || h.activeflowHandler != mockActiveflow || h.variableHandler != mockVariable {
		t.Errorf("The given handlers are not wired to the listen handler")
	}
}
