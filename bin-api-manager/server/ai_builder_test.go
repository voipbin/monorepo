package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
)

const builderServerSecret = "SECRET-INPUT-STRING-do-not-log"

func builderServerAgent() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
}

func builderServerAccesskey() *auth.AuthIdentity {
	return auth.NewAccesskeyIdentity(&csaccesskey.Accesskey{
		ID:         uuid.FromStringOrNil("11111111-0000-0000-0000-0000000000aa"),
		CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
	})
}

// serveBuilder runs one request through the generated router with the given
// identity (nil means none) and returns the recorder.
func serveBuilder(t *testing.T, svc servicehandler.ServiceHandler, identity *auth.AuthIdentity, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		if identity != nil {
			c.Set("auth_identity", identity)
		}
	})
	openapi_server.RegisterHandlers(r, &server{serviceHandler: svc})

	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func builderBody(content string) []byte {
	b, _ := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "user", "content": content}}})
	return b
}

func reasonFromBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		Error struct {
			Reason string `json:"reason"`
		} `json:"error"`
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Error.Reason != "" {
		return out.Error.Reason
	}
	return out.Reason
}

func Test_PostAiBuilderChat_success(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()

	mockSvc.EXPECT().AIBuilderChat(gomock.Any(), a, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *auth.AuthIdentity, req *builder.ChatRequest) (*builder.ChatResponse, error) {
			if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
				t.Errorf("the body did not reach the service intact: %+v", req)
			}
			return &builder.ChatResponse{Message: "hi"}, nil
		})

	w := serveBuilder(t, mockSvc, a, "POST", "/ai_builder/chat", builderBody("hello"))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"message":"hi"`) {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}

// Who may call is decided in the service handler (the one place both server
// and tests exercise); the server only refuses a caller that is not an Agent
// with 403 and never reaches the service. (That the refusal comes before the
// body is read is by the order of the code; this test does not prove it.)
func Test_PostAiBuilderChat_nonAgentIsRefusedWith403AndNeverReachesTheService(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc) // strict: no service call
	for name, id := range map[string]*auth.AuthIdentity{
		"accesskey": builderServerAccesskey(),
		"direct":    auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")}),
		"delegate":  auth.NewDelegateIdentity(&auth.DelegateScope{CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")}),
	} {
		t.Run(name, func(t *testing.T) {
			w := serveBuilder(t, mockSvc, id, "POST", "/ai_builder/chat", builderBody("hello"))
			if w.Code != http.StatusForbidden {
				t.Errorf("got %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func Test_PostAiBuilderChat_noIdentityIs401(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	w := serveBuilder(t, mockSvc, nil, "POST", "/ai_builder/chat", builderBody("hello"))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d", w.Code)
	}
}

func Test_PostAiBuilderChat_badJSONIs400AndEchoesNothing(t *testing.T) {
	var buf strings.Builder
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	w := serveBuilder(t, mockSvc, builderServerAgent(), "POST", "/ai_builder/chat", []byte("{"+builderServerSecret))
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), builderServerSecret) || strings.Contains(buf.String(), builderServerSecret) {
		t.Errorf("the body was echoed:\nresponse: %s\nlog: %s", w.Body.String(), buf.String())
	}
}

// A body larger than the cap is refused with its own reason, before it is
// parsed, and the cap is 160 KB: well above the 40000-rune conversation limit
// even in Korean (3 bytes a rune) plus JSON escaping.
//
// The size error surfaces while the JSON is being read, so it takes a body that
// is JSON up to the cap. A body that is not JSON at all fails on its first byte
// with a syntax error and is reported as INVALID_JSON_BODY; the read stops
// there, so nothing past the first bytes is consumed either way. Both are
// pinned below.
func Test_PostAiBuilderChat_oversizedBodyIsRefused(t *testing.T) {
	if builderMaxBodyBytes != 160<<10 {
		t.Errorf("cap: got %d", builderMaxBodyBytes)
	}

	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc) // strict: never reaches the service

	t.Run("json over the cap", func(t *testing.T) {
		big := []byte(`{"messages":[{"role":"user","content":"` + strings.Repeat("a", builderMaxBodyBytes+10) + `"}]}`)
		w := serveBuilder(t, mockSvc, builderServerAgent(), "POST", "/ai_builder/chat", big)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("got %d", w.Code)
		}
		if got := reasonFromBody(t, w); got != builder.ReasonInputTooLarge {
			t.Errorf("reason: got %q, want %s (%s)", got, builder.ReasonInputTooLarge, w.Body.String())
		}
	})

	t.Run("not json over the cap", func(t *testing.T) {
		big := bytes.Repeat([]byte("a"), builderMaxBodyBytes+10)
		w := serveBuilder(t, mockSvc, builderServerAgent(), "POST", "/ai_builder/chat", big)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("got %d", w.Code)
		}
		if got := reasonFromBody(t, w); got != "INVALID_JSON_BODY" {
			t.Errorf("reason: got %q (%s)", got, w.Body.String())
		}
	})
}

// Just under the cap is read normally.
func Test_PostAiBuilderChat_bodyJustUnderTheCapIsRead(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()
	mockSvc.EXPECT().AIBuilderChat(gomock.Any(), a, gomock.Any()).Return(&builder.ChatResponse{Message: "ok"}, nil)

	pad := strings.Repeat("a", builderMaxBodyBytes-200)
	body := []byte(`{"messages":[{"role":"user","content":"hi"}],"pad":"` + pad + `"}`)
	if len(body) >= builderMaxBodyBytes {
		t.Fatal("the test body is not under the cap")
	}
	w := serveBuilder(t, mockSvc, a, "POST", "/ai_builder/chat", body)
	if w.Code != http.StatusOK {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}

// Every error of the service handler becomes the status and reason of design
// 4.7, through the real translator.
func Test_PostAiBuilderChat_errorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantReason string
	}{
		{"daily limit", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonDailyLimit, "x"), http.StatusTooManyRequests, builder.ReasonDailyLimit},
		{"busy", cerrors.ResourceExhausted(commonoutline.ServiceNameAIManager, builder.ReasonBusy, "x"), http.StatusTooManyRequests, builder.ReasonBusy},
		{"timeout", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x"), http.StatusServiceUnavailable, builder.ReasonTimeout},
		{"response invalid", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonResponseInvalid, "x"), http.StatusServiceUnavailable, builder.ReasonResponseInvalid},
		{"unavailable", cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonUnavailable, "x"), http.StatusServiceUnavailable, builder.ReasonUnavailable},
		{"invalid argument", cerrors.InvalidArgument(commonoutline.ServiceNameAIManager, builder.ReasonInvalidArgument, "x"), http.StatusBadRequest, builder.ReasonInvalidArgument},
		{"untyped", errors.New("boom"), http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := builderServerAgent()
			mockSvc.EXPECT().AIBuilderChat(gomock.Any(), a, gomock.Any()).Return(nil, tt.err)

			w := serveBuilder(t, mockSvc, a, "POST", "/ai_builder/chat", builderBody("hello"))
			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}
			if got := reasonFromBody(t, w); got != tt.wantReason {
				t.Errorf("reason: got %q, want %q (%s)", got, tt.wantReason, w.Body.String())
			}
		})
	}
}

// Nothing the customer wrote reaches a log line, whatever fails below.
func Test_PostAiBuilderChat_neverLogsTheInput(t *testing.T) {
	var buf strings.Builder
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	for name, svcErr := range map[string]error{
		"success":          nil,
		"echoing provider": errors.New("400: " + builderServerSecret),
		"typed with cause": cerrors.Unavailable(commonoutline.ServiceNameAIManager, builder.ReasonTimeout, "x").Wrap(errors.New(builderServerSecret)),
	} {
		t.Run(name, func(t *testing.T) {
			buf.Reset()
			mc := gomock.NewController(t)
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			a := builderServerAgent()
			if svcErr == nil {
				mockSvc.EXPECT().AIBuilderChat(gomock.Any(), a, gomock.Any()).Return(&builder.ChatResponse{Message: "ok"}, nil)
			} else {
				mockSvc.EXPECT().AIBuilderChat(gomock.Any(), a, gomock.Any()).Return(nil, svcErr)
			}
			w := serveBuilder(t, mockSvc, a, "POST", "/ai_builder/chat", builderBody(builderServerSecret))
			if strings.Contains(buf.String(), builderServerSecret) {
				t.Errorf("a log line carries the input:\n%s", buf.String())
			}
			if strings.Contains(w.Body.String(), builderServerSecret) {
				t.Errorf("the response carries the input: %s", w.Body.String())
			}
		})
	}
}

func Test_GetAiBuilderStatus(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()
	mockSvc.EXPECT().AIBuilderStatus(gomock.Any(), a).Return(&builder.StatusResponse{Available: true, MaxMessages: 40, MaxMessageChars: 2000}, nil)

	w := serveBuilder(t, mockSvc, a, "GET", "/ai_builder/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
	var out builder.StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || !out.Available || out.MaxMessages != 40 || out.MaxMessageChars != 2000 {
		t.Errorf("got %s (%v)", w.Body.String(), err)
	}
}

// A caller who is not an Agent gets 200 with available=false, not an error:
// the client uses this to decide whether to show the entry point.
func Test_GetAiBuilderStatus_nonAgentIs200Unavailable(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc) // strict: the service is not asked
	for name, id := range map[string]*auth.AuthIdentity{
		"accesskey": builderServerAccesskey(),
		"direct":    auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")}),
	} {
		t.Run(name, func(t *testing.T) {
			w := serveBuilder(t, mockSvc, id, "GET", "/ai_builder/status", nil)
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"available":false`) {
				t.Errorf("got %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetAiBuilderStatus_noIdentityIs401(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	if w := serveBuilder(t, mockSvc, nil, "GET", "/ai_builder/status", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("got %d", w.Code)
	}
}

