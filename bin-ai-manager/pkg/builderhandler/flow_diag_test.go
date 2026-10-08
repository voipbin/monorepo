package builderhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"strings"
	"testing"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/flowbuilder"
)

const diagMessage = "The flow builder model call finished."

// diagNumericFields are present on every diagnostic line.
var diagNumericFields = []string{
	"elapsed_ms", "build_ms", "pre_call_ms", "chat_ms", "prompt_tokens", "completion_tokens",
	"response_chars", "system_chars", "request_chars", "schema_bytes", "allowed_types",
	"user_turns", "history_messages", "max_tokens", "llm_timeout_ms",
}

// diagAllowedFields is the closed set of field names of the diagnostic line:
// the new fields plus the ones Chat already puts on its logger. A string field
// added later has to be added here on purpose.
var diagAllowedFields = map[string]bool{
	"func": true, "customer_id": true, "message_count": true,
	"outcome": true, "invalid_kind": true, "finish_reason": true, "model": true,
	"reasoning_effort": true, "json_mode": true, "current_draft_present": true,
	"has_draft": true, "draft_discarded": true, "empty_draft": true,
	"elapsed_ms": true, "build_ms": true, "pre_call_ms": true, "chat_ms": true,
	"prompt_tokens": true, "completion_tokens": true, "response_chars": true,
	"system_chars": true, "request_chars": true, "schema_bytes": true,
	"allowed_types": true, "user_turns": true, "history_messages": true,
	"max_tokens": true, "llm_timeout_ms": true,
}

// nilRespSender answers with no response and no error.
type nilRespSender struct{}

func (nilRespSender) SendOnce(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return nil, nil
}

// noChoicesSender answers with usage but no choice.
type noChoicesSender struct{}

func (noChoicesSender) SendOnce(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	return &openai.ChatCompletionResponse{Usage: openai.Usage{PromptTokens: 7, CompletionTokens: 0}}, nil
}

// sleepSender waits a known short time and then answers.
type sleepSender struct {
	d     time.Duration
	reply string
}

func (s sleepSender) SendOnce(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	time.Sleep(s.d)
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: s.reply}, FinishReason: openai.FinishReasonStop}},
		Usage:   openai.Usage{PromptTokens: 10, CompletionTokens: 5},
	}, nil
}

// captureLogs routes the global logger to a test hook at Info level and
// returns the hook plus a restore function.
func captureLogs() (*logrustest.Hook, func()) {
	hook := logrustest.NewGlobal()
	oldLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.InfoLevel)
	return hook, func() { logrus.SetLevel(oldLevel); hook.Reset() }
}

func diagEntries(hook *logrustest.Hook) []*logrus.Entry {
	var out []*logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Message == diagMessage {
			out = append(out, e)
		}
	}
	return out
}

