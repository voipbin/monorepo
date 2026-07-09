package pipecatcallhandler

import (
	"context"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

func Test_runLLMIntermediateFlush(t *testing.T) {

	t.Run("tokens_batched_and_final_event_published_on_stop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
		}

		messageID := uuid.FromStringOrNil("aaa11111-1111-1111-1111-111111111111")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("bbb22222-2222-2222-2222-222222222222"),
				CustomerID: uuid.FromStringOrNil("ccc33333-3333-3333-3333-333333333333"),
			},
			Ctx:          ctx,
			LLMTokenChan: make(chan string, 64),
			LLMStopChan:  make(chan struct{}),
			LLMDoneChan:  make(chan struct{}),
		}

		// Track published events.
		var mu sync.Mutex
		var intermediates []message.Message
		var finalEvt *message.Message

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				intermediates = append(intermediates, evt)
			},
		).AnyTimes()

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				finalEvt = &evt
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID)

		// Send tokens.
		se.LLMTokenChan <- "Hello"
		se.LLMTokenChan <- " world"

		// Wait for at least one tick to fire (200ms + margin).
		time.Sleep(300 * time.Millisecond)

		// Signal stop and wait for completion.
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Should have at least one intermediate event.
		if len(intermediates) == 0 {
			t.Fatal("expected at least one intermediate event")
		}

		// Verify intermediate events have correct message ID and sequence.
		for i, evt := range intermediates {
			if evt.ID != messageID {
				t.Errorf("intermediate[%d]: expected message_id %s, got %s", i, messageID, evt.ID)
			}
			if evt.Sequence != i+1 {
				t.Errorf("intermediate[%d]: expected sequence %d, got %d", i, i+1, evt.Sequence)
			}
		}

		// Final event must exist with full text and correct ID.
		if finalEvt == nil {
			t.Fatal("expected final bot LLM event")
		}
		if finalEvt.ID != messageID {
			t.Errorf("final event: expected message_id %s, got %s", messageID, finalEvt.ID)
		}
		if finalEvt.Text != "Hello world" {
			t.Errorf("final event: expected text 'Hello world', got '%s'", finalEvt.Text)
		}
	})

	t.Run("stop_without_tick_flushes_all_tokens", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
		}

		messageID := uuid.FromStringOrNil("ddd44444-4444-4444-4444-444444444444")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("eee55555-5555-5555-5555-555555555555"),
				CustomerID: uuid.FromStringOrNil("fff66666-6666-6666-6666-666666666666"),
			},
			Ctx:          ctx,
			LLMTokenChan: make(chan string, 64),
			LLMStopChan:  make(chan struct{}),
			LLMDoneChan:  make(chan struct{}),
		}

		var mu sync.Mutex
		var intermediateCount int
		var finalText string

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				intermediateCount++
			},
		).AnyTimes()

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				finalText = evt.Text
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID)

		// Send tokens and immediately stop — no tick fires.
		se.LLMTokenChan <- "Quick"
		se.LLMTokenChan <- " response"
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// At least one intermediate event from the drain.
		if intermediateCount == 0 {
			t.Error("expected at least one intermediate event from drain")
		}

		if finalText != "Quick response" {
			t.Errorf("expected final text 'Quick response', got '%s'", finalText)
		}
	})

	t.Run("context_cancellation_publishes_partial_final_event", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
		}

		messageID := uuid.FromStringOrNil("aaa77777-7777-7777-7777-777777777777")
		ctx, cancel := context.WithCancel(context.Background())

		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("bbb88888-8888-8888-8888-888888888888"),
				CustomerID: uuid.FromStringOrNil("ccc99999-9999-9999-9999-999999999999"),
			},
			Ctx:          ctx,
			LLMTokenChan: make(chan string, 64),
			LLMStopChan:  make(chan struct{}),
			LLMDoneChan:  make(chan struct{}),
		}

		var mu sync.Mutex
		var finalText string

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).AnyTimes()

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				finalText = evt.Text
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID)

		// Send some tokens.
		se.LLMTokenChan <- "Partial"
		se.LLMTokenChan <- " text"

		// Give the goroutine a moment to receive at least one token.
		time.Sleep(50 * time.Millisecond)

		// Cancel context to simulate call ending mid-generation.
		cancel()
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		if finalText != "Partial text" {
			t.Errorf("expected partial final text 'Partial text', got '%s'", finalText)
		}
	})

	t.Run("context_cancellation_with_no_text_publishes_nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
		}

		messageID := uuid.FromStringOrNil("ddd00000-0000-0000-0000-000000000000")
		ctx, cancel := context.WithCancel(context.Background())

		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("eee11111-1111-1111-1111-111111111111"),
				CustomerID: uuid.FromStringOrNil("fff22222-2222-2222-2222-222222222222"),
			},
			Ctx:          ctx,
			LLMTokenChan: make(chan string, 64),
			LLMStopChan:  make(chan struct{}),
			LLMDoneChan:  make(chan struct{}),
		}

		// No PublishEvent expectations — nothing should be published.

		go h.runLLMIntermediateFlush(se, messageID)

		// Cancel immediately with no tokens sent.
		cancel()
		<-se.LLMDoneChan

		// If we get here without gomock errors, the test passes —
		// no unexpected PublishEvent calls were made.
	})

	t.Run("multiple_generations_per_session", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// --- Generation 1 ---
		messageID1 := uuid.FromStringOrNil("11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("22222222-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				CustomerID: uuid.FromStringOrNil("33333333-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			},
			Ctx:          ctx,
			LLMTokenChan: make(chan string, 64),
			LLMStopChan:  make(chan struct{}),
			LLMDoneChan:  make(chan struct{}),
		}
		se.LLMFlushing.Store(true)

		var mu sync.Mutex
		var gen1FinalText string
		var gen2FinalText string

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).AnyTimes()

		// Expect two final events (one per generation).
		gen1Final := mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				gen1FinalText = evt.Text
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID1)

		se.LLMTokenChan <- "First"
		se.LLMTokenChan <- " gen"
		close(se.LLMStopChan)
		<-se.LLMDoneChan
		se.LLMFlushing.Store(false)

		// --- Generation 2: reset channels, new UUID ---
		messageID2 := uuid.FromStringOrNil("44444444-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		se.LLMTokenChan = make(chan string, 64)
		se.LLMStopChan = make(chan struct{})
		se.LLMDoneChan = make(chan struct{})
		se.LLMFlushing.Store(true)

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				gen2FinalText = evt.Text
			},
		).Times(1).After(gen1Final)

		go h.runLLMIntermediateFlush(se, messageID2)

		se.LLMTokenChan <- "Second"
		se.LLMTokenChan <- " gen"
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		if gen1FinalText != "First gen" {
			t.Errorf("generation 1: expected 'First gen', got '%s'", gen1FinalText)
		}
		if gen2FinalText != "Second gen" {
			t.Errorf("generation 2: expected 'Second gen', got '%s'", gen2FinalText)
		}
	})

	t.Run("intermediate_event_metadata_correctness", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
		}

		messageID := uuid.FromStringOrNil("aaaa1111-2222-3333-4444-555566667777")
		sessionID := uuid.FromStringOrNil("bbbb1111-2222-3333-4444-555566667777")
		customerID := uuid.FromStringOrNil("cccc1111-2222-3333-4444-555566667777")
		referenceID := uuid.FromStringOrNil("dddd1111-2222-3333-4444-555566667777")
		activeflowID := uuid.FromStringOrNil("eeee1111-2222-3333-4444-555566667777")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         sessionID,
				CustomerID: customerID,
			},
			PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
			PipecatcallReferenceID:   referenceID,
			ActiveflowID:             activeflowID,
			Ctx:                      ctx,
			LLMTokenChan:             make(chan string, 64),
			LLMStopChan:              make(chan struct{}),
			LLMDoneChan:              make(chan struct{}),
		}

		var mu sync.Mutex
		var capturedIntermediate *message.Message

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				if capturedIntermediate == nil {
					capturedIntermediate = &evt
				}
			},
		).AnyTimes()

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).AnyTimes()

		go h.runLLMIntermediateFlush(se, messageID)

		se.LLMTokenChan <- "test token"

		// Wait for tick.
		time.Sleep(300 * time.Millisecond)

		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		if capturedIntermediate == nil {
			t.Fatal("expected at least one intermediate event")
		}

		if capturedIntermediate.ID != messageID {
			t.Errorf("expected ID %s, got %s", messageID, capturedIntermediate.ID)
		}
		if capturedIntermediate.CustomerID != customerID {
			t.Errorf("expected CustomerID %s, got %s", customerID, capturedIntermediate.CustomerID)
		}
		if capturedIntermediate.PipecatcallID != sessionID {
			t.Errorf("expected PipecatcallID %s, got %s", sessionID, capturedIntermediate.PipecatcallID)
		}
		if capturedIntermediate.PipecatcallReferenceType != pipecatcall.ReferenceTypeAICall {
			t.Errorf("expected ReferenceType %s, got %s", pipecatcall.ReferenceTypeAICall, capturedIntermediate.PipecatcallReferenceType)
		}
		if capturedIntermediate.PipecatcallReferenceID != referenceID {
			t.Errorf("expected ReferenceID %s, got %s", referenceID, capturedIntermediate.PipecatcallReferenceID)
		}
		if capturedIntermediate.ActiveflowID != activeflowID {
			t.Errorf("expected ActiveflowID %s, got %s", activeflowID, capturedIntermediate.ActiveflowID)
		}
		if capturedIntermediate.Text != "test token" {
			t.Errorf("expected Text 'test token', got '%s'", capturedIntermediate.Text)
		}
		if capturedIntermediate.Sequence != 1 {
			t.Errorf("expected Sequence 1, got %d", capturedIntermediate.Sequence)
		}
	})

	t.Run("non_blocking_send_when_channel_full", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		mockUtil := utilhandler.NewMockUtilHandler(mc)

		h := &pipecatcallHandler{
			notifyHandler: mockNotify,
			utilHandler:   mockUtil,
		}

		messageID := uuid.FromStringOrNil("ff001122-3344-5566-7788-99aabbccddee")
		mockUtil.EXPECT().UUIDCreate().Return(messageID)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := &pipecatcall.Session{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("aa112233-4455-6677-8899-aabbccddeeff"),
				CustomerID: uuid.FromStringOrNil("bb112233-4455-6677-8899-aabbccddeeff"),
			},
			Ctx: ctx,
		}

		// Simulate the read loop: spawn flush goroutine on first token.
		se.LLMMessageID = h.utilHandler.UUIDCreate()
		se.LLMTokenChan = make(chan string, 64)
		se.LLMStopChan = make(chan struct{})
		se.LLMDoneChan = make(chan struct{})
		se.LLMFlushing.Store(true)

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		go h.runLLMIntermediateFlush(se, se.LLMMessageID)

		// Fill the channel completely (cap 64).
		for i := 0; i < 64; i++ {
			se.LLMTokenChan <- "t"
		}

		// This 65th send must NOT block — the non-blocking select/default should drop it.
		done := make(chan struct{})
		go func() {
			select {
			case se.LLMTokenChan <- "overflow":
				// Channel had space (flush goroutine drained some) — also fine.
			default:
				// Dropped — expected behavior.
			}
			close(done)
		}()

		select {
		case <-done:
			// Success — did not block.
		case <-time.After(1 * time.Second):
			t.Fatal("non-blocking send blocked for >1s — would stall WebSocket read loop")
		}

		close(se.LLMStopChan)
		<-se.LLMDoneChan
	})
}

