package builderhandler

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
)

// secret is a user-input marker. It must never appear in any log line, error
// text or metric label: a provider 4xx can echo the prompt back, and the body
// is the customer's own business description.
const secret = "SECRET-INPUT-STRING-do-not-log"

var customerID = uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

type chatSender struct {
	mu      sync.Mutex
	calls   int
	reply   string
	finish  openai.FinishReason
	err     error
	block   chan struct{} // when set, SendOnce waits for it (or ctx)
	entered chan struct{} // signalled once SendOnce is running
}

func (f *chatSender) SendOnce(ctx context.Context, _ *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.entered != nil {
		select {
		case f.entered <- struct{}{}:
		default:
		}
	}
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	fin := f.finish
	if fin == "" {
		fin = openai.FinishReasonStop
	}
	return &openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: f.reply}, FinishReason: fin}},
		Usage:   openai.Usage{PromptTokens: 100, CompletionTokens: 20},
	}, nil
}

func (f *chatSender) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

const goodReply = `{"message":"What does the assistant do when nobody answers?"}`

func chatReq() *builder.ChatRequest {
	return &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleUser, Content: "I run a clinic"}}}
}

func testCfg() Config {
	cfg := DefaultConfig()
	cfg.LLMTimeout = 2 * time.Second
	return cfg
}

// newTestHandler builds a handler with a strict mock cache.
func newTestHandler(t *testing.T, s Sender, cfg Config, dailyLimit, maxConcurrent int, enabled, keyConfigured bool) (BuilderHandler, *cachehandler.MockCacheHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	cache := cachehandler.NewMockCacheHandler(mc)
	h := NewBuilderHandler(s, cache, cfg, Options{
		Enabled:       enabled,
		KeyConfigured: keyConfigured,
		DailyLimit:    dailyLimit,
		MaxConcurrent: maxConcurrent,
	})
	return h, cache
}

func reasonOf(t *testing.T, err error) (cerrors.Status, string) {
	t.Helper()
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expected a *VoipbinError, got %T: %v", err, err)
	}
	return ve.Status, ve.Reason
}

func Test_Chat_success(t *testing.T) {
	s := &chatSender{reply: goodReply}
	h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	resp, err := h.Chat(context.Background(), customerID, chatReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Message == "" {
		t.Error("the model's message must be returned")
	}
	if s.callCount() != 1 {
		t.Errorf("exactly one LLM call is expected, got %d", s.callCount())
	}
}

func Test_Chat_killSwitchAndKey(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		keyConfigured bool
		reason        string
		status        cerrors.Status
	}{
		{"disabled", false, true, builder.ReasonDisabled, cerrors.StatusUnavailable},
		{"enabled but no key", true, false, builder.ReasonUnavailable, cerrors.StatusUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chatSender{reply: goodReply}
			h, _ := newTestHandler(t, s, testCfg(), 200, 3, tt.enabled, tt.keyConfigured) // no cache call expected
			_, err := h.Chat(context.Background(), customerID, chatReq())
			st, reason := reasonOf(t, err)
			if reason != tt.reason || st != tt.status {
				t.Errorf("got %v/%s, want %v/%s", st, reason, tt.status, tt.reason)
			}
			if s.callCount() != 0 {
				t.Error("no LLM call may happen")
			}
		})
	}
}

// A request that fails validation is not counted: the customer did nothing the
// platform paid for.
func Test_Chat_invalidRequestIsNotCounted(t *testing.T) {
	s := &chatSender{reply: goodReply}
	h, _ := newTestHandler(t, s, testCfg(), 200, 3, true, true) // strict mock: any cache call fails the test
	bad := &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleAssistant, Content: "last is assistant"}}}

	_, err := h.Chat(context.Background(), customerID, bad)
	_, reason := reasonOf(t, err)
	if reason != builder.ReasonInvalidArgument {
		t.Errorf("reason: got %s", reason)
	}
	if s.callCount() != 0 {
		t.Error("no LLM call may happen")
	}
}

func Test_Chat_dailyLimitBoundary(t *testing.T) {
	// The limit is inclusive: the 200th call passes, the 201st is refused.
	tests := []struct {
		name  string
		count int64
		ok    bool
	}{
		{"one below the limit", 199, true},
		{"exactly at the limit", 200, true},
		{"one over the limit", 201, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chatSender{reply: goodReply}
			h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
			cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(tt.count, nil)

			_, err := h.Chat(context.Background(), customerID, chatReq())
			if tt.ok {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			st, reason := reasonOf(t, err)
			if reason != builder.ReasonDailyLimit || st != cerrors.StatusResourceExhausted {
				t.Errorf("got %v/%s", st, reason)
			}
			if s.callCount() != 0 {
				t.Error("an over-limit call must not reach the LLM")
			}
		})
	}
}