func Test_FlowChat_modelCallDiagnostic(t *testing.T) {
	expired := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 20*time.Millisecond)
	}

	tests := []struct {
		name        string
		sender      Sender
		ctx         func() (context.Context, context.CancelFunc)
		wantOutcome string
		wantKind    string
		wantFinish  string
		wantTokens  [2]int
		wantFacts   *flowDraftFacts // nil when the outcome is not ok
	}{
		{
			name:        "ok with a draft",
			sender:      &chatSender{reply: flowGoodReply},
			wantOutcome: outcomeOK, wantFinish: "stop", wantTokens: [2]int{100, 20},
			wantFacts: &flowDraftFacts{hasDraft: true},
		},
		{
			name:        "ok message only",
			sender:      &chatSender{reply: `{"message":"Which channel?"}`},
			wantOutcome: outcomeOK, wantFinish: "stop", wantTokens: [2]int{100, 20},
			wantFacts: &flowDraftFacts{},
		},
		{
			name:        "ok draft that cannot be decoded is discarded",
			sender:      &chatSender{reply: `{"message":"m","draft":{"nodes":[]}}`},
			wantOutcome: outcomeOK, wantFinish: "stop", wantTokens: [2]int{100, 20},
			wantFacts: &flowDraftFacts{draftDiscarded: true},
		},
		{
			name:        "ok but every node removed by the type filter",
			sender:      &chatSender{reply: `{"message":"m","draft":{"nodes":[{"label":"a","type":"connect","option":{}}]},"assumptions":["x"]}`},
			wantOutcome: outcomeOK, wantFinish: "stop", wantTokens: [2]int{100, 20},
			wantFacts: &flowDraftFacts{emptyDraft: true},
		},
		{
			name:        "the handler's own deadline",
			sender:      &chatSender{err: context.DeadlineExceeded},
			wantOutcome: outcomeTimeout, wantFinish: "none",
		},
		{
			name:        "the caller's deadline is a different outcome",
			sender:      &chatSender{block: make(chan struct{})},
			ctx:         expired,
			wantOutcome: "llm_timeout", wantFinish: "none",
		},
		{
			name:        "truncated by length",
			sender:      &chatSender{reply: flowGoodReply, finish: openai.FinishReasonLength},
			wantOutcome: outcomeTruncated, wantFinish: "length", wantTokens: [2]int{100, 20},
		},
		{
			name:        "unparsable content",
			sender:      &chatSender{reply: "sorry"},
			wantOutcome: outcomeInvalidResponse, wantKind: invalidKindUnparsable, wantFinish: "stop", wantTokens: [2]int{100, 20},
		},
		{
			name:        "nil response",
			sender:      nilRespSender{},
			wantOutcome: outcomeInvalidResponse, wantKind: invalidKindNilResponse, wantFinish: "none",
		},
		{
			name:        "no choices",
			sender:      noChoicesSender{},
			wantOutcome: outcomeInvalidResponse, wantKind: invalidKindNoChoices, wantFinish: "none", wantTokens: [2]int{7, 0},
		},
		{
			name:        "provider 429",
			sender:      &chatSender{err: &openai.APIError{HTTPStatusCode: 429}},
			wantOutcome: "llm_rate_limit", wantFinish: "none",
		},
		{
			name:        "provider 500",
			sender:      &chatSender{err: &openai.APIError{HTTPStatusCode: 500}},
			wantOutcome: "llm_provider_5xx", wantFinish: "none",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook, restore := captureLogs()
			defer restore()

			h, _, cache := newFlowTestHandler(t, tt.sender, 200, 3, true)
			cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}
			_, _ = h.Chat(ctx, customerID, flowChatReq())

			lines := diagEntries(hook)
			if len(lines) != 1 {
				t.Fatalf("Wrong match. expect: exactly 1 diagnostic line, got: %d", len(lines))
			}
			d := lines[0].Data

			if d["outcome"] != tt.wantOutcome {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.wantOutcome, d["outcome"])
			}
			gotKind, hasKind := d["invalid_kind"]
			if tt.wantKind == "" && hasKind {
				t.Errorf("Wrong match. expect: no invalid_kind, got: %v", gotKind)
			}
			if tt.wantKind != "" && gotKind != tt.wantKind {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.wantKind, gotKind)
			}
			if d["finish_reason"] != tt.wantFinish {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.wantFinish, d["finish_reason"])
			}
			if d["prompt_tokens"] != tt.wantTokens[0] || d["completion_tokens"] != tt.wantTokens[1] {
				t.Errorf("Wrong match. expect: tokens %v, got: %v, %v", tt.wantTokens, d["prompt_tokens"], d["completion_tokens"])
			}
			for _, f := range diagNumericFields {
				if _, ok := d[f]; !ok {
					t.Errorf("Wrong match. expect: field %s on the line, got: missing", f)
				}
			}
			if d["json_mode"] != "schema" {
				t.Errorf("Wrong match. expect: schema, got: %v", d["json_mode"])
			}
			if d["allowed_types"] != 7 || d["user_turns"] != 1 || d["history_messages"] != 1 || d["current_draft_present"] != false {
				t.Errorf("Wrong match. expect: 7 types, 1 turn, 1 message, no draft, got: %v", d)
			}
			if tt.wantFacts == nil {
				for _, f := range []string{"has_draft", "draft_discarded", "empty_draft"} {
					if _, ok := d[f]; ok {
						t.Errorf("Wrong match. expect: no %s on a failed call, got: present", f)
					}
				}
				return
			}
			if d["has_draft"] != tt.wantFacts.hasDraft || d["draft_discarded"] != tt.wantFacts.draftDiscarded || d["empty_draft"] != tt.wantFacts.emptyDraft {
				t.Errorf("Wrong match. expect: %+v, got: has_draft=%v draft_discarded=%v empty_draft=%v",
					*tt.wantFacts, d["has_draft"], d["draft_discarded"], d["empty_draft"])
			}
		})
	}
}