// TestBotLLMStopped_doubleStop_noPanic verifies that the BotLLMStopped handler
// is idempotent: a second BotLLMStopped frame for the same generation must not
// panic on a double-close of LLMStopChan, and the StopReason set by the first
// call (StopReasonNormal) must not be overwritten.
//
// The defense relies on Session.LLMFlushOnce wrapping close(LLMStopChan) and
// CompareAndSwap on LLMStopReason — both are tested together here because
// a regression in either would cause a real-world race with the watchdog and
// the flush goroutine that lands in subsequent tasks.
func TestBotLLMStopped_doubleStop_noPanic(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaa1111-bbbb-cccc-dddd-eeeeeeeeeeee"),
			CustomerID: uuid.FromStringOrNil("bbbb2222-cccc-dddd-eeee-ffffffffffff"),
		},
		Ctx:          ctx,
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}
	// Arm the flush goroutine: channels initialized + flushing flag set + goroutine running.
	se.LLMFlushing.Store(true)

	// Allow any number of intermediate / final events; they are not the focus
	// of this test — we only assert no panic and StopReason correctness.
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	go h.runLLMIntermediateFlush(se, uuid.FromStringOrNil("cccc3333-dddd-eeee-ffff-000000000000"))

	// Build a valid bot-llm-stopped RTVI frame payload that the handler can unmarshal.
	stoppedFrame := []byte(`{"label":"rtvi-ai","type":"bot-llm-stopped"}`)

	// First BotLLMStopped: should set StopReasonNormal, close LLMStopChan, and
	// block on <-LLMDoneChan until the flush goroutine completes.
	if err := h.receiveMessageFrameTypeMessage(se, stoppedFrame); err != nil {
		t.Fatalf("first BotLLMStopped returned unexpected error: %v", err)
	}

	// At this point the first call has unblocked from <-LLMDoneChan, so the
	// flush goroutine has exited. LLMFlushing is still true (Task 1.7 owns the
	// reset via defer); the second call's behavior is what this test pins down.
	//
	// Without sync.Once, the second close(LLMStopChan) below would panic.
	// We capture any panic so the test reports a clean assertion failure.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second BotLLMStopped panicked: %v", r)
		}
	}()

	if err := h.receiveMessageFrameTypeMessage(se, stoppedFrame); err != nil {
		t.Fatalf("second BotLLMStopped returned unexpected error: %v", err)
	}

	// First call set StopReasonNormal via CAS; second call's CAS must be a no-op.
	if got := StopReason(se.LLMStopReason.Load()); got != StopReasonNormal {
		t.Fatalf("expected StopReasonNormal (%d), got %d", StopReasonNormal, got)
	}
}

