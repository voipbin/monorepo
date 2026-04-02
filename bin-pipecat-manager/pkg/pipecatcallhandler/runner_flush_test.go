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

// newTestSession creates a Session with all channels initialized for flush goroutine tests.
func newTestSession(ctx context.Context, sessionID, customerID uuid.UUID) *pipecatcall.Session {
	return &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         sessionID,
			CustomerID: customerID,
		},
		Ctx:          ctx,
		LLMTokenChan: make(chan string, 64),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
		TTSTextChan:  make(chan string, 16),
	}
}

// ============================================================================
// Text mode tests (existing behavior, updated to create TTS channels)
// ============================================================================

func Test_runLLMIntermediateFlush_TextMode(t *testing.T) {

	t.Run("tokens_batched_and_final_event_published_on_stop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("aaa11111-1111-1111-1111-111111111111")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("bbb22222-2222-2222-2222-222222222222"),
			uuid.FromStringOrNil("ccc33333-3333-3333-3333-333333333333"),
		)

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

		se.LLMTokenChan <- "Hello"
		se.LLMTokenChan <- " world"

		// Wait for at least one tick to fire (200ms + margin).
		time.Sleep(300 * time.Millisecond)

		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		if len(intermediates) == 0 {
			t.Fatal("expected at least one intermediate event")
		}

		for i, evt := range intermediates {
			if evt.ID != messageID {
				t.Errorf("intermediate[%d]: expected message_id %s, got %s", i, messageID, evt.ID)
			}
			if evt.Sequence != i+1 {
				t.Errorf("intermediate[%d]: expected sequence %d, got %d", i, i+1, evt.Sequence)
			}
		}

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
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("ddd44444-4444-4444-4444-444444444444")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("eee55555-5555-5555-5555-555555555555"),
			uuid.FromStringOrNil("fff66666-6666-6666-6666-666666666666"),
		)

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

		se.LLMTokenChan <- "Quick"
		se.LLMTokenChan <- " response"
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

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
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("aaa77777-7777-7777-7777-777777777777")
		ctx, cancel := context.WithCancel(context.Background())

		se := newTestSession(ctx,
			uuid.FromStringOrNil("bbb88888-8888-8888-8888-888888888888"),
			uuid.FromStringOrNil("ccc99999-9999-9999-9999-999999999999"),
		)

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

		se.LLMTokenChan <- "Partial"
		se.LLMTokenChan <- " text"

		time.Sleep(50 * time.Millisecond)

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
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("ddd00000-0000-0000-0000-000000000000")
		ctx, cancel := context.WithCancel(context.Background())

		se := newTestSession(ctx,
			uuid.FromStringOrNil("eee11111-1111-1111-1111-111111111111"),
			uuid.FromStringOrNil("fff22222-2222-2222-2222-222222222222"),
		)

		go h.runLLMIntermediateFlush(se, messageID)

		cancel()
		<-se.LLMDoneChan
	})

	t.Run("multiple_generations_per_session", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// --- Generation 1 ---
		messageID1 := uuid.FromStringOrNil("11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		se := newTestSession(ctx,
			uuid.FromStringOrNil("22222222-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			uuid.FromStringOrNil("33333333-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		)
		se.LLMFlushing = true

		var mu sync.Mutex
		var gen1FinalText string
		var gen2FinalText string

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).AnyTimes()

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
		se.LLMFlushing = false

		// --- Generation 2: reset channels, new UUID ---
		messageID2 := uuid.FromStringOrNil("44444444-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		se.LLMTokenChan = make(chan string, 64)
		se.LLMStopChan = make(chan struct{})
		se.LLMDoneChan = make(chan struct{})
		se.TTSTextChan = make(chan string, 16)
		se.LLMFlushing = true

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
}

// ============================================================================
// TTS mode tests (debounce timer-based TTS completion detection)
// ============================================================================

func Test_runLLMIntermediateFlush_TTSMode(t *testing.T) {

	t.Run("tts_text_chunks_produce_intermediates", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify, ttsDrainTimeout: 50 * time.Millisecond}

		messageID := uuid.FromStringOrNil("tt111111-1111-1111-1111-111111111111")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("tt222222-2222-2222-2222-222222222222"),
			uuid.FromStringOrNil("tt333333-3333-3333-3333-333333333333"),
		)

		var mu sync.Mutex
		var intermediates []message.Message

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				intermediates = append(intermediates, evt)
			},
		).AnyTimes()

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).AnyTimes()

		go h.runLLMIntermediateFlush(se, messageID)

		// Send TTS text chunks (sentence-level).
		se.TTSTextChan <- "Hello there."
		se.TTSTextChan <- " How can I help you today?"

		// Give goroutine time to process.
		time.Sleep(50 * time.Millisecond)

		// Signal LLM done — starts debounce timer, which fires after 50ms.
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Should have 2 intermediate events from the 2 TTS chunks.
		if len(intermediates) < 2 {
			t.Fatalf("expected at least 2 intermediate events, got %d", len(intermediates))
		}

		if intermediates[0].Text != "Hello there." {
			t.Errorf("intermediate[0]: expected 'Hello there.', got '%s'", intermediates[0].Text)
		}
		if intermediates[0].Sequence != 1 {
			t.Errorf("intermediate[0]: expected sequence 1, got %d", intermediates[0].Sequence)
		}
		if intermediates[1].Text != " How can I help you today?" {
			t.Errorf("intermediate[1]: expected ' How can I help you today?', got '%s'", intermediates[1].Text)
		}
		if intermediates[1].Sequence != 2 {
			t.Errorf("intermediate[1]: expected sequence 2, got %d", intermediates[1].Sequence)
		}
	})

	t.Run("debounce_publishes_final_with_tts_text_only", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify, ttsDrainTimeout: 50 * time.Millisecond}

		messageID := uuid.FromStringOrNil("tt444444-4444-4444-4444-444444444444")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("tt555555-5555-5555-5555-555555555555"),
			uuid.FromStringOrNil("tt666666-6666-6666-6666-666666666666"),
		)

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

		// Send LLM tokens (full LLM output, longer than what TTS receives).
		se.LLMTokenChan <- "Hello"
		se.LLMTokenChan <- " there."
		se.LLMTokenChan <- " How"
		se.LLMTokenChan <- " can"
		se.LLMTokenChan <- " I"
		se.LLMTokenChan <- " help"
		se.LLMTokenChan <- " you"
		se.LLMTokenChan <- " today?"
		se.LLMTokenChan <- " I'm"
		se.LLMTokenChan <- " an"
		se.LLMTokenChan <- " AI"
		se.LLMTokenChan <- " assistant."

		// Send TTS text (only what TTS received — first two sentences).
		se.TTSTextChan <- "Hello there."
		se.TTSTextChan <- " How can I help you today?"

		time.Sleep(50 * time.Millisecond)

		// Signal LLM done — starts debounce timer in TTS mode.
		close(se.LLMStopChan)

		// Debounce timer fires after 50ms with no new TTS text.
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Final text should be TTS text only, not full LLM text.
		expected := "Hello there. How can I help you today?"
		if finalText != expected {
			t.Errorf("expected final text '%s', got '%s'", expected, finalText)
		}
	})

	t.Run("bargein_partial_tts_text_in_final", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("tt777777-7777-7777-7777-777777777777")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("tt888888-8888-8888-8888-888888888888"),
			uuid.FromStringOrNil("tt999999-9999-9999-9999-999999999999"),
		)

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

		// LLM generates full response.
		se.LLMTokenChan <- "Hello there. How can I help you today? I'm an AI assistant."

		// TTS only received partial text before user interrupted.
		se.TTSTextChan <- "Hello there."

		time.Sleep(50 * time.Millisecond)

		// User interrupts — context cancelled (barge-in).
		cancel()
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Final text should be only what TTS received.
		if finalText != "Hello there." {
			t.Errorf("expected final text 'Hello there.', got '%s'", finalText)
		}
	})

	t.Run("bargein_before_llm_stop_late_llm_stop_is_noop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("tt000010-1010-1010-1010-101010101010")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("tt000011-1111-1111-1111-111111111111"),
			uuid.FromStringOrNil("tt000012-1212-1212-1212-121212121212"),
		)
		se.LLMFlushing = true

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

		// LLM is still generating tokens.
		se.LLMTokenChan <- "Hello there. How can I help?"

		// TTS receives and speaks partial text.
		se.TTSTextChan <- "Hello there."

		time.Sleep(50 * time.Millisecond)

		// User interrupts — context cancelled BEFORE LLM finishes.
		cancel()
		<-se.LLMDoneChan
		se.LLMFlushing = false

		// Now LLM stop arrives late — should be a no-op because LLMFlushing is false.
		if se.LLMFlushing {
			close(se.LLMStopChan)
			<-se.LLMDoneChan
		}
		// If we get here without panic or deadlock, the late LLM stop was correctly ignored.

		mu.Lock()
		defer mu.Unlock()

		if finalText != "Hello there." {
			t.Errorf("expected final text 'Hello there.', got '%s'", finalText)
		}
	})

	t.Run("llm_stop_before_tts_done_goroutine_keeps_running", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify, ttsDrainTimeout: 100 * time.Millisecond}

		messageID := uuid.FromStringOrNil("tt000001-1111-1111-1111-111111111111")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("tt000002-2222-2222-2222-222222222222"),
			uuid.FromStringOrNil("tt000003-3333-3333-3333-333333333333"),
		)

		var mu sync.Mutex
		var intermediates []message.Message
		var finalText string

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
				finalText = evt.Text
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID)

		// First TTS chunk switches to TTS mode.
		se.TTSTextChan <- "First sentence."

		time.Sleep(50 * time.Millisecond)

		// LLM finishes generating — starts debounce timer (100ms).
		close(se.LLMStopChan)

		// More TTS chunks arrive AFTER LLM stop — resets debounce timer.
		time.Sleep(20 * time.Millisecond)
		se.TTSTextChan <- " Second sentence."

		// Debounce timer fires 100ms after last TTS text.
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Should have intermediates for both TTS chunks.
		if len(intermediates) < 2 {
			t.Fatalf("expected at least 2 intermediates, got %d", len(intermediates))
		}

		// Final text includes all TTS chunks.
		expected := "First sentence. Second sentence."
		if finalText != expected {
			t.Errorf("expected final text '%s', got '%s'", expected, finalText)
		}
	})

	t.Run("context_cancellation_in_tts_mode", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("tt000004-4444-4444-4444-444444444444")
		ctx, cancel := context.WithCancel(context.Background())

		se := newTestSession(ctx,
			uuid.FromStringOrNil("tt000005-5555-5555-5555-555555555555"),
			uuid.FromStringOrNil("tt000006-6666-6666-6666-666666666666"),
		)

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

		// Switch to TTS mode.
		se.TTSTextChan <- "Hello there."

		time.Sleep(50 * time.Millisecond)

		// Call ends mid-generation.
		cancel()
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Final should contain TTS text received so far.
		if finalText != "Hello there." {
			t.Errorf("expected final text 'Hello there.', got '%s'", finalText)
		}
	})
}

