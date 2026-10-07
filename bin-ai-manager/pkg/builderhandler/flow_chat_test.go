package builderhandler

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	fmaction "monorepo/bin-flow-manager/models/action"
)

const flowGoodReply = `{"message":"Here is the Flow.","draft":{"nodes":[` +
	`{"label":"hello","type":"talk","option":{"text":"Hello"},"next":"sms"},` +
	`{"label":"sms","type":"message_send","option":{"text":"Hi","destinations":[{"type":"tel","target":"+15551234567"}]},"next":"end"},` +
	`{"label":"end","type":"hangup","option":{}}]},"assumptions":["a1"]}`

func flowChatReq() *flowbuilder.ChatRequest {
	return &flowbuilder.ChatRequest{
		Messages:             []flowbuilder.Message{{Role: flowbuilder.RoleUser, Content: "I want a greeting flow"}},
		SupportedActionTypes: []string{"talk", "message_send", "hangup", "branch", "digits_receive", "queue_join", "stop"},
	}
}

func newFlowTestHandler(t *testing.T, s Sender, dailyLimit, maxConcurrent int, keyConfigured bool) (FlowBuilderHandler, BuilderHandler, *cachehandler.MockCacheHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	cache := cachehandler.NewMockCacheHandler(mc)
	assistant, flow := NewBuilderHandlers(s, cache, testCfg(), Options{
		KeyConfigured: keyConfigured,
		DailyLimit:    dailyLimit,
		MaxConcurrent: maxConcurrent,
	})
	return flow, assistant, cache
}

func Test_FlowChat_success(t *testing.T) {
	s := &chatSender{reply: flowGoodReply}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	// Only the flow counter may be touched. The strict mock fails on any call
	// to the Assistant counter.
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	before := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultOK))
	assistantBefore := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultOK))

	resp, err := h.Chat(context.Background(), customerID, flowChatReq())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resp.Draft == nil || len(resp.Draft.Actions) != 3 {
		t.Fatalf("Wrong match. expect: 3 actions, got: %+v", resp.Draft)
	}
	if resp.Draft.Actions[0]["type"] != "talk" {
		t.Errorf("Wrong match. expect: talk first, got: %v", resp.Draft.Actions[0]["type"])
	}
	if len(resp.Draft.Positions) != 3 || len(resp.Draft.Labels) != 3 {
		t.Errorf("Wrong match. expect: 3 positions and labels, got: %d, %d", len(resp.Draft.Positions), len(resp.Draft.Labels))
	}
	if len(resp.SensitiveNodes) != 1 || resp.SensitiveNodes[0].String() != resp.Draft.Actions[1]["id"] {
		t.Errorf("Wrong match. expect: exactly the message_send node as sensitive, got: %v", resp.SensitiveNodes)
	}
	if len(resp.Assumptions) != 1 {
		t.Errorf("Wrong match. expect: 1 assumption, got: %v", resp.Assumptions)
	}

	if got := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultOK)) - before; got != 1 {
		t.Errorf("Wrong match. expect: flow ok counter +1, got: %v", got)
	}
	if got := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultOK)) - assistantBefore; got != 0 {
		t.Errorf("Wrong match. expect: assistant counter untouched, got: %v", got)
	}
}

func Test_FlowChat_messageOnlyHasNoDraft(t *testing.T) {
	s := &chatSender{reply: `{"message":"Which channel?"}`}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	resp, err := h.Chat(context.Background(), customerID, flowChatReq())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resp.Draft != nil || resp.Message != "Which channel?" {
		t.Errorf("Wrong match. expect: message only, got: %+v", resp)
	}
}

func Test_FlowChat_allUnsupportedGivesEmptyDraft(t *testing.T) {
	// The model uses a type outside this request's allowed set.
	s := &chatSender{reply: `{"message":"m","draft":{"nodes":[{"label":"a","type":"connect","option":{}}]},"assumptions":["x"]}`}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	resp, err := h.Chat(context.Background(), customerID, flowChatReq())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if resp.Draft != nil || resp.Assumptions != nil {
		t.Errorf("Wrong match. expect: no draft and no assumptions, got: %+v", resp)
	}
	if !containsExact(resp.DraftWarnings, flowbuilder.WarningEmptyDraft) {
		t.Errorf("Wrong match. expect: empty_draft in %v", resp.DraftWarnings)
	}
}