// Redis failing must fail CLOSED: an unmetered Builder is unbounded platform
// spend.
func Test_Chat_redisErrorFailsClosed(t *testing.T) {
	s := &chatSender{reply: goodReply}
	h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(0), errors.New("redis down "+secret))

	_, err := h.Chat(context.Background(), customerID, chatReq())
	st, reason := reasonOf(t, err)
	if reason != builder.ReasonUnavailable || st != cerrors.StatusUnavailable {
		t.Errorf("got %v/%s", st, reason)
	}
	if s.callCount() != 0 {
		t.Error("the LLM must not be called when the counter is unavailable")
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("the Redis error text must not be carried into the returned error")
	}
}

// Sentinel mapping. Every provider-side failure is counted: the counter is
// incremented before the call (design 4.3), so a customer who retries against a
// failing provider spends the daily limit. That cost is a decision recorded in
// the plan (deviation 5, open question 6), pinned here so changing it is a
// conscious act.
func Test_Chat_turnErrorsMapToReasonsAndAreCounted(t *testing.T) {
	tests := []struct {
		name   string
		sender *chatSender
		reason string
		status cerrors.Status
	}{
		{"provider error", &chatSender{err: &openai.APIError{HTTPStatusCode: 500}}, builder.ReasonResponseInvalid, cerrors.StatusUnavailable},
		{"auth error", &chatSender{err: &openai.APIError{HTTPStatusCode: 401}}, builder.ReasonResponseInvalid, cerrors.StatusUnavailable},
		{"truncated", &chatSender{reply: `{"message":"x"}`, finish: openai.FinishReasonLength}, builder.ReasonResponseInvalid, cerrors.StatusUnavailable},
		{"unparseable", &chatSender{reply: "not json at all"}, builder.ReasonResponseInvalid, cerrors.StatusUnavailable},
		{"llm deadline", &chatSender{err: context.DeadlineExceeded}, builder.ReasonTimeout, cerrors.StatusUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, cache := newTestHandler(t, tt.sender, testCfg(), 200, 3, true, true)
			cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil) // counted

			_, err := h.Chat(context.Background(), customerID, chatReq())
			st, reason := reasonOf(t, err)
			if reason != tt.reason || st != tt.status {
				t.Errorf("got %v/%s, want %v/%s", st, reason, tt.status, tt.reason)
			}
		})
	}
}

// A deadline that belongs to the CALLER (api-manager's RPC timeout) is
// reported by RunTurn as an LLM error with Code "timeout", not as ErrTimeout.
// The handler must give it the same reason as the LLM deadline: a client
// should not see two different failures for one cause.
func Test_Chat_callerDeadlineMapsToTimeout(t *testing.T) {
	s := &chatSender{block: make(chan struct{})}
	cfg := testCfg()
	cfg.LLMTimeout = 5 * time.Second
	h, cache := newTestHandler(t, s, cfg, 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := h.Chat(ctx, customerID, chatReq())
	_, reason := reasonOf(t, err)
	if reason != builder.ReasonTimeout {
		t.Errorf("a caller deadline must map to %s, got %s", builder.ReasonTimeout, reason)
	}
}

// A caller that gave up (cancelled) is not a provider failure and not a
// timeout the customer should be told about.
func Test_Chat_callerCancelIsNotATimeout(t *testing.T) {
	s := &chatSender{block: make(chan struct{})}
	h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(30 * time.Millisecond); cancel() }()
	_, err := h.Chat(ctx, customerID, chatReq())
	_, reason := reasonOf(t, err)
	if reason == builder.ReasonTimeout {
		t.Errorf("a cancelled caller must not be reported as %s", builder.ReasonTimeout)
	}
}

