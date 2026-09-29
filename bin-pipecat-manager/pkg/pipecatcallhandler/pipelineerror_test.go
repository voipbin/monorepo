package pipecatcallhandler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"
)

// The literal strings below are what pipecat-ai 1.4.0 (and the pinned provider SDKs) actually push
// as RTVI error text. See VOIP-1542 design §1.3.
const (
	// The trigger incident, verbatim (Loki, 2026-09-29, pipecatcall 9c5c6e64).
	testGeminiInvalidKey = `Unknown error occurred: 400 Bad Request. {'message': '{\n  "error": {\n    "code": 400,\n    "message": "API key not valid. Please pass a valid API key.",\n    "status": "INVALID_ARGUMENT",\n    "details": [\n      {\n        "@type": "type.googleapis.com/google.rpc.ErrorInfo",\n        "reason": "API_KEY_INVALID",\n        "domain": "googleapis.com",\n        "metadata": {\n          "service": "generativelanguage.googleapis.com"\n        }\n      }\n    ]\n  }\n}\n', 'status': 'Bad Request'}`
)

func Test_classifyPipelineError(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		expect message.ErrorCategory
	}{
		// authentication
		{"gemini invalid key (incident)", testGeminiInvalidKey, message.ErrorCategoryAuthentication},
		{"openai incorrect key", "Error during completion: Error code: 401 - {'error': {'message': 'Incorrect API key provided: sk-xx***. You can find your API key at https://platform.openai.com/account/api-keys.', 'type': 'invalid_request_error', 'param': None, 'code': 'invalid_api_key'}}", message.ErrorCategoryAuthentication},
		{"gemini permission denied", "Unknown error occurred: 403 Forbidden. {'message': '{\"error\": {\"code\": 403, \"status\": \"PERMISSION_DENIED\"}}'}", message.ErrorCategoryAuthentication},
		{"elevenlabs websocket handshake 401", "Unknown error occurred: server rejected WebSocket connection: HTTP 401", message.ErrorCategoryAuthentication},
		{"status code field 401", `{"status": 401, "detail": "bad"}`, message.ErrorCategoryAuthentication},

		// rate limited
		{"gemini resource exhausted", "Unknown error occurred: 429 Too Many Requests. {'message': '{\"error\": {\"code\": 429, \"status\": \"RESOURCE_EXHAUSTED\"}}'}", message.ErrorCategoryRateLimited},
		{"openai quota", "Error during completion: Error code: 429 - {'error': {'message': 'You exceeded your current quota, please check your plan and billing details.', 'code': 'insufficient_quota'}}", message.ErrorCategoryRateLimited},
		{"bare RESOURCE_EXHAUSTED", "RESOURCE_EXHAUSTED", message.ErrorCategoryRateLimited},

		// timeout
		{"openai completion timeout", "LLM completion timeout", message.ErrorCategoryTimeout},
		{"openai request timed out", "Error during completion: Request timed out.", message.ErrorCategoryTimeout},
		{"gemini 504 reason phrase", "Unknown error occurred: 504 Gateway Timeout. {'message': 'DEADLINE_EXCEEDED', 'status': 'Gateway Timeout'}", message.ErrorCategoryTimeout},
		{"gemini 504 in-stream chunk", "Unknown error occurred: 504 DEADLINE_EXCEEDED. {'code': 504}", message.ErrorCategoryTimeout},

		// platform side
		{"function call failure", "Error executing function call [send_email]: boom", message.ErrorCategoryFunctionCall},
		{"malformed rtvi envelope", "Invalid RTVI transport message: 1 validation error", message.ErrorCategoryInternal},

		// unknown (including deliberate non-matches)
		{"google stt inactivity 409", "Unknown error occurred: 409 Stream timed out after receiving no more client requests.", message.ErrorCategoryUnknown},
		{"generic request timed out", "Request timed out", message.ErrorCategoryUnknown},
		{"deadline exceeded with space", "504 Deadline Exceeded", message.ErrorCategoryUnknown},
		{"gemini model not found", "Unknown error occurred: 404 Not Found. {'message': 'models/gemini-9 is not found', 'status': 'NOT_FOUND'}", message.ErrorCategoryUnknown},
		{"provider 500", "Unknown error occurred: 500 Internal Server Error.", message.ErrorCategoryUnknown},
		{"request id containing 401", "Unknown error occurred: upstream failure req-401-x", message.ErrorCategoryUnknown},
		{"url containing unauthorized in a timeout body", "Unknown error occurred: connect timeout to https://example.com/unauthorized/path", message.ErrorCategoryUnknown},
		{"id containing 14010", "Unknown error occurred: shard 14010 unavailable", message.ErrorCategoryUnknown},
		{"transport error", "Unknown error occurred: sent 1011 (internal error) keepalive ping timeout", message.ErrorCategoryUnknown},
		{"empty", "", message.ErrorCategoryUnknown},

		// isolating cases (VOIP-1543): each string matches ONLY the named matcher, so removing that
		// matcher flips it to unknown. Together with the realistic cases above (which already isolate
		// resource_exhausted, http 401, the auth status/code regex and the 4 timeout literals), every
		// matcher in pipelineerror.go has one. A new matcher needs a new row here.
		{"isolate tier1 api_key_invalid", "reason: API_KEY_INVALID", message.ErrorCategoryAuthentication},
		{"isolate tier1 invalid_api_key", "{'code': 'invalid_api_key'}", message.ErrorCategoryAuthentication},
		{"isolate tier1 permission_denied", "status PERMISSION_DENIED", message.ErrorCategoryAuthentication},
		{"isolate tier1 unauthenticated", "UNAUTHENTICATED: request had no credentials", message.ErrorCategoryAuthentication},
		{"isolate tier1 rate_limit_exceeded", "{'code': 'rate_limit_exceeded'}", message.ErrorCategoryRateLimited},
		{"isolate tier2 401 unauthorized", "server said 401 Unauthorized", message.ErrorCategoryAuthentication},
		{"isolate tier2 403 forbidden", "server said 403 Forbidden", message.ErrorCategoryAuthentication},
		{"isolate tier2 http 403", "server rejected WebSocket connection: HTTP 403", message.ErrorCategoryAuthentication},
		{"isolate tier2 429 reason phrase", "Unknown error occurred: 429 Too Many Requests.", message.ErrorCategoryRateLimited},
		{"isolate tier2 rate regex", "{\"status\": 429}", message.ErrorCategoryRateLimited},
		{"isolate tier3 api key not valid", "API key not valid. Please pass a valid API key.", message.ErrorCategoryAuthentication},
		{"isolate tier3 invalid api key", "Invalid API key supplied", message.ErrorCategoryAuthentication},
		{"isolate tier3 incorrect api key", "Incorrect API key provided: sk-xx", message.ErrorCategoryAuthentication},
		{"isolate tier3 invalid authentication", "Invalid authentication credentials", message.ErrorCategoryAuthentication},
		{"isolate tier3 rate limit", "slow down: rate limit reached", message.ErrorCategoryRateLimited},
		{"isolate tier3 exceeded your current quota", "You exceeded your current quota.", message.ErrorCategoryRateLimited},
		{"isolate tier3 quota exceeded", "Daily quota exceeded", message.ErrorCategoryRateLimited},

		// negatives: word boundary on the status/code regexes, prefix anchoring of tier 0 (VOIP-1543)
		{"status regex needs a word boundary after 401/403", "{\"status\": 4031}", message.ErrorCategoryUnknown},
		{"status regex needs a word boundary after 429", "{\"code\": 4290}", message.ErrorCategoryUnknown},
		{"function call phrase not at the start is not function_call", "LLM request failed while running error executing function call [x]", message.ErrorCategoryUnknown},
		{"rtvi envelope phrase not at the start is not internal", "Unknown error occurred: invalid rtvi transport message", message.ErrorCategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := classifyPipelineError(tt.raw); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func Test_shouldNotifyPipelineError(t *testing.T) {
	categories := []message.ErrorCategory{
		message.ErrorCategoryAuthentication,
		message.ErrorCategoryRateLimited,
		message.ErrorCategoryTimeout,
		message.ErrorCategoryUnknown,
		message.ErrorCategoryFunctionCall,
		message.ErrorCategoryInternal,
	}

	expect := func(c message.ErrorCategory, fatal bool, hasSTT bool) bool {
		switch c {
		case message.ErrorCategoryFunctionCall, message.ErrorCategoryInternal:
			return false
		case message.ErrorCategoryUnknown:
			return fatal || !hasSTT
		default:
			return true
		}
	}

	for _, c := range categories {
		for _, fatal := range []bool{false, true} {
			for _, hasSTT := range []bool{false, true} {
				want := expect(c, fatal, hasSTT)
				if res := shouldNotifyPipelineError(c, fatal, hasSTT); res != want {
					t.Errorf("Wrong match. category: %s, fatal: %v, hasSTT: %v, expect: %v, got: %v", c, fatal, hasSTT, want, res)
				}
			}
		}
	}
}

func Test_truncateForLog(t *testing.T) {
	if res := truncateForLog("short", 10); res != "short" {
		t.Errorf("Wrong match. got: %s", res)
	}

	res := truncateForLog(strings.Repeat("a", 20), 10)
	if res != strings.Repeat("a", 10)+"...(truncated)" {
		t.Errorf("Wrong match. got: %s", res)
	}

	// multi-byte: must not split a rune
	res = truncateForLog("가나다라", 4) // each rune is 3 bytes
	if res != "가...(truncated)" {
		t.Errorf("Wrong match. got: %q", res)
	}
}

func newPipelineErrorTestSession(hasSTT bool) *pipecatcall.Session {
	return &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("9c5c6e64-6289-4ac9-ad88-b20796a6bc96"),
			CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
		},
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.FromStringOrNil("443bfb46-fa3a-4dd5-912d-5482d31fed22"),
		ActiveflowID:             uuid.FromStringOrNil("8903319a-ddcd-4283-819e-bd7f6ce8c3f0"),
		Ctx:                      context.Background(),
		HasSTT:                   hasSTT,
	}
}

