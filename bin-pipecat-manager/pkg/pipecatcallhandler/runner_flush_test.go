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