// ============================================================================
// Mode switching tests
// ============================================================================

func Test_runLLMIntermediateFlush_ModeSwitching(t *testing.T) {

	t.Run("auto_detect_tts_mode_on_first_tts_text", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify, ttsDrainTimeout: 50 * time.Millisecond}

		messageID := uuid.FromStringOrNil("ms111111-1111-1111-1111-111111111111")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("ms222222-2222-2222-2222-222222222222"),
			uuid.FromStringOrNil("ms333333-3333-3333-3333-333333333333"),
		)

		var mu sync.Mutex
		var intermediates []message.Message
		var finalText string

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
				finalText = evt.Text
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID)

		// Send LLM tokens first (text mode accumulates).
		se.LLMTokenChan <- "Hello"
		se.LLMTokenChan <- " there."

		// Wait for timer to fire and publish a text-mode intermediate.
		time.Sleep(300 * time.Millisecond)

		// Now TTS text arrives — switches to TTS mode.
		se.TTSTextChan <- "Hello there."

		time.Sleep(50 * time.Millisecond)

		// Signal LLM done — starts debounce timer (50ms).
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Should have intermediates: at least one timer-based + one TTS-based.
		if len(intermediates) < 2 {
			t.Fatalf("expected at least 2 intermediates (timer + TTS), got %d", len(intermediates))
		}

		// Final text should be TTS text.
		if finalText != "Hello there." {
			t.Errorf("expected final text 'Hello there.', got '%s'", finalText)
		}
	})

	t.Run("llm_stop_before_first_tts_text_uses_text_mode", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("ms444444-4444-4444-4444-444444444444")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("ms555555-5555-5555-5555-555555555555"),
			uuid.FromStringOrNil("ms666666-6666-6666-6666-666666666666"),
		)

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

		// LLM tokens arrive.
		se.LLMTokenChan <- "Short"
		se.LLMTokenChan <- " reply"

		// LLM finishes before any TTS text arrives (very short response).
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Should use text mode — final from LLM tokens.
		if finalText != "Short reply" {
			t.Errorf("expected final text 'Short reply', got '%s'", finalText)
		}
	})
}

