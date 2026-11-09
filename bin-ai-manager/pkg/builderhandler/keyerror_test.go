package builderhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
)

const testKeyText = "sk-synthetic-not-a-real-key"

var errKeyRead = fmt.Errorf("%w: database is down", ErrKeyUnavailable)

func keyCheckFn(configured bool, err error) func(context.Context) (bool, error) {
	return func(context.Context) (bool, error) { return configured, err }
}

// newKeyTestHandlers builds both Builder handlers with the given key check and a strict mock cache.
func newKeyTestHandlers(t *testing.T, s Sender, check func(context.Context) (bool, error), maxConcurrent int) (BuilderHandler, FlowBuilderHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	cache := cachehandler.NewMockCacheHandler(gomock.NewController(t))
	assistant, flow := NewBuilderHandlers(s, cache, testCfg(), Options{KeyConfigured: check, DailyLimit: 200, MaxConcurrent: maxConcurrent})

	return assistant, flow, cache
}

// A key that can not be read stops the turn before anything else: before the validation, the
// semaphore and the counter, and no model call is made. The strict mock cache fails on any counter
// call. The request is invalid and the semaphore is taken on purpose: the key error still wins.
func Test_Chat_keyReadErrorComesFirst(t *testing.T) {
	logs := logtest.New(t)
	s := &chatSender{reply: goodReply}
	assistant, flow, _ := newKeyTestHandlers(t, s, keyCheckFn(false, errKeyRead), 1)

	// the semaphore (shared by both handlers) is taken
	assistant.(*builderHandler).sem <- struct{}{}

	before := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultKeyError))
	flowBefore := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultKeyError))

	badAssistant := &builder.ChatRequest{Messages: []builder.Message{{Role: builder.RoleAssistant, Content: "last is assistant"}}}
	if _, err := assistant.Chat(context.Background(), customerID, badAssistant); err == nil {
		t.Fatal("Wrong match. expect: an error")
	} else if st, reason := reasonOf(t, err); reason != builder.ReasonUnavailable || st != cerrors.StatusUnavailable {
		t.Errorf("Wrong match. expect: unavailable, got: %v %s", st, reason)
	}

	// both would be refused as invalid_argument if the key check came after them: a request that
	// fails the validation, and one with no usable action type
	badMessages := flowChatReq()
	badMessages.Messages = []flowbuilder.Message{{Role: flowbuilder.RoleAssistant, Content: "last is assistant"}}
	noType := flowChatReq()
	noType.SupportedActionTypes = []string{"no-such-type"}
	for _, bad := range []*flowbuilder.ChatRequest{badMessages, noType} {
		if _, err := flow.Chat(context.Background(), customerID, bad); err == nil {
			t.Fatal("Wrong match. expect: an error")
		} else if _, reason := reasonOf(t, err); reason != builder.ReasonUnavailable {
			t.Errorf("Wrong match. expect: unavailable, got: %s", reason)
		}
	}

	if s.callCount() != 0 {
		t.Error("Wrong match. no model call may happen")
	}
	if got := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultKeyError)) - before; got != 1 {
		t.Errorf("Wrong match. expect: one key_error, got: %v", got)
	}
	if got := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultKeyError)) - flowBefore; got != 2 {
		t.Errorf("Wrong match. expect: two flow key_error, got: %v", got)
	}
	if !strings.Contains(logs.Dump(), "Could not read the builder key.") {
		t.Errorf("Wrong match. expect: the warning, got: %s", logs.Dump())
	}
}

// A key that is not configured is a different answer from one that can not be read.
func Test_Chat_keyNotConfiguredIsNotAKeyError(t *testing.T) {
	assistant, flow, _ := newKeyTestHandlers(t, &chatSender{reply: goodReply}, keyCheckFn(false, nil), 1)

	before := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultKeyError))
	if _, err := assistant.Chat(context.Background(), customerID, chatReq()); err == nil {
		t.Fatal("Wrong match. expect: an error")
	}
	if _, err := flow.Chat(context.Background(), customerID, flowChatReq()); err == nil {
		t.Fatal("Wrong match. expect: an error")
	}
	if got := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultKeyError)) - before; got != 0 {
		t.Errorf("Wrong match. expect: no key_error, got: %v", got)
	}
}

// The model call reads the key too. A read that fails there is a key error and not a provider error
// (no llm_error, no auth). The turn was counted before the call and stays counted.
func Test_Chat_keyReadErrorInTheCall(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"plain", errKeyRead},
		{"with the deadline of the call", fmt.Errorf("%w: %w", ErrKeyUnavailable, context.DeadlineExceeded)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &chatSender{err: tt.err}
			assistant, flow, cache := newKeyTestHandlers(t, s, keyCheckFn(true, nil), 2)
			cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)
			cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

			before := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultKeyError))
			llmBefore := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultLLMError))
			flowBefore := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultKeyError))

			_, err := assistant.Chat(context.Background(), customerID, chatReq())
			if _, reason := reasonOf(t, err); reason != builder.ReasonUnavailable {
				t.Errorf("Wrong match. expect: unavailable, got: %s", reason)
			}
			_, err = flow.Chat(context.Background(), customerID, flowChatReq())
			if _, reason := reasonOf(t, err); reason != builder.ReasonUnavailable {
				t.Errorf("Wrong match. expect: unavailable, got: %s", reason)
			}

			if got := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultKeyError)) - before; got != 1 {
				t.Errorf("Wrong match. expect: one key_error, got: %v", got)
			}
			if got := testutil.ToFloat64(promBuilderChatTotal.WithLabelValues(resultLLMError)) - llmBefore; got != 0 {
				t.Errorf("Wrong match. expect: no llm_error, got: %v", got)
			}
			if got := testutil.ToFloat64(promFlowBuilderChatTotal.WithLabelValues(resultKeyError)) - flowBefore; got != 1 {
				t.Errorf("Wrong match. expect: one flow key_error, got: %v", got)
			}
		})
	}
}