// A call that never reached the model writes no model call line.
func Test_FlowChat_modelCallDiagnostic_refusalsWriteNothing(t *testing.T) {
	tests := []struct {
		name       string
		keyOK      bool
		mutate     func(r *flowbuilder.ChatRequest)
		expectIncr bool
		incrCount  int64
		incrErr    error
		fillSem    bool
	}{
		{name: "no key", keyOK: false},
		{name: "invalid request", keyOK: true, mutate: func(r *flowbuilder.ChatRequest) { r.Messages = nil }},
		{name: "no usable type", keyOK: true, mutate: func(r *flowbuilder.ChatRequest) { r.SupportedActionTypes = []string{"call", "no_such_type"} }},
		{name: "busy", keyOK: true, fillSem: true},
		{name: "counter failure", keyOK: true, expectIncr: true, incrErr: errors.New("redis down")},
		{name: "daily limit", keyOK: true, expectIncr: true, incrCount: 201},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook, restore := captureLogs()
			defer restore()

			s := &chatSender{reply: flowGoodReply}
			h, _, cache := newFlowTestHandler(t, s, 200, 1, tt.keyOK)
			if tt.expectIncr {
				cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(tt.incrCount, tt.incrErr)
			}
			if tt.fillSem {
				h.(*flowBuilderHandler).sem <- struct{}{}
			}
			req := flowChatReq()
			if tt.mutate != nil {
				tt.mutate(req)
			}

			if _, err := h.Chat(context.Background(), customerID, req); err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if got := len(diagEntries(hook)); got != 0 {
				t.Errorf("Wrong match. expect: no diagnostic line, got: %d", got)
			}
		})
	}
}

// A panic never produces a diagnostic line, in particular not a false ok.
func Test_FlowChat_modelCallDiagnostic_panicWritesNothing(t *testing.T) {
	hook, restore := captureLogs()
	defer restore()

	h, _, cache := newFlowTestHandler(t, &panicSender{}, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	func() {
		defer func() { _ = recover() }()
		_, _ = h.Chat(context.Background(), customerID, flowChatReq())
	}()

	if got := len(diagEntries(hook)); got != 0 {
		t.Errorf("Wrong match. expect: no diagnostic line, got: %d", got)
	}
}

// Nothing the customer typed, no model answer text, no draft content and no
// provider text may reach any value of the diagnostic line or of any other
// line of the call, and the line only has the fields on the closed list.
func Test_FlowChat_modelCallDiagnostic_neverCarriesText(t *testing.T) {
	answerWithSecret := `{"message":"` + secret + `","draft":{"nodes":[` +
		`{"label":"hello","type":"talk","option":{"text":"` + secret + `"},"next":"end"},` +
		`{"label":"end","type":"hangup","option":{}}]},"assumptions":["` + secret + `"]}`

	tests := []struct {
		name   string
		sender Sender
	}{
		{name: "ok with the marker in the answer and the draft", sender: &chatSender{reply: answerWithSecret}},
		{name: "provider 429 echoing the marker", sender: &chatSender{err: &openai.APIError{HTTPStatusCode: 429, Message: secret}}},
		{name: "provider 500 echoing the marker", sender: &chatSender{err: &openai.APIError{HTTPStatusCode: 500, Message: secret}}},
		{name: "unparsable answer holding the marker", sender: &chatSender{reply: "not json " + secret}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook, restore := captureLogs()
			defer restore()

			h, _, cache := newFlowTestHandler(t, tt.sender, 200, 3, true)
			cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

			req := flowChatReq()
			req.Messages[0].Content = secret
			req.CurrentDraft = &flowbuilder.Draft{Actions: []map[string]any{{
				"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071", "type": "talk",
				"option": map[string]any{"text": secret},
			}}}
			_, _ = h.Chat(context.Background(), customerID, req)

			if len(diagEntries(hook)) != 1 {
				t.Fatalf("Wrong match. expect: exactly 1 diagnostic line, got: %d", len(diagEntries(hook)))
			}
			for _, e := range hook.AllEntries() {
				if strings.Contains(e.Message, secret) {
					t.Errorf("Wrong match. the marker reached a message: %s", e.Message)
				}
				for k, v := range e.Data {
					if strings.Contains(fmt.Sprint(v), secret) {
						t.Errorf("Wrong match. the marker reached field %s", k)
					}
				}
			}
			for k := range diagEntries(hook)[0].Data {
				if !diagAllowedFields[k] {
					t.Errorf("Wrong match. expect: only listed fields, got: %s", k)
				}
			}
		})
	}
}