// The semaphore is non-blocking: when it is full the call fails at once with
// BUSY, is not counted against the daily limit, and the slot is released when
// the running call ends.
func Test_Chat_semaphore(t *testing.T) {
	s := &chatSender{block: make(chan struct{}), entered: make(chan struct{}, 8), reply: goodReply}
	h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil).Times(3)

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.Chat(context.Background(), customerID, chatReq())
		}()
	}
	for i := 0; i < 3; i++ {
		select {
		case <-s.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("the three calls did not start")
		}
	}

	// The fourth is refused at once, with no counter call (the strict mock has
	// only 3 expected increments).
	start := time.Now()
	_, err := h.Chat(context.Background(), customerID, chatReq())
	st, reason := reasonOf(t, err)
	if reason != builder.ReasonBusy || st != cerrors.StatusResourceExhausted {
		t.Errorf("got %v/%s", st, reason)
	}
	if time.Since(start) > time.Second {
		t.Error("a busy refusal must be immediate")
	}

	close(s.block)
	wg.Wait()

	// The slots are free again.
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)
	s.block = nil
	if _, err := h.Chat(context.Background(), customerID, chatReq()); err != nil {
		t.Errorf("the slot must be released after the running calls end: %v", err)
	}
}

// The slot is released on every exit path, including an over-limit refusal and
// a panic, or a few failures would lock the Builder out of a process for good.
func Test_Chat_releasesTheSlotOnEveryPath(t *testing.T) {
	t.Run("daily limit refusal", func(t *testing.T) {
		s := &chatSender{reply: goodReply}
		h, cache := newTestHandler(t, s, testCfg(), 200, 1, true, true)
		cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(999), nil).Times(3)
		for i := 0; i < 3; i++ {
			_, err := h.Chat(context.Background(), customerID, chatReq())
			if _, reason := reasonOf(t, err); reason != builder.ReasonDailyLimit {
				t.Fatalf("call %d: expected the daily limit refusal (a leaked slot would say BUSY), got %s", i, reason)
			}
		}
	})
	t.Run("redis error", func(t *testing.T) {
		s := &chatSender{reply: goodReply}
		h, cache := newTestHandler(t, s, testCfg(), 200, 1, true, true)
		cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(0), errors.New("x")).Times(3)
		for i := 0; i < 3; i++ {
			_, err := h.Chat(context.Background(), customerID, chatReq())
			if _, reason := reasonOf(t, err); reason != builder.ReasonUnavailable {
				t.Fatalf("call %d: got %s", i, reason)
			}
		}
	})
	t.Run("llm error", func(t *testing.T) {
		s := &chatSender{err: errors.New("boom")}
		h, cache := newTestHandler(t, s, testCfg(), 200, 1, true, true)
		cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil).Times(3)
		for i := 0; i < 3; i++ {
			_, err := h.Chat(context.Background(), customerID, chatReq())
			if _, reason := reasonOf(t, err); reason == builder.ReasonBusy {
				t.Fatalf("call %d: the slot leaked after an LLM error", i)
			}
		}
	})
	t.Run("panic in the engine", func(t *testing.T) {
		s := &panicSender{}
		h, cache := newTestHandler(t, s, testCfg(), 200, 1, true, true)
		cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil).Times(2)
		func() {
			defer func() { _ = recover() }()
			_, _ = h.Chat(context.Background(), customerID, chatReq())
		}()
		s.noPanic = true
		if _, err := h.Chat(context.Background(), customerID, chatReq()); err != nil {
			if _, reason := reasonOf(t, err); reason == builder.ReasonBusy {
				t.Fatal("the slot leaked after a panic")
			}
		}
	})
}

type panicSender struct{ noPanic bool }

func (p *panicSender) SendOnce(context.Context, *openai.ChatCompletionRequest) (*openai.ChatCompletionResponse, error) {
	if !p.noPanic {
		panic("engine blew up")
	}
	return &openai.ChatCompletionResponse{Choices: []openai.ChatCompletionChoice{{Message: openai.ChatCompletionMessage{Content: goodReply}, FinishReason: openai.FinishReasonStop}}}, nil
}

func Test_Status(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		keyConfigured bool
		available     bool
	}{
		{"enabled with a key", true, true, true},
		{"enabled without a key", true, false, false},
		{"disabled", false, true, false},
		{"disabled without a key", false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t, &chatSender{}, testCfg(), 200, 3, tt.enabled, tt.keyConfigured)
			st := h.Status()
			if st.Available != tt.available {
				t.Errorf("available: got %v, want %v", st.Available, tt.available)
			}
			if st.MaxMessages != builder.MaxMessages || st.MaxMessageChars != builder.MaxMessageRunes {
				t.Errorf("limits must be reported: %+v", st)
			}
		})
	}
}