func errorFrame(text string, fatal bool) []byte {
	f := "false"
	if fatal {
		f = "true"
	}
	// text is embedded as a JSON string; the test strings below contain no quotes or backslashes.
	return []byte(`{"label":"rtvi-ai","type":"error","data":{"error":"` + text + `","fatal":` + f + `}}`)
}

func Test_receiveMessageFrameTypeMessage_error_publishesOnce(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := pipecatcallHandler{notifyHandler: mockNotify}
	se := newPipelineErrorTestSession(false)

	expectEvent := &message.PipelineErrorEvent{
		CustomerID:               se.CustomerID,
		PipecatcallID:            se.ID,
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   se.PipecatcallReferenceID,
		ActiveflowID:             se.ActiveflowID,
		Category:                 message.ErrorCategoryAuthentication,
		Fatal:                    false,
	}

	before := testutil.ToFloat64(metricsPipelineErrorTotal.WithLabelValues("authentication", "false"))

	var wg sync.WaitGroup
	wg.Add(1)
	mockNotify.EXPECT().PublishEvent(se.Ctx, message.EventTypePipelineError, expectEvent).Times(1).Do(
		func(any, any, any) { wg.Done() },
	)

	frame := errorFrame("Unknown error occurred: 400 Bad Request. API_KEY_INVALID", false)
	for i := 0; i < 3; i++ {
		if err := h.receiveMessageFrameTypeMessage(se, frame); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	waitWithTimeout(t, &wg)

	if diff := testutil.ToFloat64(metricsPipelineErrorTotal.WithLabelValues("authentication", "false")) - before; diff != 3 {
		t.Errorf("Wrong metric delta. expect: 3, got: %v", diff)
	}
}

func Test_receiveMessageFrameTypeMessage_error_differentCategoryPublishesAgain(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := pipecatcallHandler{notifyHandler: mockNotify}
	se := newPipelineErrorTestSession(false)

	var wg sync.WaitGroup
	wg.Add(2)
	mockNotify.EXPECT().PublishEvent(se.Ctx, message.EventTypePipelineError, gomock.Any()).Times(2).Do(
		func(any, any, any) { wg.Done() },
	)

	if err := h.receiveMessageFrameTypeMessage(se, errorFrame("API_KEY_INVALID", false)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.receiveMessageFrameTypeMessage(se, errorFrame("LLM completion timeout", false)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitWithTimeout(t, &wg)
}

func Test_receiveMessageFrameTypeMessage_error_notNotified(t *testing.T) {
	tests := []struct {
		name   string
		hasSTT bool
		text   string
	}{
		{"voice session unknown (stt reconnect)", true, "Unknown error occurred: 409 Stream timed out after receiving no more client requests."},
		{"function call failure", false, "Error executing function call [send_email]: boom"},
		{"malformed rtvi envelope", false, "Invalid RTVI transport message: bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := pipecatcallHandler{notifyHandler: mockNotify}
			se := newPipelineErrorTestSession(tt.hasSTT)

			mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			category := string(classifyPipelineError(tt.text))
			before := testutil.ToFloat64(metricsPipelineErrorTotal.WithLabelValues(category, "false"))

			if err := h.receiveMessageFrameTypeMessage(se, errorFrame(tt.text, false)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// give any (unexpected) publish goroutine a chance to run before the controller checks
			time.Sleep(20 * time.Millisecond)

			// a non-notified error is still counted
			if diff := testutil.ToFloat64(metricsPipelineErrorTotal.WithLabelValues(category, "false")) - before; diff != 1 {
				t.Errorf("Wrong metric delta. category: %s, expect: 1, got: %v", category, diff)
			}
		})
	}
}

func Test_receiveMessageFrameTypeMessage_error_textSessionUnknownNotified(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := pipecatcallHandler{notifyHandler: mockNotify}
	se := newPipelineErrorTestSession(false)

	var wg sync.WaitGroup
	wg.Add(1)
	mockNotify.EXPECT().PublishEvent(se.Ctx, message.EventTypePipelineError, gomock.Any()).Times(1).DoAndReturn(
		func(_ any, _ any, evt any) {
			defer wg.Done()
			e := evt.(*message.PipelineErrorEvent)
			if e.Category != message.ErrorCategoryUnknown {
				t.Errorf("Wrong match. expect: unknown, got: %s", e.Category)
			}
		},
	)

	if err := h.receiveMessageFrameTypeMessage(se, errorFrame("Unknown error occurred: 404 Not Found.", false)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitWithTimeout(t, &wg)
}

func Test_receiveMessageFrameTypeMessage_errorMalformed(t *testing.T) {
	h := pipecatcallHandler{}
	se := newPipelineErrorTestSession(false)

	if err := h.receiveMessageFrameTypeMessage(se, []byte(`{"label":"rtvi-ai","type":"error","data":"not-an-object"}`)); err == nil {
		t.Errorf("expected an error for a malformed error frame")
	}
}

func Test_receiveMessageFrameTypeMessage_errorResponse(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := pipecatcallHandler{notifyHandler: mockNotify}
	se := newPipelineErrorTestSession(false)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	hook := logrustest.NewGlobal()
	defer hook.Reset()

	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(origLevel)

	before := testutil.ToFloat64(metricsRTVIErrorResponseTotal)
	frame := []byte(`{"label":"rtvi-ai","type":"error-response","id":"abc","data":{"error":"Invalid message: bad"}}`)
	if err := h.receiveMessageFrameTypeMessage(se, frame); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if diff := testutil.ToFloat64(metricsRTVIErrorResponseTotal) - before; diff != 1 {
		t.Errorf("Wrong metric delta. expect: 1, got: %v", diff)
	}

	// the rejection is an operator signal: exactly one entry for this request, logged at WARN.
	var got []*logrus.Entry
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, "request_id: abc") {
			got = append(got, e)
		}
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly one log entry for the error-response frame, got %d", len(got))
	}
	if got[0].Level != logrus.WarnLevel {
		t.Errorf("Wrong log level. expect: %s, got: %s", logrus.WarnLevel, got[0].Level)
	}
}

// The first error frame of a category in a session is logged at WARN, repeats at DEBUG, and a new
// category is WARN again. This applies to non-notified categories too (VOIP-1542 design §3.1): a
// service that re-pushes the same error every second must not flood WARN.
func Test_runnerHandlePipelineError_warnOncePerCategory(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(origLevel)

	h := pipecatcallHandler{}
	se := newPipelineErrorTestSession(true) // voice session: unknown/function_call are not published

	frames := []struct {
		text      string
		expectLvl logrus.Level
	}{
		{"Unknown error occurred: 409 Stream timed out after receiving no more client requests.", logrus.WarnLevel},
		{"Unknown error occurred: 409 Stream timed out after receiving no more client requests.", logrus.DebugLevel},
		{"Unknown error occurred: 500 Internal Server Error.", logrus.DebugLevel}, // same category (unknown)
		{"Error executing function call [send_email]: boom", logrus.WarnLevel},
		{"Error executing function call [send_email]: boom", logrus.DebugLevel},
	}

	for i, f := range frames {
		hook.Reset()
		h.runnerHandlePipelineError(se, f.text, false)

		var got []*logrus.Entry
		for _, e := range hook.AllEntries() {
			if e.Data["func"] == "runnerHandlePipelineError" {
				got = append(got, e)
			}
		}
		if len(got) != 1 {
			t.Fatalf("frame %d: expected exactly one log entry, got %d", i, len(got))
		}
		if got[0].Level != f.expectLvl {
			t.Errorf("frame %d: wrong log level. expect: %s, got: %s", i, f.expectLvl, got[0].Level)
		}
	}
}

// Test_classifyPipelineError_checkOrder pins the documented first-hit order of the classifier
// checks (VOIP-1543 design §1.2b). Each input matches exactly two checks that yield different
// categories; the earlier check must win. Checks, in classifyPipelineError order:
//
//	C1 tier 0 prefix "error executing function call [" -> function_call
//	C2 tier 0 prefix "invalid rtvi transport message"  -> internal
//	C3 tier 1 auth tokens                              -> authentication
//	C4 tier 1 rate-limit tokens                        -> rate_limited
//	C5 tier 2 auth phrases or status/code 401|403      -> authentication
//	C6 tier 2 rate phrase or status/code 429           -> rate_limited
//	C7 tier 3 auth phrases                             -> authentication
//	C8 tier 3 rate-limit phrases                       -> rate_limited
//	C9 tier 3 timeout phrases                          -> timeout
//
// Same-category pairs are omitted (their order is unobservable), as is C1/C2 (two prefixes can
// never both match). C5/C6 are composites, so that pair is pinned in both phrase/regex
// combinations. Adding a check means adding a row for every differing-category pair it forms.
func Test_classifyPipelineError_checkOrder(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		expect message.ErrorCategory
	}{
		{"C1 before C3", "error executing function call [x] invalid_api_key ", message.ErrorCategoryFunctionCall},
		{"C1 before C4", "error executing function call [x] rate_limit_exceeded ", message.ErrorCategoryFunctionCall},
		{"C1 before C5", "error executing function call [x] status 401 ", message.ErrorCategoryFunctionCall},
		{"C1 before C6", "error executing function call [x] status 429 ", message.ErrorCategoryFunctionCall},
		{"C1 before C7", "error executing function call [x] invalid api key ", message.ErrorCategoryFunctionCall},
		{"C1 before C8", "error executing function call [x] quota exceeded ", message.ErrorCategoryFunctionCall},
		{"C1 before C9", "error executing function call [x] llm completion timeout ", message.ErrorCategoryFunctionCall},
		{"C2 before C3", "invalid rtvi transport message invalid_api_key ", message.ErrorCategoryInternal},
		{"C2 before C4", "invalid rtvi transport message rate_limit_exceeded ", message.ErrorCategoryInternal},
		{"C2 before C5", "invalid rtvi transport message status 401 ", message.ErrorCategoryInternal},
		{"C2 before C6", "invalid rtvi transport message status 429 ", message.ErrorCategoryInternal},
		{"C2 before C7", "invalid rtvi transport message invalid api key ", message.ErrorCategoryInternal},
		{"C2 before C8", "invalid rtvi transport message quota exceeded ", message.ErrorCategoryInternal},
		{"C2 before C9", "invalid rtvi transport message llm completion timeout ", message.ErrorCategoryInternal},
		{"C3 before C4", "invalid_api_key rate_limit_exceeded ", message.ErrorCategoryAuthentication},
		{"C3 before C6", "invalid_api_key status 429 ", message.ErrorCategoryAuthentication},
		{"C3 before C8", "invalid_api_key quota exceeded ", message.ErrorCategoryAuthentication},
		{"C3 before C9", "invalid_api_key llm completion timeout ", message.ErrorCategoryAuthentication},
		{"C4 before C5", "rate_limit_exceeded status 401 ", message.ErrorCategoryRateLimited},
		{"C4 before C7", "rate_limit_exceeded invalid api key ", message.ErrorCategoryRateLimited},
		{"C4 before C9", "rate_limit_exceeded llm completion timeout ", message.ErrorCategoryRateLimited},
		{"C5 before C6 (auth regex, rate phrase)", "status 401 429 too many requests ", message.ErrorCategoryAuthentication},
		{"C5 before C6 (auth phrase, rate regex)", "401 unauthorized status 429 ", message.ErrorCategoryAuthentication},
		{"C5 before C8", "status 401 quota exceeded ", message.ErrorCategoryAuthentication},
		{"C5 before C9", "status 401 llm completion timeout ", message.ErrorCategoryAuthentication},
		{"C6 before C7", "status 429 invalid api key ", message.ErrorCategoryRateLimited},
		{"C6 before C9", "status 429 llm completion timeout ", message.ErrorCategoryRateLimited},
		{"C7 before C8", "invalid api key quota exceeded ", message.ErrorCategoryAuthentication},
		{"C7 before C9", "invalid api key llm completion timeout ", message.ErrorCategoryAuthentication},
		{"C8 before C9", "quota exceeded llm completion timeout ", message.ErrorCategoryRateLimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := classifyPipelineError(tt.raw); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// A fatal unknown error on a voice session must be published with Fatal=true and counted under the
// fatal="true" label: the fatal flag is what lets a voice session surface an unknown error at all
// (VOIP-1543, F11/F12/F14).
//
// Diagnosis note: if the fatal flag is lost on the way to the policy or the payload, a voice
// unknown error is either not published at all or published with a mismatching event; in both
// cases this test fails with "timed out waiting for the expected PublishEvent call" after 5s
// (gomock logs the argument mismatch just above it), not with an assertion on Fatal itself.
func Test_receiveMessageFrameTypeMessage_error_fatalVoiceUnknownNotified(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := pipecatcallHandler{notifyHandler: mockNotify}
	se := newPipelineErrorTestSession(true)

	expectEvent := &message.PipelineErrorEvent{
		CustomerID:               se.CustomerID,
		PipecatcallID:            se.ID,
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   se.PipecatcallReferenceID,
		ActiveflowID:             se.ActiveflowID,
		Category:                 message.ErrorCategoryUnknown,
		Fatal:                    true,
	}

	before := testutil.ToFloat64(metricsPipelineErrorTotal.WithLabelValues("unknown", "true"))

	var wg sync.WaitGroup
	wg.Add(1)
	mockNotify.EXPECT().PublishEvent(se.Ctx, message.EventTypePipelineError, expectEvent).Times(1).Do(
		func(any, any, any) { wg.Done() },
	)

	if err := h.receiveMessageFrameTypeMessage(se, errorFrame("Unknown error occurred: 500 Internal Server Error.", true)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	waitWithTimeout(t, &wg)

	if diff := testutil.ToFloat64(metricsPipelineErrorTotal.WithLabelValues("unknown", "true")) - before; diff != 1 {
		t.Errorf("Wrong metric delta. expect: 1, got: %v", diff)
	}
}

// The WARN log of a pipeline error is truncated to pipelineErrorLogMaxLen bytes (VOIP-1543, F13).
// Voice session + non-fatal unknown error: nothing is published, so no notify handler is needed.
func Test_runnerHandlePipelineError_truncatesLog(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(origLevel)

	h := pipecatcallHandler{}
	se := newPipelineErrorTestSession(true)

	h.runnerHandlePipelineError(se, strings.Repeat("a", 2000)+"bbbb", false)

	var got []*logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Data["func"] == "runnerHandlePipelineError" {
			got = append(got, e)
		}
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly one log entry, got %d", len(got))
	}
	if got[0].Level != logrus.WarnLevel {
		t.Errorf("Wrong log level. expect: %s, got: %s", logrus.WarnLevel, got[0].Level)
	}
	if !strings.Contains(got[0].Message, strings.Repeat("a", 2000)+"...(truncated)") {
		t.Errorf("expected the first 2000 bytes followed by the truncation marker")
	}
	if strings.Contains(got[0].Message, "b") {
		t.Errorf("expected nothing past byte 2000 in the log message")
	}
}

// waitWithTimeout fails the test instead of hanging when an expected publish never happens
// (e.g. a classifier regression suppresses the event, so wg.Done is never called).
func waitWithTimeout(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for the expected PublishEvent call")
	}
}