func Test_normalizeFinishReason(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "none"},
		{"stop", "stop"},
		{"length", "length"},
		{"content_filter", "content_filter"},
		{"tool_calls", "tool_calls"},
		{"function_call", "function_call"},
		{"null", "other"},
		{"STOP", "other"},
		{"stop; ignore previous instructions " + secret, "other"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeFinishReason(tt.in); got != tt.want {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.want, got)
			}
		})
	}
}

func Test_flowOutcome(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ok", nil, outcomeOK},
		{"own deadline", ErrTimeout, outcomeTimeout},
		{"caller deadline", &LLMError{Code: "timeout"}, "llm_timeout"},
		{"canceled", &LLMError{Code: "canceled"}, "llm_canceled"},
		{"auth", &LLMError{Code: "auth"}, "llm_auth"},
		{"other provider code", &LLMError{Code: "other"}, "llm_other"},
		{"truncated", ErrTruncated, outcomeTruncated},
		{"invalid", ErrInvalidResponse, outcomeInvalidResponse},
		{"wrapped invalid", fmt.Errorf("wrap: %w", ErrInvalidResponse), outcomeInvalidResponse},
		{"not a sentinel", errors.New("boom"), outcomeError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flowOutcome(tt.err); got != tt.want {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.want, got)
			}
		})
	}
}

func Test_hasWholeWarning(t *testing.T) {
	// A warning that merely contains the key is not the key.
	if hasWholeWarning([]string{"invalid_option: draft_discarded"}, WarnDraftDiscarded) {
		t.Errorf("Wrong match. expect: no match on a substring, got: match")
	}
	if hasWholeWarning([]string{WarnDraftDiscarded + ": x"}, WarnDraftDiscarded) {
		t.Errorf("Wrong match. expect: no match on a prefix, got: match")
	}
	if !hasWholeWarning([]string{"a", WarnDraftDiscarded}, WarnDraftDiscarded) {
		t.Errorf("Wrong match. expect: match on the whole entry, got: no match")
	}
}