// RunTurn and RunFlowTurn pass a key error on as it is: it is neither the LLM timeout nor an LLM
// error, even when the read ended with the deadline of the call, and the cause stays visible.
func Test_RunTurn_keyErrorWinsOverTheDeadline(t *testing.T) {
	cause := fmt.Errorf("%w: %w", ErrKeyUnavailable, context.DeadlineExceeded)
	s := &chatSender{err: cause}

	_, err := RunTurn(context.Background(), s, testCfg(), chatReq())
	if !errors.Is(err, ErrKeyUnavailable) || errors.Is(err, ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Wrong match. expect: the key error with its cause, got: %v", err)
	}

	_, err = RunFlowTurn(context.Background(), s, testCfg(), flowChatReq(), FlowAllowedTypes(flowChatReq().SupportedActionTypes))
	if !errors.Is(err, ErrKeyUnavailable) || errors.Is(err, ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Wrong match. expect: the key error with its cause, got: %v", err)
	}
}

// The deadline of the call that is not a key error stays the LLM timeout.
func Test_RunTurn_deadlineIsStillTheTimeout(t *testing.T) {
	s := &chatSender{err: context.DeadlineExceeded}
	if _, err := RunTurn(context.Background(), s, testCfg(), chatReq()); !errors.Is(err, ErrTimeout) {
		t.Errorf("Wrong match. expect: timeout, got: %v", err)
	}
}

// Status reports a key that can not be read as not available, and passes the context on.
func Test_Status_keyCheck(t *testing.T) {
	tests := []struct {
		name       string
		check      func(context.Context) (bool, error)
		expectAvbl bool
	}{
		{"configured", keyCheckFn(true, nil), true},
		{"not configured", keyCheckFn(false, nil), false},
		{"read error", keyCheckFn(false, errKeyRead), false},
		{"read error with a true value", keyCheckFn(true, errKeyRead), false},
		{"no check", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assistant, _, _ := newKeyTestHandlers(t, &chatSender{}, tt.check, 1)
			if got := assistant.Status(context.Background()); got.Available != tt.expectAvbl {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAvbl, got.Available)
			}
		})
	}
}

// Every key check runs under a deadline: Chat and Status are called with a context that has none.
// The check is the only time that is added to the call deadline, so its bound is what keeps the
// whole under the wait of api-manager.
func Test_keyCheck_runsUnderADeadline(t *testing.T) {
	var got time.Time
	var ok bool
	check := func(ctx context.Context) (bool, error) {
		got, ok = ctx.Deadline()
		return true, nil
	}

	assistant, flow, cache := newKeyTestHandlers(t, &chatSender{reply: goodReply}, check, 2)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)
	cache.EXPECT().BuilderFlowChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	run := map[string]func(){
		"Status":   func() { assistant.Status(context.Background()) },
		"Chat":     func() { _, _ = assistant.Chat(context.Background(), customerID, chatReq()) },
		"FlowChat": func() { _, _ = flow.Chat(context.Background(), customerID, flowChatReq()) },
	}

	for name, fn := range run {
		got, ok = time.Time{}, false

		done := make(chan struct{})
		start := time.Now()
		go func() {
			defer close(done)
			fn()
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. %s did not return", name)
		}

		if !ok {
			t.Errorf("Wrong match. %s: the key check ran without a deadline", name)
			continue
		}
		if left := got.Sub(start); left > keyCheckTimeout+500*time.Millisecond || left <= 0 {
			t.Errorf("Wrong match. %s: expect: a deadline within %v, got: %v", name, keyCheckTimeout, left)
		}
	}
}

// The check and the longest call deadline together stay under the wait of api-manager.
func Test_timeBudget(t *testing.T) {
	budget := keyCheckTimeout + time.Duration(config.BuilderLLMTimeoutMaxSeconds)*time.Second
	if limit := time.Duration(config.BuilderRPCTimeoutSeconds) * time.Second; budget >= limit {
		t.Errorf("Wrong match. expect: the check plus the call deadline below %v, got: %v", limit, budget)
	}
}

// Nothing the engine or the check returns puts a key into a log line or an error.
func Test_Chat_keyTextNeverLogged(t *testing.T) {
	logs := logtest.New(t)
	s := &chatSender{err: fmt.Errorf("%w: read failed", ErrKeyUnavailable)}
	assistant, _, cache := newKeyTestHandlers(t, s, keyCheckFn(true, nil), 1)
	cache.EXPECT().BuilderChatCountIncr(gomock.Any(), customerID, gomock.Any()).Return(int64(1), nil)

	_, err := assistant.Chat(context.Background(), customerID, chatReq())
	if err == nil || strings.Contains(err.Error(), testKeyText) || strings.Contains(logs.Dump(), testKeyText) {
		t.Errorf("Wrong match. the key text reached the error or the log: %v", err)
	}
}
