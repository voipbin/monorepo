package listenhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

// The sentinels below are fake values. They go into the request only, never into a
// response agent: the excluded "Sending result" line prints the agent JSON.
const (
	logSentinelA = "SENTINEL-A-MUST-NOT-LEAK-7c1d"
	logSentinelB = "SENTINEL-B-MUST-NOT-LEAK-52ab"
	logSentinelC = "SENTINEL-C-MUST-NOT-LEAK-91ef"

	logTestAgentID    = "f1ba04b0-951e-11ee-a0a2-7b8600a1ee45"
	logTestCustomerID = "5e4a0680-804e-11ec-8477-2fea5968d85b"
)

// newLogRedactionHook attaches a hook to the standard logger and enables the debug
// level, because most of the log calls under test are Debug and would otherwise be
// dropped, which would make the assertion pass vacuously. The tests that use it
// must not run in parallel.
func newLogRedactionHook(t *testing.T) *logrustest.Hook {
	t.Helper()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})
	return hook
}

// decimalForm returns the sentinel as space separated decimal byte values, which is how
// a json.RawMessage or []byte is printed by %v.
func decimalForm(s string) string {
	parts := make([]string, 0, len(s))
	for _, b := range []byte(s) {
		parts = append(parts, fmt.Sprint(b))
	}
	return strings.Join(parts, " ")
}

// assertNoSecretInEntries requires at least one entry whose message contains wantMsg and
// no entry that carries a secret, in plain text or as decimal bytes, in its message, in
// its JSON-marshaled fields (the production formatter) or in the %v of each field.
func assertNoSecretInEntries(t *testing.T, hook *logrustest.Hook, wantMsg string, secrets ...string) {
	t.Helper()

	found := 0
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, wantMsg) {
			found++
		}

		views := []string{entry.Message}
		if b, err := json.Marshal(entry.Data); err == nil {
			views = append(views, string(b))
		}
		for k, v := range entry.Data {
			views = append(views, k+"="+fmt.Sprintf("%v", v), k+"="+fmt.Sprintf("%+v", v))
		}

		for _, secret := range secrets {
			for _, view := range views {
				if strings.Contains(view, secret) || strings.Contains(view, decimalForm(secret)) {
					t.Errorf("secret leaked into a log entry. level: %s, message: %q", entry.Level, entry.Message)
				}
			}
		}
	}
	if found == 0 {
		t.Errorf("no log entry has the message %q, the check would be vacuous", wantMsg)
	}
}

func newLogRedactionHandler(mc *gomock.Controller) (*listenHandler, *agenthandler.MockAgentHandler) {
	mockAgent := agenthandler.NewMockAgentHandler(mc)
	h := &listenHandler{
		sockHandler:  sockhandler.NewMockSockHandler(mc),
		agentHandler: mockAgent,
	}
	return h, mockAgent
}