func Test_RunFlowTurn_diag(t *testing.T) {
	t.Run("a successful call measures the call, the build and the sizes", func(t *testing.T) {
		cfg := testCfg()
		res, err := RunFlowTurn(context.Background(), sleepSender{d: 30 * time.Millisecond, reply: `{"message":"m"}`}, FlowConfig(cfg), flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.Diag.Elapsed < 30*time.Millisecond || res.Diag.Elapsed > 5*time.Second {
			t.Errorf("Wrong match. expect: about 30ms, got: %v", res.Diag.Elapsed)
		}
		if res.Diag.BuildElapsed < 0 || res.Diag.SystemChars <= 0 || res.Diag.RequestChars <= res.Diag.SystemChars || res.Diag.SchemaBytes <= 0 {
			t.Errorf("Wrong match. expect: positive sizes and a request larger than the prompt, got: %+v", res.Diag)
		}
		if res.Diag.ResponseChars != len(`{"message":"m"}`) || res.Diag.UserTurns != 1 || res.Diag.HistoryMessages != 1 || res.Diag.InvalidKind != "" {
			t.Errorf("Wrong match. expect: response 15 chars, 1 turn, 1 message, no invalid kind, got: %+v", res.Diag)
		}
	})

	t.Run("no schema is built when the mode does not send one", func(t *testing.T) {
		cfg := FlowConfig(testCfg())
		cfg.JSONMode = JSONModeNone
		res, err := RunFlowTurn(context.Background(), sleepSender{reply: `{"message":"m"}`}, cfg, flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
		if err != nil || res.Diag.SchemaBytes != 0 {
			t.Errorf("Wrong match. expect: ok and 0 schema bytes, got: %v, %d", err, res.Diag.SchemaBytes)
		}
	})

	t.Run("a timeout still returns a result with the timing", func(t *testing.T) {
		cfg := FlowConfig(testCfg())
		cfg.LLMTimeout = 30 * time.Millisecond
		res, err := RunFlowTurn(context.Background(), &chatSender{block: make(chan struct{})}, cfg, flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("Wrong match. expect: ErrTimeout, got: %v", err)
		}
		if res == nil || res.Diag.Elapsed < 25*time.Millisecond || res.Usage.PromptTokens != 0 {
			t.Errorf("Wrong match. expect: a result with the elapsed time and no usage, got: %+v", res)
		}
	})

	t.Run("a provider error still returns a result", func(t *testing.T) {
		res, err := RunFlowTurn(context.Background(), &chatSender{err: &openai.APIError{HTTPStatusCode: 500}}, FlowConfig(testCfg()), flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
		var llm *LLMError
		if !errors.As(err, &llm) || llm.Code != "provider_5xx" {
			t.Fatalf("Wrong match. expect: provider_5xx, got: %v", err)
		}
		if res == nil || res.Diag.SystemChars <= 0 {
			t.Errorf("Wrong match. expect: a result with sizes, got: %+v", res)
		}
	})

	t.Run("the guards still return no result", func(t *testing.T) {
		if res, err := RunFlowTurn(context.Background(), nil, FlowConfig(testCfg()), flowChatReq(), nil); res != nil || err == nil {
			t.Errorf("Wrong match. expect: nil result and an error, got: %v, %v", res, err)
		}
		if res, err := RunFlowTurn(context.Background(), &chatSender{}, FlowConfig(testCfg()), nil, nil); res != nil || err == nil {
			t.Errorf("Wrong match. expect: nil result and an error, got: %v, %v", res, err)
		}
	})
}

// recordSender records the request and answers with a fixed reply.
type recordSender struct {
	req   *openai.ChatCompletionRequest
	reply string
}

func (s *recordSender) SendOnce(_ context.Context, r *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	s.req = r
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: s.reply}, FinishReason: openai.FinishReasonStop}},
	}, nil
}

// The sizes are counted in runes, not bytes, and the turn count only counts
// user messages. Korean text makes a byte count differ from a rune count.
func Test_RunFlowTurn_diag_countsRunesAndUserTurns(t *testing.T) {
	reply := `{"message":"안녕하세요 무엇을 도와드릴까요"}`
	s := &recordSender{reply: reply}
	req := &flowbuilder.ChatRequest{
		Messages: []flowbuilder.Message{
			{Role: flowbuilder.RoleUser, Content: "인사 플로우를 만들어 주세요"},
			{Role: flowbuilder.RoleAssistant, Content: "어떤 채널인가요"},
			{Role: flowbuilder.RoleUser, Content: "전화입니다"},
		},
		SupportedActionTypes: []string{"talk", "hangup"},
	}

	res, err := RunFlowTurn(context.Background(), s, FlowConfig(testCfg()), req, FlowAllowedTypes(req.SupportedActionTypes))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	wantRequest := 0
	for _, m := range s.req.Messages {
		wantRequest += len([]rune(m.Content))
	}
	if res.Diag.RequestChars != wantRequest {
		t.Errorf("Wrong match. expect: %d runes, got: %d", wantRequest, res.Diag.RequestChars)
	}
	if want := len([]rune(s.req.Messages[0].Content)); res.Diag.SystemChars != want {
		t.Errorf("Wrong match. expect: %d, got: %d", want, res.Diag.SystemChars)
	}
	if want := len([]rune(reply)); res.Diag.ResponseChars != want || want == len(reply) {
		t.Errorf("Wrong match. expect: %d runes (not %d bytes), got: %d", want, len(reply), res.Diag.ResponseChars)
	}
	if res.Diag.UserTurns != 2 || res.Diag.HistoryMessages != 3 {
		t.Errorf("Wrong match. expect: 2 user turns of 3 messages, got: %d of %d", res.Diag.UserTurns, res.Diag.HistoryMessages)
	}
}