// TestRunLLMIntermediateFlush_publishesAfterCtxCancel verifies that the final
// and intermediate message_bot_llm publishes use a context that is independent
// of se.Ctx. When terminate() cancels the session context (Task 2.x), in-flight
// publishes that hand se.Ctx to PublishEvent would be dropped by the
// notifyHandler's RPC layer, losing the partial bot response in conversation
// history.
//
// This test cancels se.Ctx and then closes LLMStopChan. Whichever select branch
// wins (LLMStopChan or Ctx.Done), the ctx passed to PublishEvent for both the
// intermediate and final events must NOT be cancelled.
func TestRunLLMIntermediateFlush_publishesAfterCtxCancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
	}

	messageID := uuid.FromStringOrNil("aaaa9999-1111-2222-3333-444444444444")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("bbbb9999-1111-2222-3333-444444444444"),
			CustomerID: uuid.FromStringOrNil("cccc9999-1111-2222-3333-444444444444"),
		},
		Ctx:          ctx,
		Cancel:       cancel,
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}
	se.LLMFlushing.Store(true)

	var (
		mu                sync.Mutex
		finalCtx          context.Context
		finalText         string
		finalCalled       bool
		intermediateCtxs  []context.Context
		intermediateCount int
	)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).DoAndReturn(
		func(c context.Context, _ any, _ message.Message) {
			mu.Lock()
			defer mu.Unlock()
			intermediateCtxs = append(intermediateCtxs, c)
			intermediateCount++
		},
	).AnyTimes()

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
		func(c context.Context, _ any, evt message.Message) {
			mu.Lock()
			defer mu.Unlock()
			finalCalled = true
			finalText = evt.Text
			finalCtx = c
		},
	).Times(1)

	go h.runLLMIntermediateFlush(se, messageID)

	// Send one token, wait for the ticker to publish at least one intermediate
	// while se.Ctx is still alive (exercises the ticker branch deterministically).
	se.LLMTokenChan <- "hello "
	time.Sleep(300 * time.Millisecond)

	// Now simulate the terminate path: cancel se.Ctx then close LLMStopChan.
	// The final publish must use context.Background() regardless of which
	// select branch wins (LLMStopChan vs Ctx.Done) — both must be cancellation-
	// independent so partial replies still reach ai-manager on terminate.
	se.Cancel()
	close(se.LLMStopChan)

	<-se.LLMDoneChan

	mu.Lock()
	defer mu.Unlock()

	if !finalCalled {
		t.Fatal("expected final bot LLM event to be published")
	}
	if finalText != "hello " {
		t.Errorf("expected final text 'hello ', got '%s'", finalText)
	}

	bg := context.Background()
	if finalCtx != bg {
		t.Errorf("final publish must use context.Background(); got ctx with err=%v", finalCtx.Err())
	}

	if intermediateCount == 0 {
		t.Fatal("expected at least one intermediate event from the ticker branch")
	}
	for i, c := range intermediateCtxs {
		if c != bg {
			t.Errorf("intermediate[%d]: must use context.Background(); got ctx with err=%v", i, c.Err())
		}
	}
}