// No log line, error text or label may carry the customer's input, including
// when the engine's own error echoes the prompt back.
func Test_Chat_neverLogsOrReturnsTheUsersInput(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.TraceLevel)
	defer logrus.SetOutput(logrus.StandardLogger().Out)

	req := &builder.ChatRequest{
		Messages:     []builder.Message{{Role: builder.RoleUser, Content: secret}},
		CurrentDraft: &builder.Draft{Name: secret, InitPrompt: secret},
	}

	cases := []struct {
		name   string
		sender *chatSender
	}{
		{"engine error echoes the prompt", &chatSender{err: errors.New("400 bad request: " + secret)}},
		{"unparseable reply echoes the prompt", &chatSender{reply: secret}},
		{"success", &chatSender{reply: `{"message":"ok"}`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buf.Reset()
			h, cache := newTestHandler(t, c.sender, testCfg(), 200, 3, true, true)
			cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)
			_, err := h.Chat(context.Background(), customerID, req)
			if err != nil && strings.Contains(err.Error(), secret) {
				t.Errorf("the returned error carries the input: %v", err)
			}
			if strings.Contains(buf.String(), secret) {
				t.Errorf("a log line carries the input:\n%s", buf.String())
			}
		})
	}
}

func Test_NewBuilderHandler_clampsAnUnusableConcurrency(t *testing.T) {
	// Config validation rejects 0 at startup; the constructor must still never
	// build a semaphore that blocks every call forever.
	s := &chatSender{reply: goodReply}
	h, cache := newTestHandler(t, s, testCfg(), 200, 0, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)
	if _, err := h.Chat(context.Background(), customerID, chatReq()); err != nil {
		t.Fatalf("a zero concurrency must be raised to at least one, not reject every call: %v", err)
	}
}

// Tokens are recorded from the turn's Usage even when the answer is unusable:
// the platform paid for them.
func Test_Chat_recordsTokensEvenOnAParseFailure(t *testing.T) {
	before := tokenCount("prompt")
	s := &chatSender{reply: "garbage"}
	h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	_, _ = h.Chat(context.Background(), customerID, chatReq())
	if tokenCount("prompt")-before != 100 {
		t.Errorf("the prompt tokens of an unusable answer must still be recorded")
	}
}

// tokenCount reads the token counter for tests.
func tokenCount(kind string) float64 {
	return testutil.ToFloat64(promBuilderTokensTotal.WithLabelValues(kind))
}

// The daily window is part of the limit: 24 hours, not a calendar day. The
// handler passes it to the counter, so a change here changes how much a
// customer may use, and no other test would notice.
func Test_Chat_countsWithA24HourWindow(t *testing.T) {
	s := &chatSender{reply: goodReply}
	h, cache := newTestHandler(t, s, testCfg(), 200, 3, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, 24*time.Hour).Return(int64(1), nil)

	if _, err := h.Chat(context.Background(), customerID, chatReq()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// An invalid request is refused before anything is counted or any slot taken.
func Test_Chat_validationRunsBeforeTheSemaphoreAndCounter(t *testing.T) {
	s := &chatSender{reply: goodReply}
	h, _ := newTestHandler(t, s, testCfg(), 200, 1, true, true) // strict mock: a counter call fails the test
	for i := 0; i < 3; i++ {
		_, err := h.Chat(context.Background(), customerID, &builder.ChatRequest{})
		if _, reason := reasonOf(t, err); reason != builder.ReasonInvalidArgument {
			t.Fatalf("call %d: got %s", i, reason)
		}
	}
	if s.callCount() != 0 {
		t.Error("no LLM call may happen")
	}
}

// resultCount reads ai_manager_builder_chat_total for one result label.
func resultCount(result string) float64 {
	return testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(result))
}

var allResultLabels = []string{
	resultOK, resultDailyLimit, resultBusy, resultDisabled, resultUnavailable,
	resultInvalidResponse, resultLLMError, resultInvalidArgument, resultInternal,
}

// Every outcome raises exactly its own label by one and no other. The labels
// are what operators alert on and what operations.md documents, and the plan
// requires llm_error to stay apart from the unavailable of a Redis failure.
func Test_Chat_resultLabels(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		sender *chatSender
		req    *builder.ChatRequest
		setup  func(c *cachehandler.MockCacheHandler)
		opts   [2]bool // enabled, keyConfigured
		limit  int
	}{
		{"ok", resultOK, &chatSender{reply: goodReply}, chatReq(), counted(1), [2]bool{true, true}, 200},
		{"invalid request", resultInvalidArgument, &chatSender{reply: goodReply}, &builder.ChatRequest{}, nil, [2]bool{true, true}, 200},
		{"disabled", resultDisabled, &chatSender{reply: goodReply}, chatReq(), nil, [2]bool{false, true}, 200},
		{"no key", resultUnavailable, &chatSender{reply: goodReply}, chatReq(), nil, [2]bool{true, false}, 200},
		{"counter down", resultUnavailable, &chatSender{reply: goodReply}, chatReq(), func(c *cachehandler.MockCacheHandler) {
			c.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(0), errors.New("x"))
		}, [2]bool{true, true}, 200},
		{"daily limit", resultDailyLimit, &chatSender{reply: goodReply}, chatReq(), counted(201), [2]bool{true, true}, 200},
		{"provider error", resultLLMError, &chatSender{err: &openai.APIError{HTTPStatusCode: 500}}, chatReq(), counted(1), [2]bool{true, true}, 200},
		{"llm deadline", resultLLMError, &chatSender{err: context.DeadlineExceeded}, chatReq(), counted(1), [2]bool{true, true}, 200},
		{"unparseable", resultInvalidResponse, &chatSender{reply: "not json"}, chatReq(), counted(1), [2]bool{true, true}, 200},
		{"truncated", resultInvalidResponse, &chatSender{reply: `{"message":"x"}`, finish: openai.FinishReasonLength}, chatReq(), counted(1), [2]bool{true, true}, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := map[string]float64{}
			for _, l := range allResultLabels {
				before[l] = resultCount(l)
			}

			h, cache := newTestHandler(t, tt.sender, testCfg(), tt.limit, 3, tt.opts[0], tt.opts[1])
			if tt.setup != nil {
				tt.setup(cache)
			}
			_, _ = h.Chat(context.Background(), customerID, tt.req)

			for _, l := range allResultLabels {
				delta := resultCount(l) - before[l]
				want := 0.0
				if l == tt.want {
					want = 1
				}
				if delta != want {
					t.Errorf("label %q moved by %v, want %v", l, delta, want)
				}
			}
		})
	}
}