// The mode name, the schema build condition and the request format always
// agree with each other.
func Test_flowJSONMode_agreesWithRequestFormat(t *testing.T) {
	tests := []struct {
		name       string
		mode       JSONMode
		wantName   string
		wantSchema bool
	}{
		{"none", JSONModeNone, "none", false},
		{"object", JSONModeObject, "object", false},
		{"schema", JSONModeSchema, "schema", true},
		{"empty falls to schema as before", JSONMode(""), "schema", true},
		{"unknown falls to schema as before", JSONMode("x"), "schema", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flowJSONModeName(tt.mode); got != tt.wantName {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.wantName, got)
			}
			if got := flowUsesSchema(tt.mode); got != tt.wantSchema {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.wantSchema, got)
			}
			sent := flowResponseFormat(tt.mode, "{}")
			if sentSchema := sent != nil && sent.Type == openai.ChatCompletionResponseFormatTypeJSONSchema; sentSchema != tt.wantSchema {
				t.Errorf("Wrong match. expect: schema sent %v, got: %v", tt.wantSchema, sentSchema)
			}

			cfg := FlowConfig(testCfg())
			cfg.JSONMode = tt.mode
			res, err := RunFlowTurn(context.Background(), &recordSender{reply: `{"message":"m"}`}, cfg, flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if (res.Diag.SchemaBytes > 0) != tt.wantSchema {
				t.Errorf("Wrong match. expect: schema built %v, got: %d bytes", tt.wantSchema, res.Diag.SchemaBytes)
			}
		})
	}
}

// The timing and settings fields carry the values they are defined to carry.
func Test_logModelCall_values(t *testing.T) {
	hook, restore := captureLogs()
	defer restore()

	h, _, _ := newFlowTestHandler(t, &chatSender{}, 200, 3, true)
	fh := h.(*flowBuilderHandler)
	fh.cfg.Model = "model-x"
	fh.cfg.ReasoningEffort = "low"
	fh.cfg.MaxOutputTokens = 1234
	fh.cfg.LLMTimeout = 7 * time.Second
	fh.cfg.JSONMode = JSONModeObject

	now := time.Now()
	start := now.Add(-300 * time.Millisecond)
	callStart := now.Add(-100 * time.Millisecond)
	res := &FlowTurnResult{Diag: FlowTurnDiag{Elapsed: 11 * time.Millisecond, BuildElapsed: 7 * time.Millisecond}}
	fh.logModelCall(logrus.NewEntry(logrus.StandardLogger()), res, nil, start, callStart, nil)

	lines := diagEntries(hook)
	if len(lines) != 1 {
		t.Fatalf("Wrong match. expect: 1 line, got: %d", len(lines))
	}
	d := lines[0].Data
	if d["elapsed_ms"] != int64(11) || d["build_ms"] != int64(7) {
		t.Errorf("Wrong match. expect: 11 and 7, got: %v and %v", d["elapsed_ms"], d["build_ms"])
	}
	if pre, _ := d["pre_call_ms"].(int64); pre < 195 || pre > 205 {
		t.Errorf("Wrong match. expect: about 200, got: %v", d["pre_call_ms"])
	}
	if chat, _ := d["chat_ms"].(int64); chat < 295 || chat > 5000 {
		t.Errorf("Wrong match. expect: about 300, got: %v", d["chat_ms"])
	}
	if d["model"] != "model-x" || d["reasoning_effort"] != "low" || d["max_tokens"] != 1234 || d["llm_timeout_ms"] != int64(7000) || d["json_mode"] != "object" {
		t.Errorf("Wrong match. expect: the settings in force, got: %v", d)
	}
	if d["finish_reason"] != "none" {
		t.Errorf("Wrong match. expect: none, got: %v", d["finish_reason"])
	}
}