// TestRunLLMIntermediateFlush_resetsLLMFlushing_onAllExits verifies that
// runLLMIntermediateFlush always clears Session.LLMFlushing before returning,
// regardless of which exit path triggered the goroutine to return.
//
// The reset is owned by a defer at the top of the goroutine (Task 1.7) so it
// fires on every exit path: normal close (LLMStopChan), context cancel
// (Ctx.Done), or any future addition.
func TestRunLLMIntermediateFlush_resetsLLMFlushing_onAllExits(t *testing.T) {
	cases := []struct {
		name      string
		causeExit func(*pipecatcall.Session, context.CancelFunc)
	}{
		{
			name: "stopChan",
			causeExit: func(s *pipecatcall.Session, _ context.CancelFunc) {
				close(s.LLMStopChan)
			},
		},
		{
			name: "ctxDone",
			causeExit: func(s *pipecatcall.Session, cancel context.CancelFunc) {
				cancel()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			h := &pipecatcallHandler{
				notifyHandler: mockNotify,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-1111-2222-3333-444444444444"),
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-1111-2222-3333-444444444444"),
				},
				Ctx:          ctx,
				Cancel:       cancel,
				LLMTokenChan: make(chan string, 64),
				LLMStopChan:  make(chan struct{}),
				LLMDoneChan:  make(chan struct{}),
			}
			// Arm the flush state to mimic a live generation, then start the goroutine.
			se.LLMFlushing.Store(true)
			go h.runLLMIntermediateFlush(se, uuid.FromStringOrNil("cccccccc-1111-2222-3333-444444444444"))

			c.causeExit(se, cancel)

			// Wait for goroutine to exit.
			<-se.LLMDoneChan

			if se.LLMFlushing.Load() {
				t.Fatalf("LLMFlushing not reset on %s exit", c.name)
			}
		})
	}
}