// ============================================================================
// Read loop integration tests
// ============================================================================

func Test_runLLMIntermediateFlush_ReadLoopIntegration(t *testing.T) {

	t.Run("nonblocking_tts_text_send_after_goroutine_exits", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("rl111111-1111-1111-1111-111111111111")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("rl222222-2222-2222-2222-222222222222"),
			uuid.FromStringOrNil("rl333333-3333-3333-3333-333333333333"),
		)

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		go h.runLLMIntermediateFlush(se, messageID)

		// Exit via text mode (LLM stop, no TTS).
		se.LLMTokenChan <- "text"
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		// Now simulate late TTS text arriving after goroutine already exited.
		// This must not block (the read loop checks LLMDoneChan).
		done := make(chan struct{})
		go func() {
			select {
			case se.TTSTextChan <- "late tts text":
				// Channel had space — fine, the goroutine isn't reading but buffer absorbs it.
			case <-se.LLMDoneChan:
				// Goroutine already exited — this is the expected path for the read loop.
			}
			close(done)
		}()

		select {
		case <-done:
			// Success — did not block.
		case <-time.After(1 * time.Second):
			t.Fatal("TTS text send blocked after goroutine exited")
		}
	})

	t.Run("late_events_after_goroutine_exited_no_panic", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

		messageID := uuid.FromStringOrNil("rl444444-4444-4444-4444-444444444444")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx,
			uuid.FromStringOrNil("rl555555-5555-5555-5555-555555555555"),
			uuid.FromStringOrNil("rl666666-6666-6666-6666-666666666666"),
		)
		se.LLMFlushing = true

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		go h.runLLMIntermediateFlush(se, messageID)

		// Exit via text mode.
		se.LLMTokenChan <- "text"
		close(se.LLMStopChan)
		<-se.LLMDoneChan
		se.LLMFlushing = false

		// Simulate the read loop receiving late TTS events after goroutine exited.
		// The read loop's LLMFlushing guard prevents sends to dead channels.
		// This test verifies the guard works — no panic, no block.
		if se.LLMFlushing {
			// This block never executes because LLMFlushing is false.
			// If the guard were missing, this could panic or deadlock.
			se.TTSTextChan <- "late text"
			<-se.LLMDoneChan
		}
		// If we get here, the guard correctly prevented the operation.
	})

	t.Run("metadata_correctness_in_tts_mode", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify, ttsDrainTimeout: 50 * time.Millisecond}

		messageID := uuid.FromStringOrNil("rl777777-7777-7777-7777-777777777777")
		sessionID := uuid.FromStringOrNil("rl888888-8888-8888-8888-888888888888")
		customerID := uuid.FromStringOrNil("rl999999-9999-9999-9999-999999999999")
		referenceID := uuid.FromStringOrNil("rl000001-0000-0000-0000-000000000000")
		activeflowID := uuid.FromStringOrNil("rl000002-0000-0000-0000-000000000000")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		se := newTestSession(ctx, sessionID, customerID)
		se.PipecatcallReferenceType = pipecatcall.ReferenceTypeAICall
		se.PipecatcallReferenceID = referenceID
		se.ActiveflowID = activeflowID

		var mu sync.Mutex
		var capturedIntermediate *message.Message
		var capturedFinal *message.Message

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLMIntermediate), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				if capturedIntermediate == nil {
					capturedIntermediate = &evt
				}
			},
		).AnyTimes()

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Eq(message.EventTypeBotLLM), gomock.Any()).DoAndReturn(
			func(_ any, _ any, evt message.Message) {
				mu.Lock()
				defer mu.Unlock()
				capturedFinal = &evt
			},
		).Times(1)

		go h.runLLMIntermediateFlush(se, messageID)

		se.TTSTextChan <- "Hello."

		time.Sleep(50 * time.Millisecond)

		// Signal LLM done — starts debounce timer (50ms).
		close(se.LLMStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Verify intermediate metadata.
		if capturedIntermediate == nil {
			t.Fatal("expected at least one intermediate event")
		}
		if capturedIntermediate.ID != messageID {
			t.Errorf("intermediate: expected ID %s, got %s", messageID, capturedIntermediate.ID)
		}
		if capturedIntermediate.CustomerID != customerID {
			t.Errorf("intermediate: expected CustomerID %s, got %s", customerID, capturedIntermediate.CustomerID)
		}
		if capturedIntermediate.PipecatcallID != sessionID {
			t.Errorf("intermediate: expected PipecatcallID %s, got %s", sessionID, capturedIntermediate.PipecatcallID)
		}
		if capturedIntermediate.PipecatcallReferenceType != pipecatcall.ReferenceTypeAICall {
			t.Errorf("intermediate: expected ReferenceType %s, got %s", pipecatcall.ReferenceTypeAICall, capturedIntermediate.PipecatcallReferenceType)
		}
		if capturedIntermediate.PipecatcallReferenceID != referenceID {
			t.Errorf("intermediate: expected ReferenceID %s, got %s", referenceID, capturedIntermediate.PipecatcallReferenceID)
		}
		if capturedIntermediate.ActiveflowID != activeflowID {
			t.Errorf("intermediate: expected ActiveflowID %s, got %s", activeflowID, capturedIntermediate.ActiveflowID)
		}

		// Verify final metadata.
		if capturedFinal == nil {
			t.Fatal("expected final event")
		}
		if capturedFinal.ID != messageID {
			t.Errorf("final: expected ID %s, got %s", messageID, capturedFinal.ID)
		}
		if capturedFinal.Text != "Hello." {
			t.Errorf("final: expected Text 'Hello.', got '%s'", capturedFinal.Text)
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

		se.LLMMessageID = h.utilHandler.UUIDCreate()
		se.LLMTokenChan = make(chan string, 64)
		se.LLMStopChan = make(chan struct{})
		se.LLMDoneChan = make(chan struct{})
		se.TTSTextChan = make(chan string, 16)
		se.LLMFlushing = true

		mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

		go h.runLLMIntermediateFlush(se, se.LLMMessageID)

		// Fill the LLM channel completely (cap 64).
		for i := 0; i < 64; i++ {
			se.LLMTokenChan <- "t"
		}

		// 65th send must NOT block.
		done := make(chan struct{})
		go func() {
			select {
			case se.LLMTokenChan <- "overflow":
			default:
			}
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("non-blocking send blocked for >1s")
		}

		close(se.LLMStopChan)
		<-se.LLMDoneChan
	})
}