// Every Diag value reaches its own log field. The values are all different so
// a swapped or constant mapping cannot pass.
func Test_logModelCall_mapsEveryDiagField(t *testing.T) {
	hook, restore := captureLogs()
	defer restore()

	h, _, _ := newFlowTestHandler(t, &chatSender{}, 200, 3, true)
	fh := h.(*flowBuilderHandler)

	res := &FlowTurnResult{
		FinishReason: "length",
		Usage:        openai.Usage{PromptTokens: 101, CompletionTokens: 202},
		Diag: FlowTurnDiag{
			SystemChars: 11, RequestChars: 22, SchemaBytes: 33, ResponseChars: 44,
			UserTurns: 3, HistoryMessages: 5, AllowedTypes: 7, CurrentDraftPresent: true,
			InvalidKind: invalidKindNoChoices,
		},
	}
	now := time.Now()
	fh.logModelCall(logrus.NewEntry(logrus.StandardLogger()), res, ErrTruncated, now, now, nil)

	lines := diagEntries(hook)
	if len(lines) != 1 {
		t.Fatalf("Wrong match. expect: 1 line, got: %d", len(lines))
	}
	d := lines[0].Data
	want := map[string]any{
		"system_chars": 11, "request_chars": 22, "schema_bytes": 33, "response_chars": 44,
		"user_turns": 3, "history_messages": 5, "allowed_types": 7, "current_draft_present": true,
		"prompt_tokens": 101, "completion_tokens": 202, "finish_reason": "length",
		"outcome": outcomeTruncated, "invalid_kind": invalidKindNoChoices,
	}
	for k, v := range want {
		if d[k] != v {
			t.Errorf("Wrong match. field %s expect: %v, got: %v", k, v, d[k])
		}
	}
}

// A nil result means the model was never called: no line, no panic.
func Test_logModelCall_nilResultWritesNothing(t *testing.T) {
	hook, restore := captureLogs()
	defer restore()

	h, _, _ := newFlowTestHandler(t, &chatSender{}, 200, 3, true)
	now := time.Now()
	h.(*flowBuilderHandler).logModelCall(logrus.NewEntry(logrus.StandardLogger()), nil, errors.New("x"), now, now, nil)

	if got := len(diagEntries(hook)); got != 0 {
		t.Errorf("Wrong match. expect: no line, got: %d", got)
	}
}

// pre_call_ms covers the work before the model call (here a slow counter), and
// build_ms stays far below it.
func Test_FlowChat_modelCallDiagnostic_preCallCoversTheCounter(t *testing.T) {
	hook, restore := captureLogs()
	defer restore()

	h, _, cache := newFlowTestHandler(t, &chatSender{reply: flowGoodReply}, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).
		DoAndReturn(func(context.Context, uuid.UUID, time.Duration) (int64, error) {
			time.Sleep(60 * time.Millisecond)
			return 1, nil
		})

	if _, err := h.Chat(context.Background(), customerID, flowChatReq()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	lines := diagEntries(hook)
	if len(lines) != 1 {
		t.Fatalf("Wrong match. expect: 1 line, got: %d", len(lines))
	}
	d := lines[0].Data
	pre, _ := d["pre_call_ms"].(int64)
	build, _ := d["build_ms"].(int64)
	chat, _ := d["chat_ms"].(int64)
	if pre < 60 || pre > 2000 || build >= 1000 || chat < pre {
		t.Errorf("Wrong match. expect: pre >= 60, build small, chat >= pre, got: pre=%d build=%d chat=%d", pre, build, chat)
	}
}

// The request that goes out is the one the mode defines: nothing, a JSON
// object, or the JSON schema built from the allowed types.
func Test_RunFlowTurn_requestFormatPerMode(t *testing.T) {
	allowed := FlowAllowedTypes(flowChatReq().SupportedActionTypes)

	tests := []struct {
		name     string
		mode     JSONMode
		wantType openai.ChatCompletionResponseFormatType
	}{
		{"none", JSONModeNone, ""},
		{"object", JSONModeObject, openai.ChatCompletionResponseFormatTypeJSONObject},
		{"schema", JSONModeSchema, openai.ChatCompletionResponseFormatTypeJSONSchema},
		{"empty falls to schema", JSONMode(""), openai.ChatCompletionResponseFormatTypeJSONSchema},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &recordSender{reply: `{"message":"m"}`}
			cfg := FlowConfig(testCfg())
			cfg.JSONMode = tt.mode
			if _, err := RunFlowTurn(context.Background(), s, cfg, flowChatReq(), allowed); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			got := s.req.ResponseFormat
			if tt.wantType == "" {
				if got != nil {
					t.Errorf("Wrong match. expect: no response format, got: %+v", got)
				}
				return
			}
			if got == nil || got.Type != tt.wantType {
				t.Fatalf("Wrong match. expect: %s, got: %+v", tt.wantType, got)
			}
			if tt.wantType == openai.ChatCompletionResponseFormatTypeJSONSchema {
				if got.JSONSchema == nil || got.JSONSchema.Name != FlowResponseSchemaName || string(got.JSONSchema.Schema.(json.RawMessage)) != FlowResponseSchema(allowed) {
					t.Errorf("Wrong match. expect: the schema built from the allowed types, got: %+v", got.JSONSchema)
				}
			}
		})
	}
}