// TestRunLLMIntermediateFlush_idleWatchdog_fires verifies that the idle
// watchdog fires after idleWatchdogTimeout of inactivity following at least
// one received token, sets StopReasonIdleWatchdog atomically, increments the
// idle-watchdog Prometheus counter, and the flush goroutine exits cleanly via
// the LLMStopChan branch (the watchdog closes LLMStopChan via LLMFlushOnce).
func TestRunLLMIntermediateFlush_idleWatchdog_fires(t *testing.T) {
	// Patch the timeout to a short value so the test runs quickly. Restore on
	// cleanup so other tests see the production default.
	origTimeout := idleWatchdogTimeout
	origTickRate := idleWatchdogTickRate
	idleWatchdogTimeout = 200 * time.Millisecond
	idleWatchdogTickRate = 50 * time.Millisecond
	t.Cleanup(func() {
		idleWatchdogTimeout = origTimeout
		idleWatchdogTickRate = origTickRate
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
	}

	messageID := uuid.FromStringOrNil("aaaa1234-1111-2222-3333-444444444444")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("bbbb1234-1111-2222-3333-444444444444"),
			CustomerID: uuid.FromStringOrNil("cccc1234-1111-2222-3333-444444444444"),
		},
		Ctx:          ctx,
		Cancel:       cancel,
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}
	se.LLMFlushing.Store(true)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	beforeFired := testutil.ToFloat64(metricsIdleWatchdogFired)
	beforeExitWatchdog := testutil.ToFloat64(metricsLLMFlushExit.WithLabelValues(reasonLabel(StopReasonIdleWatchdog)))

	go h.runLLMIntermediateFlush(se, messageID)

	// Send one token to arm the watchdog (it only fires after the first token).
	se.LLMTokenChan <- "first token "

	// Wait long enough for the watchdog to fire and the goroutine to exit.
	select {
	case <-se.LLMDoneChan:
	case <-time.After(idleWatchdogTimeout + 2*time.Second):
		t.Fatal("flush goroutine did not exit within expected window")
	}

	if got := StopReason(se.LLMStopReason.Load()); got != StopReasonIdleWatchdog {
		t.Fatalf("expected StopReasonIdleWatchdog (%d), got %d", StopReasonIdleWatchdog, got)
	}

	afterFired := testutil.ToFloat64(metricsIdleWatchdogFired)
	if delta := afterFired - beforeFired; delta < 1 {
		t.Errorf("expected metricsIdleWatchdogFired to be incremented by at least 1, got delta %v", delta)
	}

	afterExitWatchdog := testutil.ToFloat64(metricsLLMFlushExit.WithLabelValues(reasonLabel(StopReasonIdleWatchdog)))
	if delta := afterExitWatchdog - beforeExitWatchdog; delta < 1 {
		t.Errorf("expected metricsLLMFlushExit{reason=idle_watchdog} to be incremented by at least 1, got delta %v", delta)
	}
}

// TestRunLLMIntermediateFlush_idleWatchdog_doesNotFireBeforeFirstToken
// verifies the watchdog's first-token guard: with no token ever sent,
// the watchdog must NOT fire even if more than idleWatchdogTimeout elapses.
func TestRunLLMIntermediateFlush_idleWatchdog_doesNotFireBeforeFirstToken(t *testing.T) {
	origTimeout := idleWatchdogTimeout
	origTickRate := idleWatchdogTickRate
	idleWatchdogTimeout = 200 * time.Millisecond
	idleWatchdogTickRate = 50 * time.Millisecond
	t.Cleanup(func() {
		idleWatchdogTimeout = origTimeout
		idleWatchdogTickRate = origTickRate
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
	}

	messageID := uuid.FromStringOrNil("dddd1234-1111-2222-3333-444444444444")
	ctx, cancel := context.WithCancel(context.Background())

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eeee1234-1111-2222-3333-444444444444"),
			CustomerID: uuid.FromStringOrNil("ffff1234-1111-2222-3333-444444444444"),
		},
		Ctx:          ctx,
		Cancel:       cancel,
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}
	se.LLMFlushing.Store(true)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	go h.runLLMIntermediateFlush(se, messageID)

	// Wait > idleWatchdogTimeout. No token was sent, so watchdog must NOT fire.
	time.Sleep(idleWatchdogTimeout + 500*time.Millisecond)

	if got := StopReason(se.LLMStopReason.Load()); got != StopReasonUnset {
		t.Fatalf("expected StopReasonUnset (%d) before any token, got %d", StopReasonUnset, got)
	}

	// Cleanup: cancel context and wait for goroutine to exit.
	cancel()
	<-se.LLMDoneChan
}

