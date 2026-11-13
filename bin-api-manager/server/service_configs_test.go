package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/confighandler/configrpc"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

const testServiceConfigSecret = "test-secret-not-real"

func testServiceConfigKey() *configrpc.Key {
	return &configrpc.Key{Key: "mailgun.api_key", Kind: "secret", Type: "string", Description: "The key.", Source: "db", HasSecret: true, EnvName: "MAILGUN_API_KEY", KeyVersion: 1, Revision: 4}
}

const testServiceConfigKeyJSON = `{"key":"mailgun.api_key","kind":"secret","type":"string","description":"The key.","source":"db","has_secret":true,"env_name":"MAILGUN_API_KEY","env_shadowed":false,"key_version":1,"revision":4}`

// assertNoSecret checks that the secret is in no log entry and in none of the outputs.
func assertNoSecret(t *testing.T, hook *logrustest.Hook, secret string, outputs ...string) {
	t.Helper()

	for _, entry := range hook.AllEntries() {
		b, _ := json.Marshal(entry.Data)
		if strings.Contains(entry.Message, secret) || strings.Contains(string(b), secret) {
			t.Errorf("Wrong match. the secret is in a log entry: %q", entry.Message)
		}
	}
	for _, out := range outputs {
		if strings.Contains(out, secret) {
			t.Errorf("Wrong match. the secret is in an output: %s", out)
		}
	}
}

func do(r http.Handler, w *httptest.ResponseRecorder, method, uri, body string) {
	var req *http.Request
	if body == "" {
		req, _ = http.NewRequest(method, uri, http.NoBody)
	} else {
		req, _ = http.NewRequest(method, uri, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
}

func Test_serviceConfigsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigGet(gomock.Any(), a, "email").Return(&configrpc.Service{Service: "email", SecretWritable: true, Keys: []configrpc.Key{*testServiceConfigKey()}}, nil)

	do(r, w, "GET", "/service_configs/email", "")

	if w.Code != http.StatusOK {
		t.Fatalf("Wrong match. expect: %d, got: %d %s", http.StatusOK, w.Code, w.Body.String())
	}
	expect := `{"service":"email","secret_writable":true,"keys":[` + testServiceConfigKeyJSON + `]}`
	if w.Body.String() != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, w.Body.String())
	}
}

func Test_serviceConfigsKeySecretPut(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "email", "mailgun.api_key", testServiceConfigSecret, int64(4)).Return(testServiceConfigKey(), nil)

	do(r, w, "PUT", "/service_configs/email/keys/mailgun.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":4}`)

	if w.Code != http.StatusOK || w.Body.String() != testServiceConfigKeyJSON {
		t.Errorf("Wrong match.\nexpect: 200 %s\ngot: %d %s", testServiceConfigKeyJSON, w.Code, w.Body.String())
	}
}

// Revision 0 reaches the service handler as 0: it means "no stored value".
func Test_serviceConfigsKeySecretPut_revisionZero(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "email", "mailgun.api_key", testServiceConfigSecret, int64(0)).Return(testServiceConfigKey(), nil)

	do(r, w, "PUT", "/service_configs/email/keys/mailgun.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":0}`)

	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: 200, got: %d %s", w.Code, w.Body.String())
	}
}

