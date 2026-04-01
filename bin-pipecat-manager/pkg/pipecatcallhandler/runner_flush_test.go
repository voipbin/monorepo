package pipecatcallhandler

import (
	"context"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
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
}
