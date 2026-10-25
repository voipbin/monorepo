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
	"monorepo/bin-common-handler/pkg/requesthandler"
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
		KeyVersion:       1,
		CredentialSource: nmprovider.CredentialSourceDB,
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

	expect := `{"result":[{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":true,"key_version":1,"credential_source":"db","usable":true}],"next_page_token":""}`
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

	expect := `{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":true,"key_version":1,"credential_source":"db","usable":true}`
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

func Test_numberProvidersIDSecretPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newNumberProviderLogHook(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, accessLog, recoveryLog, ginErrors := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().NumberProviderUpdateSecret(gomock.Any(), a, "telnyx", testNumberProviderSecret).Return(testNumberProviderMessage(), nil)

	req, _ := http.NewRequest("PUT", "/number_providers/telnyx/secret", bytes.NewBufferString(`{"secret":"`+testNumberProviderSecret+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
	}

	// the response has no secret field at all, only has_secret
	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"secret", "secret_ciphertext", "ciphertext"} {
		if _, ok := res[key]; ok {
			t.Errorf("The response has the field %s", key)
		}
	}
	if res["has_secret"] != true {
		t.Errorf("Wrong match. expect: has_secret true, got: %v", res["has_secret"])
	}

	assertNoNumberProviderSecret(t, hook, w.Body.String(), accessLog.String(), recoveryLog.String())
	if *ginErrors != 0 {
		t.Errorf("The context holds %d errors, which the access log would print.", *ginErrors)
	}
	if accessLog.Len() == 0 {
		t.Error("The access log is empty, so the secret check on it would be vacuous.")
	}
}

// the body limit is 8 KiB: a body just below it reaches the service handler, a body above it does not
func Test_numberProvidersIDSecretPut_bodyLimit(t *testing.T) {
	if numberProviderMaxBodyBytes != 8192 {
		t.Fatalf("Wrong match. expect: 8192, got: %d", numberProviderMaxBodyBytes)
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	// number-manager decides the length of the secret, so it answers 400 for this one
	secret := strings.Repeat("a", 8000)
	mockSvc.EXPECT().NumberProviderUpdateSecret(gomock.Any(), a, "telnyx", secret).Return(nil, requesthandler.ErrBadRequest)

	req, _ := http.NewRequest("PUT", "/number_providers/telnyx/secret", bytes.NewBufferString(`{"secret":"`+secret+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}

func Test_numberProvidersIDSecretPut_failures(t *testing.T) {
	tests := []struct {
		name string
		body string

		serviceErr   error
		expectCall   bool
		expectStatus int
	}{
		{name: "malformed body holding the secret", body: `{"secret":"` + testNumberProviderSecret + `"`, expectStatus: http.StatusBadRequest},
		{name: "wrong type", body: `{"secret":["` + testNumberProviderSecret + `"]}`, expectStatus: http.StatusBadRequest},
		{name: "missing secret", body: `{}`, expectStatus: http.StatusBadRequest},
		{name: "null secret", body: `{"secret":null}`, expectStatus: http.StatusBadRequest},
		{name: "too large body", body: `{"secret":"` + testNumberProviderSecret + strings.Repeat("a", 9000) + `"}`, expectStatus: http.StatusBadRequest},
		{name: "number-manager rejects it", body: `{"secret":"` + testNumberProviderSecret + `"}`, serviceErr: requesthandler.ErrBadRequest, expectCall: true, expectStatus: http.StatusBadRequest},
		{name: "encryption key is not configured", body: `{"secret":"` + testNumberProviderSecret + `"}`, serviceErr: requesthandler.ErrServiceUnavailable, expectCall: true, expectStatus: http.StatusServiceUnavailable},
		{name: "provider is unknown", body: `{"secret":"` + testNumberProviderSecret + `"}`, serviceErr: requesthandler.ErrNotFound, expectCall: true, expectStatus: http.StatusNotFound},
		{name: "not allowed", body: `{"secret":"` + testNumberProviderSecret + `"}`, serviceErr: serviceerrors.ErrPermissionDenied, expectCall: true, expectStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newNumberProviderLogHook(t)
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, accessLog, recoveryLog, ginErrors := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			if tt.expectCall {
				mockSvc.EXPECT().NumberProviderUpdateSecret(gomock.Any(), a, "telnyx", testNumberProviderSecret).Return(nil, tt.serviceErr)
			}

			req, _ := http.NewRequest("PUT", "/number_providers/telnyx/secret", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, w.Code)
			}

			assertNoNumberProviderSecret(t, hook, w.Body.String(), accessLog.String(), recoveryLog.String())
			if *ginErrors != 0 {
				t.Errorf("The context holds %d errors, which the access log would print.", *ginErrors)
			}

			// a binding error text can quote the input, so a rejected body is logged with a fixed message only
			if !tt.expectCall {
				for _, entry := range hook.AllEntries() {
					if entry.Message != "Could not parse the request body." && entry.Message != "The secret is missing." {
						t.Errorf("A rejected body was logged with an unexpected message: %q", entry.Message)
					}
				}
			}
		})
	}
}

// a panic below the handler is caught by the recovery, whose dump must not hold the body
func Test_numberProvidersIDSecretPut_panic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newNumberProviderLogHook(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, accessLog, recoveryLog, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().NumberProviderUpdateSecret(gomock.Any(), a, "telnyx", testNumberProviderSecret).
		DoAndReturn(func(_ any, _ *auth.AuthIdentity, _ string, _ string) (*nmprovider.WebhookMessage, error) {
			panic("test panic")
		})

	req, _ := http.NewRequest("PUT", "/number_providers/telnyx/secret", bytes.NewBufferString(`{"secret":"`+testNumberProviderSecret+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusInternalServerError, w.Code)
	}
	if recoveryLog.Len() == 0 {
		t.Error("The recovery log is empty, so the secret check on it would be vacuous.")
	}

	assertNoNumberProviderSecret(t, hook, w.Body.String(), accessLog.String(), recoveryLog.String())
}

func Test_numberProvidersIDSecretDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	msg := testNumberProviderMessage()
	msg.HasSecret = false
	msg.KeyVersion = 0
	msg.CredentialSource = nmprovider.CredentialSourceEnv
	mockSvc.EXPECT().NumberProviderDeleteSecret(gomock.Any(), a, "telnyx").Return(msg, nil)

	req, _ := http.NewRequest("DELETE", "/number_providers/telnyx/secret", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
	}

	expect := `{"provider_name":"telnyx","enabled":true,"priority":10,"has_secret":false,"key_version":0,"credential_source":"env","usable":true}`
	if w.Body.String() != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
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
		{"PUT", "/number_providers/telnyx/secret", `{"secret":"` + testNumberProviderSecret + `"}`},
		{"DELETE", "/number_providers/telnyx/secret", ""},
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