// A body that is not what the contract says is a 400 INVALID_JSON_BODY and the service handler is
// not called. None of it is logged: the body is a secret.
func Test_serviceConfigsKeySecretPut_badBody(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"not json", `{"secret":"` + testServiceConfigSecret},
		{"missing secret", `{"revision":1}`},
		{"missing revision", `{"secret":"` + testServiceConfigSecret + `"}`},
		{"null revision", `{"secret":"` + testServiceConfigSecret + `","revision":null}`},
		{"revision is a string", `{"secret":"` + testServiceConfigSecret + `","revision":"1"}`},
		{"revision is a fraction", `{"secret":"` + testServiceConfigSecret + `","revision":1.5}`},
		{"secret is a number", `{"secret":123,"revision":1}`},
		{"empty body", ``},
		{"too large", `{"secret":"` + testServiceConfigSecret + strings.Repeat("a", 5000) + `","revision":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newNumberProviderLogHook(t)

			// no call is expected
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, accessLog, recoveryLog, ginErrors := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			do(r, w, "PUT", "/service_configs/email/keys/mailgun.api_key/secret", tt.body)

			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "INVALID_JSON_BODY") {
				t.Errorf("Wrong match. expect: 400 INVALID_JSON_BODY, got: %d %s", w.Code, w.Body.String())
			}
			if *ginErrors != 0 {
				t.Errorf("Wrong match. an error is attached to the context and would reach the access log")
			}
			assertNoSecret(t, hook, testServiceConfigSecret, w.Body.String(), accessLog.String(), recoveryLog.String())
		})
	}
}

func Test_serviceConfigsKeyDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigKeyClear(gomock.Any(), a, "email", "mailgun.api_key", int64(4)).Return(testServiceConfigKey(), nil)

	do(r, w, "DELETE", "/service_configs/email/keys/mailgun.api_key?revision=4", "")

	if w.Code != http.StatusOK || w.Body.String() != testServiceConfigKeyJSON {
		t.Errorf("Wrong match.\nexpect: 200 %s\ngot: %d %s", testServiceConfigKeyJSON, w.Code, w.Body.String())
	}
}

// The revision is required: without it, or with a value that is not a number, the generated
// binding answers 400 before the service handler is called.
func Test_serviceConfigsKeyDelete_badRevision(t *testing.T) {
	for name, uri := range map[string]string{
		"missing":      "/service_configs/email/keys/mailgun.api_key",
		"not a number": "/service_configs/email/keys/mailgun.api_key?revision=x",
		"empty":        "/service_configs/email/keys/mailgun.api_key?revision=",
	} {
		t.Run(name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, testNumberProviderAgent())

			do(r, w, "DELETE", uri, "")

			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func Test_serviceConfigsKeyHistories(t *testing.T) {
	tests := []struct {
		name string
		uri  string

		expectToken string
		expectSize  int
	}{
		{"no paging", "/service_configs/email/keys/mailgun.api_key/histories", "", 0},
		{"paging", "/service_configs/email/keys/mailgun.api_key/histories?page_size=5&page_token=abc", "abc", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			mockSvc.EXPECT().ServiceConfigKeyHistoryList(gomock.Any(), a, "email", "mailgun.api_key", tt.expectToken, tt.expectSize).Return(&configrpc.HistoryList{
				Result:        []configrpc.History{{Key: "mailgun.api_key", Action: "set", IsSecret: true, NewKeyVersion: 1, Source: "api"}},
				NextPageToken: "next",
			}, nil)

			do(r, w, "GET", tt.uri, "")

			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"next_page_token":"next"`) || !strings.Contains(w.Body.String(), `"result":[{`) {
				t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

// The errors of the service handler reach the client as the typed envelope with their reason.
func Test_serviceConfigs_errors(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expectStatus int
		expectReason string
	}{
		{"permission", serviceerrors.ErrPermissionDenied, http.StatusForbidden, "PERMISSION_DENIED"},
		{"direct access", serviceerrors.ErrDirectAccessNotSupported, http.StatusForbidden, "DIRECT_ACCESS_NOT_SUPPORTED"},
		{"unknown service", serviceerrors.ErrNotFound, http.StatusNotFound, "RESOURCE_NOT_FOUND"},
		{"bad key", serviceerrors.ErrInvalidArgument, http.StatusBadRequest, "INVALID_ARGUMENT"},
		{"manager without the route", serviceerrors.ErrServiceUnavailable, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE"},
		{"stale", cerrors.FailedPrecondition("email-manager", "REVISION_MISMATCH", "stale"), http.StatusConflict, "REVISION_MISMATCH"},
		{"no master key", cerrors.Unavailable("email-manager", "ENCRYPTION_KEY_NOT_CONFIGURED", "no key"), http.StatusServiceUnavailable, "ENCRYPTION_KEY_NOT_CONFIGURED"},
		{"unknown key", cerrors.NotFound("email-manager", "CONFIG_KEY_NOT_FOUND", "no key"), http.StatusNotFound, "CONFIG_KEY_NOT_FOUND"},
		{"invalid value", cerrors.InvalidArgument("email-manager", "CONFIG_INVALID", "bad"), http.StatusBadRequest, "CONFIG_INVALID"},
		{"write verify", cerrors.Internal("email-manager", "CONFIG_WRITE_VERIFY_FAILED", "verify"), http.StatusInternalServerError, "CONFIG_WRITE_VERIFY_FAILED"},
		{"timeout", requesthandler.ErrInternal, http.StatusInternalServerError, "INTERNAL"},
		// the allow list of the service handler has a service that the request handler does not know
		{"service unknown to the request handler", requesthandler.ErrConfigServiceUnknown, http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newNumberProviderLogHook(t)

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, accessLog, recoveryLog, ginErrors := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "email", "mailgun.api_key", testServiceConfigSecret, int64(1)).Return(nil, tt.err)

			do(r, w, "PUT", "/service_configs/email/keys/mailgun.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":1}`)

			if w.Code != tt.expectStatus || !strings.Contains(w.Body.String(), tt.expectReason) {
				t.Errorf("Wrong match. expect: %d %s, got: %d %s", tt.expectStatus, tt.expectReason, w.Code, w.Body.String())
			}
			_ = ginErrors
			assertNoSecret(t, hook, testServiceConfigSecret, w.Body.String(), accessLog.String(), recoveryLog.String())
		})
	}
}

// The service ai is routed like email: the generated binding passes the service name on as it is.
// The typed errors of the ai manager have the same status and reason as the ones of the email
// manager, because the mapping uses the reason and the status only.
func Test_serviceConfigs_aiService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigGet(gomock.Any(), a, "ai").Return(&configrpc.Service{Service: "ai", SecretWritable: true, Keys: []configrpc.Key{*testServiceConfigKey()}}, nil)
	do(r, w, "GET", "/service_configs/ai", "")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), `{"service":"ai",`) {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "ai", "llm.gemini.api_key", testServiceConfigSecret, int64(2)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "PUT", "/service_configs/ai/keys/llm.gemini.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":2}`)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyClear(gomock.Any(), a, "ai", "llm.openai_compat.api_key", int64(3)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "DELETE", "/service_configs/ai/keys/llm.openai_compat.api_key?revision=3", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyHistoryList(gomock.Any(), a, "ai", "llm.gemini.api_key", "", 0).Return(&configrpc.HistoryList{Result: []configrpc.History{}}, nil)
	w = httptest.NewRecorder()
	do(r, w, "GET", "/service_configs/ai/keys/llm.gemini.api_key/histories", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}
}

// The service message is routed like email and ai: the service name reaches the service handler as
// it is, for the keys of the two providers.
func Test_serviceConfigs_messageService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigGet(gomock.Any(), a, "message").Return(&configrpc.Service{Service: "message", SecretWritable: true, Keys: []configrpc.Key{*testServiceConfigKey()}}, nil)
	do(r, w, "GET", "/service_configs/message", "")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), `{"service":"message",`) {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "message", "provider.telnyx.auth_token", testServiceConfigSecret, int64(2)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "PUT", "/service_configs/message/keys/provider.telnyx.auth_token/secret", `{"secret":"`+testServiceConfigSecret+`","revision":2}`)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyClear(gomock.Any(), a, "message", "provider.messagebird.auth_token", int64(3)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "DELETE", "/service_configs/message/keys/provider.messagebird.auth_token?revision=3", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyHistoryList(gomock.Any(), a, "message", "provider.telnyx.auth_token", "", 0).Return(&configrpc.HistoryList{Result: []configrpc.History{}}, nil)
	w = httptest.NewRecorder()
	do(r, w, "GET", "/service_configs/message/keys/provider.telnyx.auth_token/histories", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}
}

// The service billing is routed like email: the service name reaches the service handler as it is.
func Test_serviceConfigs_billingService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigGet(gomock.Any(), a, "billing").Return(&configrpc.Service{Service: "billing", SecretWritable: true, Keys: []configrpc.Key{*testServiceConfigKey()}}, nil)
	do(r, w, "GET", "/service_configs/billing", "")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), `{"service":"billing",`) {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "billing", "paddle.api_key", testServiceConfigSecret, int64(2)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "PUT", "/service_configs/billing/keys/paddle.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":2}`)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}
}

// The service pipecat is routed like email, for the key with an underscore in the name.
func Test_serviceConfigs_pipecatService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "pipecat", "local_llm.endpoints", testServiceConfigSecret, int64(1)).Return(testServiceConfigKey(), nil)
	do(r, w, "PUT", "/service_configs/pipecat/keys/local_llm.endpoints/secret", `{"secret":"`+testServiceConfigSecret+`","revision":1}`)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyClear(gomock.Any(), a, "pipecat", "local_llm.endpoints", int64(3)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "DELETE", "/service_configs/pipecat/keys/local_llm.endpoints?revision=3", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}
}

// The longest secret of a key (2020 bytes, the local_llm.endpoints key of pipecat) in the worst
// case of JSON escaping (every byte a quote or a backslash, 4040 bytes) with the longest
// revision fits in the body limit, and reaches the service handler unchanged. This holds the
// limit: lowering it or the secret limit going up breaks this test.
func Test_serviceConfigsKeySecretPut_worstCaseBody(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{"quotes", strings.Repeat(`"`, 2020)},
		{"backslashes", strings.Repeat(`\`, 2020)},
		{"mixed", strings.Repeat(`"\`, 1010)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			if len(tt.secret) != 2020 {
				t.Fatalf("Wrong test data. expect: 2020 bytes, got: %d", len(tt.secret))
			}
			encoded, err := json.Marshal(tt.secret)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			// the value of the encoder is the worst case itself: 2 bytes per byte, in the quotes of the string.
			body := `{"secret":` + string(encoded) + `,"revision":4294967295}`
			if len(body) > serviceConfigMaxBodyBytes {
				t.Fatalf("Wrong match. the body is over the limit. limit: %d, got: %d", serviceConfigMaxBodyBytes, len(body))
			}

			mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "pipecat", "local_llm.endpoints", tt.secret, int64(4294967295)).Return(testServiceConfigKey(), nil)
			do(r, w, "PUT", "/service_configs/pipecat/keys/local_llm.endpoints/secret", body)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: 200, got: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