// A service error (not a permission refusal) also becomes available=false:
// the status route must not break the screen that calls it.
func Test_GetAiBuilderStatus_serviceErrorIsStillUnavailable200(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()
	mockSvc.EXPECT().AIBuilderStatus(gomock.Any(), a).Return(nil, errors.New("down"))

	w := serveBuilder(t, mockSvc, a, "GET", "/ai_builder/status", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"available":false`) {
		t.Errorf("got %d %s", w.Code, w.Body.String())
	}
}

// Everything the client sends reaches the service: both roles in order, and the
// draft with every field. A copy that drops the draft or flattens the roles
// would make the "refine the form" turn start from nothing.
func Test_PostAiBuilderChat_passesTheWholeRequestOn(t *testing.T) {
	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	a := builderServerAgent()

	want := &builder.ChatRequest{
		Messages: []builder.Message{
			{Role: builder.RoleUser, Content: "first"},
			{Role: builder.RoleAssistant, Content: "second"},
			{Role: builder.RoleUser, Content: "third"},
		},
		CurrentDraft: &builder.Draft{Name: "n", Detail: "d", InitPrompt: "p", ToolNames: []string{"connect_call", "stop_service"}},
	}
	mockSvc.EXPECT().AIBuilderChat(gomock.Any(), a, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *auth.AuthIdentity, got *builder.ChatRequest) (*builder.ChatResponse, error) {
			if !reflect.DeepEqual(got, want) {
				t.Errorf("the request changed on the way.\nwant: %+v\ngot:  %+v", want, got)
			}
			return &builder.ChatResponse{Message: "ok"}, nil
		})

	body, _ := json.Marshal(want)
	w := serveBuilder(t, mockSvc, a, "POST", "/ai_builder/chat", body)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d %s", w.Code, w.Body.String())
	}
}

// A parse failure is logged without the underlying error: encoding/json can
// quote the bytes it stopped at, and they are the customer's text. A substring
// check cannot prove that (the error may quote only one byte), so the shape is
// pinned instead.
func Test_PostAiBuilderChat_parseFailureLogCarriesNoUnderlyingError(t *testing.T) {
	var buf strings.Builder
	origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer func() { logrus.SetOutput(origOut); logrus.SetLevel(origLevel) }()

	mc := gomock.NewController(t)
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	w := serveBuilder(t, mockSvc, builderServerAgent(), "POST", "/ai_builder/chat", []byte("{"+builderServerSecret))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d", w.Code)
	}
	if strings.Contains(buf.String(), "err:") || strings.Contains(buf.String(), "invalid character") {
		t.Errorf("the log carries the underlying json error:\n%s", buf.String())
	}
}