// TestBotLLMText_inReplyToMessageID_snapshotsAtGenerationStart verifies the
// VOIP-1234 §4-1 cross-talk correlation: receiveMessageFrameTypeMessage
// snapshots Session.PendingInReplyToMessageID into
// Session.LLMInReplyToMessageID exactly once, at the start of each LLM
// generation (the first bot-llm-text token). If SendMessage overwrites
// PendingInReplyToMessageID with a newer value while a generation is still
// in flight, that generation's published events must still carry the
// snapshot taken at its own start -- not the newer pending value -- so an
// agent-facing client can correctly attribute a response to the message
// that actually triggered it.
func TestBotLLMText_inReplyToMessageID_snapshotsAtGenerationStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
	}

	gen1MessageID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	gen2MessageID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	inReplyTo1 := uuid.FromStringOrNil("00000001-0000-0000-0000-000000000000")
	inReplyTo2 := uuid.FromStringOrNil("00000002-0000-0000-0000-000000000000")

	gomock.InOrder(
		mockUtil.EXPECT().UUIDCreate().Return(gen1MessageID),
		mockUtil.EXPECT().UUIDCreate().Return(gen2MessageID),
	)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).AnyTimes()

	finalInReplyTo := make(chan uuid.UUID, 2)
	mockNotify.EXPECT().
		PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).
		DoAndReturn(func(_ any, _ any, evt message.Message) {
			finalInReplyTo <- evt.InReplyToMessageID
		}).
		Times(2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("ffffffff-9999-2222-3333-444444444444"),
			CustomerID: uuid.FromStringOrNil("dddddddd-9999-2222-3333-444444444444"),
		},
		Ctx: ctx,
	}

	textFrame := func(text string) []byte {
		return []byte(`{"label":"rtvi-ai","type":"bot-llm-text","data":{"text":"` + text + `"}}`)
	}
	stoppedFrame := []byte(`{"label":"rtvi-ai","type":"bot-llm-stopped"}`)

	// --- Generation 1: pending = inReplyTo1 ---
	se.PendingInReplyToMessageID = inReplyTo1
	if err := h.receiveMessageFrameTypeMessage(se, textFrame("First")); err != nil {
		t.Fatalf("gen1 BotLLMText returned unexpected error: %v", err)
	}
	if se.LLMInReplyToMessageID != inReplyTo1 {
		t.Fatalf("gen1: expected LLMInReplyToMessageID snapshot %s, got %s", inReplyTo1, se.LLMInReplyToMessageID)
	}
	gen1Done := se.LLMDoneChan

	// Simulate a second SendMessage arriving mid-generation-1, overwriting the
	// pending value before generation 1 has finished.
	se.PendingInReplyToMessageID = inReplyTo2

	if err := h.receiveMessageFrameTypeMessage(se, stoppedFrame); err != nil {
		t.Fatalf("gen1 BotLLMStopped returned unexpected error: %v", err)
	}
	select {
	case <-gen1Done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("gen1 flush goroutine did not exit within 500ms")
	}

	// --- Generation 2: pending is now inReplyTo2 ---
	if err := h.receiveMessageFrameTypeMessage(se, textFrame("Second")); err != nil {
		t.Fatalf("gen2 BotLLMText returned unexpected error: %v", err)
	}
	if se.LLMInReplyToMessageID != inReplyTo2 {
		t.Fatalf("gen2: expected LLMInReplyToMessageID snapshot %s, got %s", inReplyTo2, se.LLMInReplyToMessageID)
	}
	gen2Done := se.LLMDoneChan

	if err := h.receiveMessageFrameTypeMessage(se, stoppedFrame); err != nil {
		t.Fatalf("gen2 BotLLMStopped returned unexpected error: %v", err)
	}
	select {
	case <-gen2Done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("gen2 flush goroutine did not exit within 500ms")
	}

	close(finalInReplyTo)
	got := []uuid.UUID{}
	for id := range finalInReplyTo {
		got = append(got, id)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 final events (one per generation), got %d", len(got))
	}
	// Generation 1's final event must report inReplyTo1 (its own snapshot),
	// NOT inReplyTo2 (the value pending was overwritten to mid-flight).
	if got[0] != inReplyTo1 {
		t.Errorf("gen1 final event: expected InReplyToMessageID %s (snapshot at gen1 start), got %s", inReplyTo1, got[0])
	}
	if got[1] != inReplyTo2 {
		t.Errorf("gen2 final event: expected InReplyToMessageID %s (snapshot at gen2 start), got %s", inReplyTo2, got[1])
	}
}