// The service tts is routed like email: the generated binding passes the service name on as it is.
func Test_serviceConfigs_ttsService(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigGet(gomock.Any(), a, "tts").Return(&configrpc.Service{Service: "tts", SecretWritable: true, Keys: []configrpc.Key{*testServiceConfigKey()}}, nil)
	do(r, w, "GET", "/service_configs/tts", "")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Body.String(), `{"service":"tts",`) {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "tts", "aws.access_key", testServiceConfigSecret, int64(2)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "PUT", "/service_configs/tts/keys/aws.access_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":2}`)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyClear(gomock.Any(), a, "tts", "elevenlabs.api_key", int64(3)).Return(testServiceConfigKey(), nil)
	w = httptest.NewRecorder()
	do(r, w, "DELETE", "/service_configs/tts/keys/elevenlabs.api_key?revision=3", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}

	mockSvc.EXPECT().ServiceConfigKeyHistoryList(gomock.Any(), a, "tts", "aws.access_key", "", 0).Return(&configrpc.HistoryList{Result: []configrpc.History{}}, nil)
	w = httptest.NewRecorder()
	do(r, w, "GET", "/service_configs/tts/keys/aws.access_key/histories", "")
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. got: %d %s", w.Code, w.Body.String())
	}
}

// The generated binding does not check the service name against the enum: a value that is not
// allowed reaches the service handler, which answers 404. The allow list of the service handler
// is the only place that decides.
func Test_serviceConfigs_serviceNameIsNotCheckedByTheBinding(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigGet(gomock.Any(), a, "voice").Return(nil, serviceerrors.ErrNotFound)
	do(r, w, "GET", "/service_configs/voice", "")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "RESOURCE_NOT_FOUND") {
		t.Errorf("Wrong match. expect: 404 RESOURCE_NOT_FOUND, got: %d %s", w.Code, w.Body.String())
	}
}

// The typed errors of the ai manager have the same status and reason as the ones of the email manager.
func Test_serviceConfigs_aiErrors(t *testing.T) {
	tests := []struct {
		err          error
		expectStatus int
		expectReason string
	}{
		{cerrors.FailedPrecondition("bin-ai-manager", "REVISION_MISMATCH", "stale"), http.StatusConflict, "REVISION_MISMATCH"},
		{cerrors.Unavailable("bin-ai-manager", "ENCRYPTION_KEY_NOT_CONFIGURED", "no key"), http.StatusServiceUnavailable, "ENCRYPTION_KEY_NOT_CONFIGURED"},
		{cerrors.NotFound("bin-ai-manager", "CONFIG_KEY_NOT_FOUND", "no key"), http.StatusNotFound, "CONFIG_KEY_NOT_FOUND"},
		{cerrors.InvalidArgument("bin-ai-manager", "CONFIG_INVALID", "bad"), http.StatusBadRequest, "CONFIG_INVALID"},
		{cerrors.Internal("bin-ai-manager", "CONFIG_WRITE_VERIFY_FAILED", "verify"), http.StatusInternalServerError, "CONFIG_WRITE_VERIFY_FAILED"},
		{cerrors.Internal("bin-ai-manager", "INTERNAL", "internal"), http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.expectReason, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := testNumberProviderAgent()
			r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

			mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "ai", "llm.gemini.api_key", testServiceConfigSecret, int64(1)).Return(nil, tt.err)
			do(r, w, "PUT", "/service_configs/ai/keys/llm.gemini.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":1}`)

			if w.Code != tt.expectStatus || !strings.Contains(w.Body.String(), tt.expectReason) || strings.Contains(w.Body.String(), "bin-ai-manager") {
				t.Errorf("Wrong match. expect: %d %s, got: %d %s", tt.expectStatus, tt.expectReason, w.Code, w.Body.String())
			}
		})
	}
}

