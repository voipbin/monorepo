package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/logmask"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	nmprovider "monorepo/bin-number-manager/models/provider"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
)

const testNumberProviderSecret = "test-secret-value-9f3a"

func testNumberProviderAgent() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})
}

func testNumberProviderMessage() *nmprovider.WebhookMessage {
	return &nmprovider.WebhookMessage{
		ProviderName:     "telnyx",
		Enabled:          true,
		Priority:         10,
		HasSecret:        true,
		CredentialSource: nmprovider.CredentialSourceEnv,
		Usable:           true,
	}
}

// newNumberProviderRouter builds a real gin router with the access logger and the recovery of
// api-manager, so the tests see what really reaches the log sinks.
func newNumberProviderRouter(h *server, a *auth.AuthIdentity) (*gin.Engine, *httptest.ResponseRecorder, *bytes.Buffer, *bytes.Buffer, *int) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	accessLog := &bytes.Buffer{}
	recoveryLog := &bytes.Buffer{}
	r.Use(logmask.Middlewares(nil, accessLog, recoveryLog)...)

	// the access log prints c.Errors, so an error attached to the context would reach the log
	ginErrors := new(int)
	r.Use(func(c *gin.Context) {
		c.Next()
		*ginErrors = len(c.Errors)
	})
	r.Use(func(c *gin.Context) {
		if a != nil {
			c.Set("auth_identity", a)
		}
	})
	openapi_server.RegisterHandlers(r, h)

	return r, w, accessLog, recoveryLog, ginErrors
}

// newNumberProviderLogHook captures the logs of the standard logger at the debug level.
func newNumberProviderLogHook(t *testing.T) *logrustest.Hook {
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

func assertNoNumberProviderSecret(t *testing.T, hook *logrustest.Hook, outputs ...string) {
	t.Helper()

	for _, entry := range hook.AllEntries() {
		b, _ := json.Marshal(entry.Data)
		if strings.Contains(entry.Message, testNumberProviderSecret) || strings.Contains(string(b), testNumberProviderSecret) {
			t.Errorf("The secret leaked into a log entry: %q", entry.Message)
		}
	}
	for _, out := range outputs {
		if strings.Contains(out, testNumberProviderSecret) {
			t.Errorf("The secret leaked into an output: %s", out)
		}
	}
}

func Test_numberProvidersGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().NumberProviderList(gomock.Any(), a).Return([]*nmprovider.WebhookMessage{testNumberProviderMessage()}, nil)

	req, _ := http.NewRequest("GET", "/number_providers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
	}

	expect := `{"result":[{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":true,"credential_source":"env","usable":true}],"next_page_token":""}`
	if w.Body.String() != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
	}
}

func Test_numberProvidersGet_empty(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().NumberProviderList(gomock.Any(), a).Return(nil, nil)

	req, _ := http.NewRequest("GET", "/number_providers", nil)
	r.ServeHTTP(w, req)

	if expect := `{"result":[],"next_page_token":""}`; w.Body.String() != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
	}
}

func Test_numberProvidersIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().NumberProviderGet(gomock.Any(), a, "telnyx").Return(testNumberProviderMessage(), nil)

	req, _ := http.NewRequest("GET", "/number_providers/telnyx", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
	}

	expect := `{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":true,"credential_source":"env","usable":true}`
	if w.Body.String() != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
	}
}

func Test_numberProvidersIDPut(t *testing.T) {
	falseValue := false
	zero := 0

	tests := []struct {
		name string
		body string

		expectEnabled  *bool
		expectPriority *int
		expectConfig   map[string]string
	}{
		{
			name:           "zero values reach the service handler",
			body:           `{"enabled":false,"priority":0}`,
			expectEnabled:  &falseValue,
			expectPriority: &zero,
		},
		{
			name:         "config",
			body:         `{"config":{"connection_id":"test-connection","profile_id":""}}`,
			expectConfig: map[string]string{"connection_id": "test-connection", "profile_id": ""},
		},
		{
			name: "empty body is passed on and number-manager answers 400",
			body: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			mockSvc.EXPECT().NumberProviderUpdate(gomock.Any(), a, "telnyx", tt.expectEnabled, tt.expectPriority, tt.expectConfig).Return(testNumberProviderMessage(), nil)

			req, _ := http.NewRequest("PUT", "/number_providers/telnyx", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_numberProvidersIDPut_badBody(t *testing.T) {
	for name, body := range map[string]string{
		"not json":   `{"enabled":`,
		"wrong type": `{"priority":"high"}`,
		"too large":  `{"config":{"connection_id":"` + strings.Repeat("a", 9000) + `"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no service handler call is expected
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: servicehandler.NewMockServiceHandler(mc)}, testNumberProviderAgent())

			req, _ := http.NewRequest("PUT", "/number_providers/telnyx", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
			if !strings.Contains(w.Body.String(), "INVALID_JSON_BODY") {
				t.Errorf("Wrong match. expect the INVALID_JSON_BODY reason, got: %s", w.Body.String())
			}
		})
	}
}

// every route answers 401 without an auth identity, before any service handler call
func Test_numberProviders_unauthenticated(t *testing.T) {
	routes := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/number_providers", ""},
		{"GET", "/number_providers/telnyx", ""},
		{"PUT", "/number_providers/telnyx", `{"enabled":true}`},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newNumberProviderLogHook(t)
			r, w, accessLog, recoveryLog, _ := newNumberProviderRouter(&server{serviceHandler: servicehandler.NewMockServiceHandler(mc)}, nil)

			req, _ := http.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
			}

			assertNoNumberProviderSecret(t, hook, w.Body.String(), accessLog.String(), recoveryLog.String())
		})
	}
}

// the permission and direct access errors of the service handler become 403
func Test_numberProviders_forbidden(t *testing.T) {
	for name, serviceErr := range map[string]error{
		"permission denied": serviceerrors.ErrPermissionDenied,
		"direct access":     serviceerrors.ErrDirectAccessNotSupported,
	} {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			mockSvc.EXPECT().NumberProviderGet(gomock.Any(), a, "telnyx").Return(nil, serviceErr)

			req, _ := http.NewRequest("GET", "/number_providers/telnyx", nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusForbidden, w.Code)
			}
		})
	}
}