// TestFlushAndFinalize_outcomes covers the four observable outcomes of
// flushAndFinalize, the synchronous helper terminate() will call to
// deterministically force the flush goroutine to publish its final event
// before SessionStop tears down the session.
//
// Outcome labels (closed set):
//   - noop_never_started: no flush goroutine ever ran (no BotLLMText was
//     received). LLMFlushing is false AND LLMMessageID is uuid.Nil.
//   - noop_already_done: flush goroutine ran and exited cleanly. LLMFlushing
//     is false but LLMMessageID is non-nil (set when the goroutine was armed).
//   - done: flush goroutine was running; we closed StopChan and it exited
//     within timeout.
//   - timeout: flush goroutine was running but did not return within
//     flushFinalizeTimeout.
func TestFlushAndFinalize_outcomes(t *testing.T) {
	// Patch the timeout to a short value so the timeout case runs quickly.
	// Restore on cleanup so other tests see the production default.
	origTimeout := flushFinalizeTimeout
	flushFinalizeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { flushFinalizeTimeout = origTimeout })

	// armFlush starts a real flush goroutine on `se` so that flushAndFinalize
	// must close LLMStopChan and wait for the goroutine to exit via LLMDoneChan.
	armFlush := func(t *testing.T, h *pipecatcallHandler, se *pipecatcall.Session) {
		t.Helper()
		se.LLMMessageID = uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
		se.LLMTokenChan = make(chan string, 64)
		se.LLMStopChan = make(chan struct{})
		se.LLMDoneChan = make(chan struct{})
		se.LLMFlushing.Store(true)
		go h.runLLMIntermediateFlush(se, se.LLMMessageID)
	}

	// armAndExitFlush arms a real flush goroutine, then waits for it to drain
	// and exit cleanly so that by the time flushAndFinalize is called,
	// LLMFlushing has been reset to false but LLMMessageID is still set.
	armAndExitFlush := func(t *testing.T, h *pipecatcallHandler, se *pipecatcall.Session) {
		t.Helper()
		armFlush(t, h, se)
		close(se.LLMStopChan)
		<-se.LLMDoneChan
		// LLMFlushing is reset to false by the goroutine's defer; LLMMessageID stays set.
	}

	// armFlushBlocking simulates a flush goroutine that does NOT drain — we
	// initialize the channels and set the flushing flag, but never start a
	// goroutine. flushAndFinalize will close LLMStopChan via LLMFlushOnce, then
	// the timer must fire because LLMDoneChan is never closed.
	armFlushBlocking := func(t *testing.T, _ *pipecatcallHandler, se *pipecatcall.Session) {
		t.Helper()
		se.LLMMessageID = uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
		se.LLMTokenChan = make(chan string, 64)
		se.LLMStopChan = make(chan struct{})
		se.LLMDoneChan = make(chan struct{})
		se.LLMFlushing.Store(true)
	}

	cases := []struct {
		name    string
		setup   func(*testing.T, *pipecatcallHandler, *pipecatcall.Session)
		outcome string
	}{
		{
			name:    "never_started",
			setup:   func(_ *testing.T, _ *pipecatcallHandler, _ *pipecatcall.Session) { /* no flush armed */ },
			outcome: "noop_never_started",
		},
		{
			name:    "already_done",
			setup:   armAndExitFlush,
			outcome: "noop_already_done",
		},
		{
			name:    "done",
			setup:   armFlush,
			outcome: "done",
		},
		{
			name:    "timeout",
			setup:   armFlushBlocking,
			outcome: "timeout",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			// Any PublishEvent calls (final/intermediate) the flush goroutine
			// emits during the "done" case are not the focus of this test.
			mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			h := &pipecatcallHandler{
				notifyHandler: mockNotify,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaaaaaaa-1111-2222-3333-444444444444"),
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-1111-2222-3333-444444444444"),
				},
				Ctx:    ctx,
				Cancel: cancel,
			}

			c.setup(t, h, se)

			before := testutil.ToFloat64(metricsFlushFinalizeOutcome.WithLabelValues(c.outcome))

			h.flushAndFinalize(se)

			after := testutil.ToFloat64(metricsFlushFinalizeOutcome.WithLabelValues(c.outcome))
			if delta := after - before; delta != 1 {
				t.Fatalf("expected metricsFlushFinalizeOutcome{outcome=%q} to increment by 1, got delta %v", c.outcome, delta)
			}
		})
	}
}