func Test_FlowChat_refusals(t *testing.T) {
	tests := []struct {
		name       string
		keyOK      bool
		mutate     func(r *flowbuilder.ChatRequest)
		expectIncr bool
		incrCount  int64
		incrErr    error
		wantReason string
		wantResult string
	}{
		{
			name:       "key not configured",
			keyOK:      false,
			wantReason: builder.ReasonUnavailable,
			wantResult: resultUnavailable,
		},
		{
			name:       "validation failure is never counted",
			keyOK:      true,
			mutate:     func(r *flowbuilder.ChatRequest) { r.Messages = nil },
			wantReason: builder.ReasonInvalidArgument,
			wantResult: resultInvalidArgument,
		},
		{
			name:  "no usable supported type is never counted",
			keyOK: true,
			mutate: func(r *flowbuilder.ChatRequest) {
				r.SupportedActionTypes = []string{"call", "email_send", "no_such_type"}
			},
			wantReason: builder.ReasonInvalidArgument,
			wantResult: resultInvalidArgument,
		},
		{
			name:       "daily limit",
			keyOK:      true,
			expectIncr: true,
			incrCount:  201,
			wantReason: builder.ReasonDailyLimit,
			wantResult: resultDailyLimit,
		},
		{
			name:       "counter failure fails closed",
			keyOK:      true,
			expectIncr: true,
			incrErr:    errors.New("redis down"),
			wantReason: builder.ReasonUnavailable,
			wantResult: resultUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chatSender{reply: flowGoodReply}
			h, _, cache := newFlowTestHandler(t, s, 200, 3, tt.keyOK)
			if tt.expectIncr {
				cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(tt.incrCount, tt.incrErr)
			}
			req := flowChatReq()
			if tt.mutate != nil {
				tt.mutate(req)
			}
			before := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(tt.wantResult))

			_, err := h.Chat(context.Background(), customerID, req)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if _, reason := reasonOf(t, err); reason != tt.wantReason {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.wantReason, reason)
			}
			if s.callCount() != 0 {
				t.Errorf("Wrong match. expect: no model call, got: %d", s.callCount())
			}
			if got := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(tt.wantResult)) - before; got != 1 {
				t.Errorf("Wrong match. expect: flow %s counter +1, got: %v", tt.wantResult, got)
			}
		})
	}
}

func Test_FlowChat_sharesTheSemaphoreWithTheAssistantBuilder(t *testing.T) {
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	s := &chatSender{reply: goodReply, block: block, entered: entered}
	flow, assistant, cache := newFlowTestHandler(t, s, 200, 1, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	done := make(chan error, 1)
	go func() {
		_, err := assistant.Chat(context.Background(), customerID, chatReq())
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatalf("the assistant call never started")
	}

	// The single slot is held by the Assistant Builder, so the Flow Builder is
	// refused at once, before it spends any allowance (strict mock: no
	// counter expectation for the flow key).
	_, err := flow.Chat(context.Background(), customerID, flowChatReq())
	if err == nil {
		t.Fatalf("Wrong match. expect: busy, got: nil")
	}
	if _, reason := reasonOf(t, err); reason != builder.ReasonBusy {
		t.Errorf("Wrong match. expect: %s, got: %s", builder.ReasonBusy, reason)
	}

	close(block)
	if err := <-done; err != nil {
		t.Errorf("Wrong match. expect: assistant ok, got: %v", err)
	}
}

func Test_FlowChat_turnErrors(t *testing.T) {
	tests := []struct {
		name       string
		sender     *chatSender
		wantReason string
	}{
		{"truncated", &chatSender{reply: flowGoodReply, finish: openai.FinishReasonLength}, builder.ReasonResponseInvalid},
		{"no object", &chatSender{reply: "sorry"}, builder.ReasonResponseInvalid},
		{"provider error", &chatSender{err: &openai.APIError{HTTPStatusCode: 500, Message: secret}}, builder.ReasonResponseInvalid},
		{"llm timeout", &chatSender{err: context.DeadlineExceeded}, builder.ReasonTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, cache := newFlowTestHandler(t, tt.sender, 200, 3, true)
			cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

			_, err := h.Chat(context.Background(), customerID, flowChatReq())
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if _, reason := reasonOf(t, err); reason != tt.wantReason {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.wantReason, reason)
			}
			if strings.Contains(err.Error(), secret) {
				t.Errorf("Wrong match. the provider text reached the error: %v", err)
			}
		})
	}
}