func Test_processRequest_logsDoNotContainBody(t *testing.T) {
	errBoom := errors.New("boom")
	okAgent := &agent.Agent{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil(logTestAgentID)}}

	tests := []struct {
		name string

		request  *sock.Request
		mockFunc func(m *agenthandler.MockAgentHandler)

		wantMsg        string
		expectParseErr bool
	}{
		{
			name: "login success",
			request: &sock.Request{
				URI:    "/v1/login",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"username":"user-a","password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().Login(gomock.Any(), "user-a", logSentinelA).Return(okAgent, nil)
			},
			wantMsg: "Received request.",
		},
		{
			name: "login failure reaches the response line",
			request: &sock.Request{
				URI:    "/v1/login",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"username":"user-a","password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().Login(gomock.Any(), "user-a", logSentinelA).Return(nil, errBoom)
			},
			wantMsg: "Sending response.",
		},
		{
			name: "login by username",
			request: &sock.Request{
				URI:    "/v1/agents/user-a/login",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().Login(gomock.Any(), "user-a", logSentinelA).Return(okAgent, nil)
			},
			wantMsg: "Received request.",
		},
		{
			name: "agent create",
			request: &sock.Request{
				URI:    "/v1/agents",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id":"` + logTestCustomerID + `","username":"user-a","password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().Create(gomock.Any(), uuid.FromStringOrNil(logTestCustomerID), "user-a", logSentinelA, "", "", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(okAgent, nil)
			},
			wantMsg: "Received request.",
		},
		{
			name: "password update",
			request: &sock.Request{
				URI:    "/v1/agents/" + logTestAgentID + "/password",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().UpdatePassword(gomock.Any(), uuid.FromStringOrNil(logTestAgentID), logSentinelA).Return(okAgent, nil)
			},
			wantMsg: "Received request.",
		},
		{
			name: "password reset",
			request: &sock.Request{
				URI:    "/v1/password-reset",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"token":"` + logSentinelB + `","password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().PasswordReset(gomock.Any(), logSentinelB, logSentinelA).Return(nil)
			},
			wantMsg: "Received request.",
		},
		{
			name: "count by customer carries the body on the function logger",
			request: &sock.Request{
				URI:    "/v1/agents/count_by_customer",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"` + logTestCustomerID + `","note":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().CountByCustomerID(gomock.Any(), uuid.FromStringOrNil(logTestCustomerID)).Return(0, errBoom)
			},
			wantMsg: "Could not get agent count.",
		},
		{
			name: "count by customer with a broken body",
			request: &sock.Request{
				URI:    "/v1/agents/count_by_customer",
				Method: sock.RequestMethodGet,
				Data:   []byte(`[1,"` + logSentinelA + `"]`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {},
			wantMsg:  "Could not unmarshal the request data.",
		},
		{
			name: "direct hash regenerate",
			request: &sock.Request{
				URI:    "/v1/agents/" + logTestAgentID + "/direct-hash-regenerate",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"note":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().DirectHashRegenerate(gomock.Any(), uuid.FromStringOrNil(logTestAgentID)).Return(okAgent, nil)
			},
			wantMsg: "Received request.",
		},
		{
			name: "unknown uri reaches the not found error line",
			request: &sock.Request{
				URI:    "/v1/unknown",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"password":"` + logSentinelA + `"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {},
			wantMsg:  "Could not find corresponded message handler.",
		},
		{
			name: "list with a broken uri reaches the handler error line",
			request: &sock.Request{
				URI:    "/%zz/v1/agents?a=1",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"password":"` + logSentinelA + `"}`),
			},
			mockFunc:       func(m *agenthandler.MockAgentHandler) {},
			wantMsg:        "Could not find corresponded message handler.",
			expectParseErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !bytes.Contains(tt.request.Data, []byte(logSentinelA)) {
				t.Fatalf("the request body must carry the sentinel")
			}
			if tt.expectParseErr {
				if _, err := url.Parse(tt.request.URI); err == nil {
					t.Fatalf("the uri must fail url.Parse to reach the error branch")
				}
			}

			hook := newLogRedactionHook(t)
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockAgent := newLogRedactionHandler(mc)
			tt.mockFunc(mockAgent)

			tt.request.DataType = "application/json"
			if _, err := h.processRequest(tt.request); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			assertNoSecretInEntries(t, hook, tt.wantMsg, logSentinelA, logSentinelB)
		})
	}
}

// Test_processRequest_receivedRequestFields checks the values of the allowlisted fields.
func Test_processRequest_receivedRequestFields(t *testing.T) {
	hook := newLogRedactionHook(t)
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockAgent := newLogRedactionHandler(mc)
	mockAgent.EXPECT().Login(gomock.Any(), "user-a", logSentinelA).Return(nil, errors.New("boom"))

	req := &sock.Request{
		URI:       "/v1/login",
		Method:    sock.RequestMethodPost,
		DataType:  "application/json",
		Data:      []byte(`{"username":"user-a","password":"` + logSentinelA + `"}`),
		RequestID: "req-1",
	}
	if _, err := h.processRequest(req); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var got *logrus.Entry
	for _, entry := range hook.AllEntries() {
		if strings.HasPrefix(entry.Message, "Received request.") {
			got = entry
			break
		}
	}
	if got == nil {
		t.Fatalf("no received request entry")
	}

	expect := map[string]interface{}{
		"uri":        "/v1/login",
		"method":     sock.RequestMethodPost,
		"data_type":  "application/json",
		"request_id": "req-1",
		"data_len":   len(req.Data),
	}
	for k, v := range expect {
		if got.Data[k] != v {
			t.Errorf("Wrong field %s. expect: %v, got: %v", k, v, got.Data[k])
		}
	}
	if _, ok := got.Data["request"]; ok {
		t.Errorf("the request object must not be a log field")
	}
}

// Test_processRequest_unmarshalFailureLogsDoNotContainBody covers every route whose
// body cannot be decoded. The body is valid JSON of the wrong shape so that the
// unfixed log fields are still marshalable and a leak is the only possible failure.
func Test_processRequest_unmarshalFailureLogsDoNotContainBody(t *testing.T) {
	body := []byte(`[1,"` + logSentinelA + `"]`)

	tests := []struct {
		name   string
		uri    string
		method sock.RequestMethod
	}{
		{"agent create", "/v1/agents", sock.RequestMethodPost},
		{"get by customer id address", "/v1/agents/get_by_customer_id_address", sock.RequestMethodPost},
		{"login by username", "/v1/agents/user-a/login", sock.RequestMethodPost},
		{"addresses", "/v1/agents/" + logTestAgentID + "/addresses", sock.RequestMethodPut},
		{"basic info", "/v1/agents/" + logTestAgentID, sock.RequestMethodPut},
		{"status", "/v1/agents/" + logTestAgentID + "/status", sock.RequestMethodPut},
		{"password", "/v1/agents/" + logTestAgentID + "/password", sock.RequestMethodPut},
		{"tag ids", "/v1/agents/" + logTestAgentID + "/tag_ids", sock.RequestMethodPut},
		{"permission", "/v1/agents/" + logTestAgentID + "/permission", sock.RequestMethodPut},
		{"reserve", "/v1/agents/" + logTestAgentID + "/reserve", sock.RequestMethodPost},
		{"reserve release", "/v1/agents/" + logTestAgentID + "/reserve_release", sock.RequestMethodPost},
		{"login", "/v1/login", sock.RequestMethodPost},
		{"password forgot", "/v1/password-forgot", sock.RequestMethodPost},
		{"password reset", "/v1/password-reset", sock.RequestMethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := newLogRedactionHook(t)
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newLogRedactionHandler(mc)

			req := &sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: body}
			res, err := h.processRequest(req)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 400 {
				t.Fatalf("Wrong status. the body must be rejected. expect: 400, got: %d", res.StatusCode)
			}

			assertNoSecretInEntries(t, hook, "Could not unmarshal the data.", logSentinelA)
		})
	}
}

// Test_processRequest_marshalFailureLogsDoNotContainPasswordHash covers the response
// marshal failure lines, which used to print the agent with %v.
// The list route is not covered: its failure line printed a slice of agent pointers, which
// %v renders as addresses only, so it never exposed the hash and cannot fail this check.
func Test_processRequest_marshalFailureLogsDoNotContainPasswordHash(t *testing.T) {
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		request *sock.Request
		expect  func(m *agenthandler.MockAgentHandler, bad *agent.Agent)
	}{
		{
			name:    "get",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID, Method: sock.RequestMethodGet},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "create",
			request: &sock.Request{URI: "/v1/agents", Method: sock.RequestMethodPost,
				Data: []byte(`{"customer_id":"` + logTestCustomerID + `","username":"user-a"}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name:    "delete",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID, Method: sock.RequestMethodDelete},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "get by customer id address",
			request: &sock.Request{URI: "/v1/agents/get_by_customer_id_address", Method: sock.RequestMethodPost,
				Data: []byte(`{"customer_id":"` + logTestCustomerID + `","address":{"type":"extension","target":"1000"}}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().GetByCustomerIDAndAddress(gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "login by username",
			request: &sock.Request{URI: "/v1/agents/user-a/login", Method: sock.RequestMethodPost,
				Data: []byte(`{"password":"secret-a"}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Login(gomock.Any(), "user-a", "secret-a").Return(bad, nil)
			},
		},
		{
			name: "login",
			request: &sock.Request{URI: "/v1/login", Method: sock.RequestMethodPost,
				Data: []byte(`{"username":"user-a","password":"secret-a"}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().Login(gomock.Any(), "user-a", "secret-a").Return(bad, nil)
			},
		},
		{
			name: "update addresses",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID + "/addresses", Method: sock.RequestMethodPut,
				Data: []byte(`{"addresses":[]}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateAddresses(gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "update basic info",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID, Method: sock.RequestMethodPut,
				Data: []byte(`{"name":"name-a"}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateBasicInfo(gomock.Any(), gomock.Any(), gomock.Any(), nil, nil).Return(bad, nil)
			},
		},
		{
			name: "update status",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID + "/status", Method: sock.RequestMethodPut,
				Data: []byte(`{"status":"available"}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), agent.StatusAvailable).Return(bad, nil)
			},
		},
		{
			name: "update password",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID + "/password", Method: sock.RequestMethodPut,
				Data: []byte(`{"password":"secret-a"}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdatePassword(gomock.Any(), gomock.Any(), "secret-a").Return(bad, nil)
			},
		},
		{
			name: "update tag ids",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID + "/tag_ids", Method: sock.RequestMethodPut,
				Data: []byte(`{"tag_ids":[]}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdateTagIDs(gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name: "update permission",
			request: &sock.Request{URI: "/v1/agents/" + logTestAgentID + "/permission", Method: sock.RequestMethodPut,
				Data: []byte(`{"permission":0}`)},
			expect: func(m *agenthandler.MockAgentHandler, bad *agent.Agent) {
				m.EXPECT().UpdatePermission(gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := newLogRedactionHook(t)
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockAgent := newLogRedactionHandler(mc)
			bad := &agent.Agent{
				Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil(logTestAgentID)},
				PasswordHash: logSentinelC,
				TMCreate:     &outOfRange,
			}
			tt.expect(mockAgent, bad)

			tt.request.DataType = "application/json"
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 500 {
				t.Fatalf("Wrong status. the marshal must fail. expect: 500, got: %d", res.StatusCode)
			}

			assertNoSecretInEntries(t, hook, "Could not marshal the response message.", logSentinelC)
		})
	}
}