// build_ms is the time spent before the call, not the call itself: a call
// that takes a known 300ms leaves the build far below it.
func Test_RunFlowTurn_diag_buildIsShort(t *testing.T) {
	res, err := RunFlowTurn(context.Background(), sleepSender{d: 300 * time.Millisecond, reply: `{"message":"m"}`}, FlowConfig(testCfg()), flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Diag.BuildElapsed >= 200*time.Millisecond || res.Diag.Elapsed < 300*time.Millisecond {
		t.Errorf("Wrong match. expect: build well below the 300ms call, got: build=%v call=%v", res.Diag.BuildElapsed, res.Diag.Elapsed)
	}
}

// The values an operator greps for are fixed strings, written out here on
// purpose so a changed constant cannot pass unnoticed.
func Test_diagLiterals(t *testing.T) {
	tests := []struct{ got, want string }{
		{outcomeOK, "ok"}, {outcomeTimeout, "timeout"}, {outcomeTruncated, "truncated"},
		{outcomeInvalidResponse, "invalid_response"}, {outcomeError, "error"},
		{invalidKindNilResponse, "nil_response"}, {invalidKindNoChoices, "no_choices"}, {invalidKindUnparsable, "unparsable"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("Wrong match. expect: %s, got: %s", tt.want, tt.got)
		}
	}
}

// The truncated line carries the length of what the model wrote, and a request
// that holds a current draft says so (an empty draft counts as present).
func Test_RunFlowTurn_diag_truncatedAndDraftPresent(t *testing.T) {
	reply := "abcdef"
	req := flowChatReq()
	req.CurrentDraft = &flowbuilder.Draft{}
	res, err := RunFlowTurn(context.Background(), &chatSender{reply: reply, finish: openai.FinishReasonLength}, FlowConfig(testCfg()), req, FlowAllowedTypes(req.SupportedActionTypes))
	if !errors.Is(err, ErrTruncated) {
		t.Fatalf("Wrong match. expect: ErrTruncated, got: %v", err)
	}
	if res.Diag.ResponseChars != len(reply) || !res.Diag.CurrentDraftPresent {
		t.Errorf("Wrong match. expect: 6 chars and a draft present, got: %+v", res.Diag)
	}
	if res.FinishReason != "length" {
		t.Errorf("Wrong match. expect: length, got: %s", res.FinishReason)
	}
}

// allowed_types counts the types that are really offered, after the filter,
// not the types the client asked for.
func Test_RunFlowTurn_diag_allowedTypesIsAfterTheFilter(t *testing.T) {
	req := flowChatReq()
	req.SupportedActionTypes = []string{"talk", "hangup", "no_such_type"}
	allowed := FlowAllowedTypes(req.SupportedActionTypes)

	res, err := RunFlowTurn(context.Background(), &recordSender{reply: `{"message":"m"}`}, FlowConfig(testCfg()), req, allowed)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Diag.AllowedTypes != len(allowed) || res.Diag.AllowedTypes >= len(req.SupportedActionTypes) {
		t.Errorf("Wrong match. expect: the filtered count below %d, got: %d", len(req.SupportedActionTypes), res.Diag.AllowedTypes)
	}
}