// TestBotLLMText_armNewGeneration_resetsOncePrimitives verifies the
// arm-new-generation path in the BotLLMText handler resets LLMFlushOnce and
// LLMStopReason between generations. Without the reset, generation 2's
// LLMFlushOnce.Do(close) would be a no-op (Once already fired in gen 1), so
// LLMStopChan would never close, the flush goroutine would block forever, and
// the per-generation final event would never be published.
//
// This test drives the real handler (receiveMessageFrameTypeMessage with a
// BotLLMText frame, then a BotLLMStopped frame) for two generations and
// asserts that gen 2's goroutine exits within a deadline AND that a final
// event is published for both generations.
func TestBotLLMText_armNewGeneration_resetsOncePrimitives(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
	}

	gen1MessageID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	gen2MessageID := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	// Each BotLLMText that arms a new generation calls UUIDCreate() once.
	gomock.InOrder(
		mockUtil.EXPECT().UUIDCreate().Return(gen1MessageID),
		mockUtil.EXPECT().UUIDCreate().Return(gen2MessageID),
	)

	// Allow any number of intermediate publishes; we focus on final events.
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).AnyTimes()

	finalCount := make(chan uuid.UUID, 2)
	mockNotify.EXPECT().
		PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).
		DoAndReturn(func(_ any, _ any, evt message.Message) {
			finalCount <- evt.ID
		}).
		Times(2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("ffffffff-1111-2222-3333-444444444444"),
			CustomerID: uuid.FromStringOrNil("dddddddd-1111-2222-3333-444444444444"),
		},
		Ctx: ctx,
	}

	textFrame := func(text string) []byte {
		return []byte(`{"label":"rtvi-ai","type":"bot-llm-text","data":{"text":"` + text + `"}}`)
	}
	stoppedFrame := []byte(`{"label":"rtvi-ai","type":"bot-llm-stopped"}`)

	// --- Generation 1 ---
	if err := h.receiveMessageFrameTypeMessage(se, textFrame("First")); err != nil {
		t.Fatalf("gen1 BotLLMText returned unexpected error: %v", err)
	}
	if se.LLMMessageID != gen1MessageID {
		t.Fatalf("gen1: expected LLMMessageID %s, got %s", gen1MessageID, se.LLMMessageID)
	}
	gen1Done := se.LLMDoneChan
	// BotLLMStopped uses LLMFlushOnce.Do (the production path). With the bug,
	// gen 2 would see Once already fired and never close LLMStopChan.
	if err := h.receiveMessageFrameTypeMessage(se, stoppedFrame); err != nil {
		t.Fatalf("gen1 BotLLMStopped returned unexpected error: %v", err)
	}
	select {
	case <-gen1Done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("gen1 flush goroutine did not exit within 500ms")
	}

	// --- Generation 2 ---
	if err := h.receiveMessageFrameTypeMessage(se, textFrame("Second")); err != nil {
		t.Fatalf("gen2 BotLLMText returned unexpected error: %v", err)
	}
	if se.LLMMessageID != gen2MessageID {
		t.Fatalf("gen2: expected LLMMessageID %s, got %s", gen2MessageID, se.LLMMessageID)
	}
	gen2Done := se.LLMDoneChan
	if gen1Done == gen2Done {
		t.Fatal("gen2 LLMDoneChan must be a fresh channel, got the gen1 channel")
	}
	// Confirm StopReason was reset back to Unset for gen 2.
	if got := StopReason(se.LLMStopReason.Load()); got != StopReasonUnset {
		t.Fatalf("gen2: expected LLMStopReason reset to Unset, got %d", got)
	}

	if err := h.receiveMessageFrameTypeMessage(se, stoppedFrame); err != nil {
		t.Fatalf("gen2 BotLLMStopped returned unexpected error: %v", err)
	}
	select {
	case <-gen2Done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("gen2 flush goroutine did not exit within 500ms — LLMFlushOnce was not reset between generations")
	}

	// Both generations must have produced a final event.
	close(finalCount)
	got := []uuid.UUID{}
	for id := range finalCount {
		got = append(got, id)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 final events (one per generation), got %d", len(got))
	}
}

// TestBotLLMStopped_boundedWaitOnLLMDoneChan verifies that the BotLLMStopped
// handler does not wait indefinitely on LLMDoneChan. A stalled flush goroutine
// (e.g. RabbitMQ publish wedged) must not stall the WebSocket read loop —
// otherwise audio frames stop being processed and the WebSocket peer eventually
// times out.
//
// The test arms a "fake" flush by setting LLMFlushing=true with channels but
// does NOT start the runLLMIntermediateFlush goroutine. As a result, no one
// will ever close LLMDoneChan. The handler must give up after
// flushFinalizeTimeout and return.
func TestBotLLMStopped_boundedWaitOnLLMDoneChan(t *testing.T) {
	// Patch flushFinalizeTimeout to a short value so the test runs quickly.
	origTimeout := flushFinalizeTimeout
	flushFinalizeTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		flushFinalizeTimeout = origTimeout
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	h := &pipecatcallHandler{
		notifyHandler: mockNotify,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11112222-3333-4444-5555-666666666666"),
			CustomerID: uuid.FromStringOrNil("99998888-7777-6666-5555-444444444444"),
		},
		Ctx:          ctx,
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}
	// Arm the flush state but DO NOT start the goroutine — LLMDoneChan will
	// never be closed. The handler must time out and return.
	se.LLMFlushing.Store(true)

	stoppedFrame := []byte(`{"label":"rtvi-ai","type":"bot-llm-stopped"}`)

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- h.receiveMessageFrameTypeMessage(se, stoppedFrame)
	}()

	deadline := flushFinalizeTimeout + 100*time.Millisecond
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("BotLLMStopped returned unexpected error: %v", err)
		}
		if elapsed := time.Since(start); elapsed > deadline {
			t.Fatalf("BotLLMStopped returned but took %v (>%v) — bound not enforced", elapsed, deadline)
		}
	case <-time.After(deadline):
		t.Fatalf("BotLLMStopped did not return within %v — wait on LLMDoneChan was unbounded", deadline)
	}
}