func counted(n int64) func(c *cachehandler.MockCacheHandler) {
	return func(c *cachehandler.MockCacheHandler) {
		c.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(n, nil)
	}
}

// A busy refusal is counted under its own label, and not as anything else.
func Test_Chat_resultLabelBusy(t *testing.T) {
	s := &chatSender{block: make(chan struct{}), entered: make(chan struct{}, 2), reply: goodReply}
	h, cache := newTestHandler(t, s, testCfg(), 200, 1, true, true)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	done := make(chan struct{})
	go func() { _, _ = h.Chat(context.Background(), customerID, chatReq()); close(done) }()
	<-s.entered

	before := resultCount(resultBusy)
	_, _ = h.Chat(context.Background(), customerID, chatReq())
	if resultCount(resultBusy)-before != 1 {
		t.Error("a busy refusal must raise the busy label by one")
	}
	close(s.block)
	<-done
}

// A provider authentication failure means the platform's own key is wrong or
// expired and every customer is affected. It is logged at error level so an
// alert can be built on it; the other provider failures are expected noise.
func Test_Chat_providerAuthFailureIsLoggedAtErrorLevel(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantErr  bool
		wantCode string
	}{
		{"401", &openai.APIError{HTTPStatusCode: 401}, true, "auth"},
		{"403", &openai.APIError{HTTPStatusCode: 403}, true, "auth"},
		{"500", &openai.APIError{HTTPStatusCode: 500}, false, "provider_5xx"},
		{"429", &openai.APIError{HTTPStatusCode: 429}, false, "rate_limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logrusTestHook(t)
			h, cache := newTestHandler(t, &chatSender{err: tt.err}, testCfg(), 200, 3, true, true)
			cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

			_, _ = h.Chat(context.Background(), customerID, chatReq())

			sawError := false
			sawCode := false
			for _, e := range hook.AllEntries() {
				if e.Level <= logrus.ErrorLevel {
					sawError = true
				}
				if e.Data["llm_error"] == tt.wantCode {
					sawCode = true
				}
			}
			if sawError != tt.wantErr {
				t.Errorf("error-level entry: got %v, want %v", sawError, tt.wantErr)
			}
			if !sawCode {
				t.Errorf("the fixed classification code %q must be logged", tt.wantCode)
			}
		})
	}
}

func logrusTestHook(t *testing.T) *logrustest.Hook {
	t.Helper()
	hook := logrustest.NewGlobal()
	logrus.SetLevel(logrus.TraceLevel)
	t.Cleanup(func() { hook.Reset() })
	return hook
}