// Nothing the customer typed, no draft option and no provider text may reach
// a log line (design doc 5.1). The request carries the marker in the message,
// in a draft option value and in the provider error.
func Test_FlowChat_logsNeverCarryCustomerInput(t *testing.T) {
	var buf bytes.Buffer
	hook := logrustest.NewGlobal()
	oldOut := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.DebugLevel)
	defer func() { logrus.SetOutput(oldOut); logrus.SetLevel(logrus.InfoLevel); hook.Reset() }()

	s := &chatSender{err: &openai.APIError{HTTPStatusCode: 400, Message: secret}}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	req := flowChatReq()
	req.Messages[0].Content = secret
	req.CurrentDraft = &flowbuilder.Draft{Actions: []map[string]any{{
		"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071", "type": "talk",
		"option": map[string]any{"text": secret},
	}}}

	_, _ = h.Chat(context.Background(), customerID, req)

	if strings.Contains(buf.String(), secret) {
		t.Errorf("Wrong match. the marker reached a log line: %s", buf.String())
	}
	for _, e := range hook.AllEntries() {
		if strings.Contains(e.Message, secret) {
			t.Errorf("Wrong match. the marker reached a log entry: %s", e.Message)
		}
	}
}

func Test_FlowChat_requestToModelCarriesGeneratedCatalogAndLabelGraph(t *testing.T) {
	var got *openai.ChatCompletionRequest
	s := &captureSender{reply: `{"message":"m"}`, got: &got}
	h, _, cache := newFlowTestHandler(t, s, 200, 3, true)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	req := flowChatReq()
	req.SupportedActionTypes = []string{"talk", "hangup", "call"} // call is structurally excluded
	req.CurrentDraft = &flowbuilder.Draft{
		Actions: []map[string]any{
			{"id": "6c73ff34-7f4c-11ec-b4d5-5b94d40e4071", "type": "talk", "option": map[string]any{"text": "hi"}, "next_id": "841c5fa2-f0c2-11ee-834f-53b2b00ec88d"},
			{"id": "841c5fa2-f0c2-11ee-834f-53b2b00ec88d", "type": "hangup"},
		},
		Labels: map[string]string{"6c73ff34-7f4c-11ec-b4d5-5b94d40e4071": "greet"},
	}

	if _, err := h.Chat(context.Background(), customerID, req); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	system := got.Messages[0].Content
	if !strings.Contains(system, "- type talk") || !strings.Contains(system, "- type hangup") {
		t.Errorf("Wrong match. expect: talk and hangup in the catalog")
	}
	if strings.Contains(system, "- type call ") || strings.Contains(system, "- type connect") {
		t.Errorf("Wrong match. expect: no excluded or unsupported type in the catalog")
	}
	last := got.Messages[len(got.Messages)-1].Content
	if !strings.Contains(last, `"label":"greet"`) || !strings.Contains(last, `"next":"n1"`) {
		t.Errorf("Wrong match. expect: label graph with labels, got: %s", last)
	}
	if strings.Contains(last, "6c73ff34") || strings.Contains(last, "841c5fa2") {
		t.Errorf("Wrong match. expect: no UUID shown to the model, got: %s", last)
	}
	if got.MaxTokens < flowDefaultMaxOutputTokens {
		t.Errorf("Wrong match. expect: >= %d output tokens, got: %d", flowDefaultMaxOutputTokens, got.MaxTokens)
	}
}

type captureSender struct {
	reply string
	got   **openai.ChatCompletionRequest
}

func (c *captureSender) SendOnce(_ context.Context, req *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	*c.got = req
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: c.reply}, FinishReason: openai.FinishReasonStop}},
	}, nil
}

func Test_RecordFlowPanic_countsOnlyTheFlowSeries(t *testing.T) {
	flowBefore := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultInternal))
	assistantBefore := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultInternal))

	RecordFlowPanic()

	if got := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultInternal)) - flowBefore; got != 1 {
		t.Errorf("Wrong match. expect: flow +1, got: %v", got)
	}
	if got := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultInternal)) - assistantBefore; got != 0 {
		t.Errorf("Wrong match. expect: assistant untouched, got: %v", got)
	}
}

func Test_FlowAllowedTypes(t *testing.T) {
	got := FlowAllowedTypes([]string{"talk", "talk", "call", "email_send", "goto", "no_such_type", "connect", "hangup"})
	want := []fmaction.Type{fmaction.TypeConnect, fmaction.TypeHangup, fmaction.TypeTalk}
	if len(got) != len(want) {
		t.Fatalf("Wrong match. expect: %v, got: %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Wrong match. expect: %v, got: %v", want, got)
		}
	}
}