// No identity is a 401 and the service handler is not called.
func Test_serviceConfigs_noIdentity(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	r, w, _, _, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, nil)

	do(r, w, "PUT", "/service_configs/email/keys/mailgun.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":1}`)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong match. expect: 401, got: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	do(r, w, "GET", "/service_configs/email", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong match. expect: 401, got: %d %s", w.Code, w.Body.String())
	}
}

// A request that is not routed by the generated binding (a mistyped path) never reaches the
// handlers; the secret in its body is still not logged by the access logger.
func Test_serviceConfigs_secretNotInTheAccessLog(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newNumberProviderLogHook(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := testNumberProviderAgent()
	r, w, accessLog, recoveryLog, _ := newNumberProviderRouter(&server{serviceHandler: mockSvc}, a)

	mockSvc.EXPECT().ServiceConfigKeySecretSet(gomock.Any(), a, "email", "mailgun.api_key", testServiceConfigSecret, int64(1)).Return(testServiceConfigKey(), nil)
	do(r, w, "PUT", "/service_configs/email/keys/mailgun.api_key/secret", `{"secret":"`+testServiceConfigSecret+`","revision":1}`)

	w2 := httptest.NewRecorder()
	do(r, w2, "PUT", "/service_configs/email/keys/mailgun.api_key/secrets", `{"secret":"`+testServiceConfigSecret+`","revision":1}`)

	assertNoSecret(t, hook, testServiceConfigSecret, w.Body.String(), w2.Body.String(), accessLog.String(), recoveryLog.String())
	if accessLog.Len() == 0 {
		t.Errorf("Wrong match. the access log is empty, so the check above proves nothing")
	}
}

// The responses are the RPC models of configrpc, and the public contract is the generated model.
// A drift of a field name or a type in either one is caught here: a response must decode into the
// generated model without an unknown field, and both must have the same set of fields.
func Test_serviceConfigs_responsesMatchTheContract(t *testing.T) {
	jsonNames := func(v any) []string {
		var res []string
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			if name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]; name != "" && name != "-" {
				res = append(res, name)
			}
		}
		sort.Strings(res)

		return res
	}

	actor := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	key := testServiceConfigKey()
	key.UpdatedBy = &actor
	key.TMUpdate = &now
	key.EnvShadowed = true

	tests := []struct {
		name   string
		rpc    any
		model  any
		fields [2]any
	}{
		{"service", &configrpc.Service{Service: "email", SecretWritable: true, Keys: []configrpc.Key{*key}}, &openapi_server.ServiceConfig{}, [2]any{configrpc.Service{}, openapi_server.ServiceConfig{}}},
		{"key", key, &openapi_server.ServiceConfigKey{}, [2]any{configrpc.Key{}, openapi_server.ServiceConfigKey{}}},
		{"history", &configrpc.HistoryList{Result: []configrpc.History{{ID: actor, Key: "k", Action: "set", IsSecret: true, ActorID: actor, Source: "api", TMCreate: now}}, NextPageToken: "n"}, &openapi_server.ServiceConfigHistoryList{}, [2]any{configrpc.History{}, openapi_server.ServiceConfigHistory{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.rpc)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			dec := json.NewDecoder(bytes.NewReader(b))
			dec.DisallowUnknownFields()
			if err := dec.Decode(tt.model); err != nil {
				t.Errorf("Wrong match. the response does not fit the generated model: %v", err)
			}

			if got, expect := jsonNames(tt.fields[0]), jsonNames(tt.fields[1]); !reflect.DeepEqual(got, expect) {
				t.Errorf("Wrong match. the rpc and the generated model differ.\nrpc: %v\nmodel: %v", got, expect)
			}
		})
	}
}
