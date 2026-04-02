# Sync aimessage_intermediate with TTS — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Synchronize intermediate and final AI message events with TTS output for voice calls, while preserving existing 200ms timer behavior for text-based calls.

**Architecture:** Dual-mode flush goroutine that auto-detects voice vs text calls. Two new channels (`TTSTextChan`, `TTSStopChan`) carry TTS events from the read loop to the goroutine. Mode switches from text to TTS on first `bot-tts-text` event.

**Tech Stack:** Go, gomock, gorilla/websocket, RabbitMQ (via notifyHandler)

**Design doc:** `docs/plans/2026-04-02-sync-aimessage-intermediate-with-tts.md`

---

### Task 1: Add RTVIFrameTypeBotTTSText constant

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatframe/helper.go:18` (add constant between existing TTS constants)
- Modify: `bin-pipecat-manager/models/pipecatframe/helper_test.go:100` (add test case)

**Step 1: Add test case for the new constant**

In `helper_test.go`, add this test case inside `TestRTVIFrameTypeConstants` after the `bot_tts_started` case (line 99):

```go
		{
			name:     "bot_tts_text",
			constant: RTVIFrameTypeBotTTSText,
			expected: "bot-tts-text",
		},
```

**Step 2: Run test to verify it fails**

Run: `cd bin-pipecat-manager && go test -v -run TestRTVIFrameTypeConstants ./models/pipecatframe/...`
Expected: Compilation error — `RTVIFrameTypeBotTTSText` undefined.

**Step 3: Add the constant**

In `helper.go`, add between `RTVIFrameTypeBotTTSStarted` and `RTVIFrameTypeBotTTSStopped` (line 19):

```go
	RTVIFrameTypeBotTTSText    = "bot-tts-text"
```

The const block should look like:
```go
	RTVIFrameTypeBotTTSStarted = "bot-tts-started"
	RTVIFrameTypeBotTTSText    = "bot-tts-text"
	RTVIFrameTypeBotTTSStopped = "bot-tts-stopped"
```

**Step 4: Run test to verify it passes**

Run: `cd bin-pipecat-manager && go test -v -run TestRTVIFrameTypeConstants ./models/pipecatframe/...`
Expected: PASS

**Step 5: Commit**

```bash
git add bin-pipecat-manager/models/pipecatframe/helper.go bin-pipecat-manager/models/pipecatframe/helper_test.go
git commit -m "NOJIRA-Sync-aimessage-intermediate-with-tts

- bin-pipecat-manager: Add RTVIFrameTypeBotTTSText constant for bot-tts-text events"
```

---

### Task 2: Add TTS channels to Session struct

**Files:**
- Modify: `bin-pipecat-manager/models/pipecatcall/session.go:43` (add fields after LLMMessageID)

**Step 1: Add TTSTextChan and TTSStopChan fields**

After line 43 (`LLMMessageID uuid.UUID`), add:

```go
	// TTS sync channels for voice call intermediate event synchronization.
	// TTSTextChan carries sentence-level TTS text chunks from the read loop to the flush goroutine.
	// TTSStopChan is closed when bot-tts-stopped is received, signaling TTS completion.
	TTSTextChan chan string   `json:"-"` // TTS text chunks from bot-tts-text events (cap 16)
	TTSStopChan chan struct{} `json:"-"` // closed when bot-tts-stopped received
```

**Step 2: Run tests to verify nothing breaks**

Run: `cd bin-pipecat-manager && go test ./models/pipecatcall/...`
Expected: PASS (no logic change, just new fields)

**Step 3: Commit**

```bash
git add bin-pipecat-manager/models/pipecatcall/session.go
git commit -m "NOJIRA-Sync-aimessage-intermediate-with-tts

- bin-pipecat-manager: Add TTSTextChan and TTSStopChan to Session for TTS event sync"
```

---

### Task 3: Write failing tests for dual-mode flush goroutine

This is the largest task — rewrite `runner_flush_test.go` with 15 test cases. The tests are written first (TDD); they will fail until the goroutine is updated in Task 4.

**Files:**
- Rewrite: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go`

**Step 1: Write the complete test file**

Replace the entire contents of `runner_flush_test.go` with the following. The existing 6 tests are preserved (updated to create TTS channels) and 9 new TTS-mode tests are added.

```go
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
		TTSStopChan:  make(chan struct{}),
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
		se.TTSStopChan = make(chan struct{})
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
// TTS mode tests (new behavior)
// ============================================================================

func Test_runLLMIntermediateFlush_TTSMode(t *testing.T) {

	t.Run("tts_text_chunks_produce_intermediates", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

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

		close(se.TTSStopChan)
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

	t.Run("tts_stop_publishes_final_with_tts_text_only", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

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

		// Signal LLM done (should NOT trigger final in TTS mode).
		close(se.LLMStopChan)

		// Small delay to verify goroutine keeps running.
		time.Sleep(50 * time.Millisecond)

		// Signal TTS done (should trigger final).
		close(se.TTSStopChan)
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

		// User interrupts — TTS stops early.
		close(se.TTSStopChan)
		<-se.LLMDoneChan

		mu.Lock()
		defer mu.Unlock()

		// Final text should be only what TTS received.
		if finalText != "Hello there." {
			t.Errorf("expected final text 'Hello there.', got '%s'", finalText)
		}
	})

	t.Run("llm_stop_before_tts_stop_goroutine_keeps_running", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

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

		// LLM finishes generating (but TTS is still speaking).
		close(se.LLMStopChan)

		// More TTS chunks arrive AFTER LLM stop.
		time.Sleep(50 * time.Millisecond)
		se.TTSTextChan <- " Second sentence."

		time.Sleep(50 * time.Millisecond)

		// TTS finishes speaking.
		close(se.TTSStopChan)
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
		h := &pipecatcallHandler{notifyHandler: mockNotify}

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

		close(se.TTSStopChan)
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

	t.Run("tts_stop_after_goroutine_already_exited_no_panic", func(t *testing.T) {
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

		// Simulate the read loop receiving bot-tts-stopped after goroutine exited.
		// In the read loop, this is guarded by `if se.LLMFlushing`.
		// This test verifies the guard works — no panic, no block.
		if se.LLMFlushing {
			close(se.TTSStopChan)
			<-se.LLMDoneChan // This would deadlock if LLMFlushing check is missing.
		}
		// If we get here, the guard correctly prevented the close/wait.
	})

	t.Run("metadata_correctness_in_tts_mode", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockNotify := notifyhandler.NewMockNotifyHandler(mc)
		h := &pipecatcallHandler{notifyHandler: mockNotify}

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

		close(se.TTSStopChan)
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
		se.TTSStopChan = make(chan struct{})
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
```

**Step 2: Run tests to verify they fail**

Run: `cd bin-pipecat-manager && go test -v -run Test_runLLMIntermediateFlush ./pkg/pipecatcallhandler/...`

Expected: Text-mode tests PASS (existing behavior preserved). TTS-mode tests FAIL because the goroutine doesn't handle `TTSTextChan` or `TTSStopChan` yet — they will hang or produce wrong results (the goroutine never reads from these channels).

**Note:** Some TTS tests may hang because the goroutine never exits via `TTSStopChan`. Run with `-timeout 30s` to limit hanging.

**Step 3: Commit the failing tests**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go
git commit -m "NOJIRA-Sync-aimessage-intermediate-with-tts

- bin-pipecat-manager: Add TTS-mode test cases for dual-mode flush goroutine (tests fail until implementation)"
```

---

### Task 4: Evolve flush goroutine to dual-mode

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:585-660` (rewrite `runLLMIntermediateFlush`)

**Step 1: Replace the flush goroutine implementation**

Replace the entire `runLLMIntermediateFlush` function (lines 585-660) with the dual-mode version:

```go
// runLLMIntermediateFlush is the per-generation flush goroutine.
//
// It supports two modes, auto-detected at runtime:
//   - Text mode (no TTS): 200ms timer batches LLM tokens into intermediate events.
//     bot-llm-stopped triggers drain + final event with full LLM text.
//   - TTS mode (voice calls): bot-tts-text events trigger intermediate events with
//     natural sentence chunks. bot-tts-stopped triggers final event with accumulated
//     TTS text only (not full LLM output). Timer-driven intermediates stop once TTS
//     mode is entered.
//
// Mode switches from text to TTS on the first bot-tts-text event received.
func (h *pipecatcallHandler) runLLMIntermediateFlush(se *pipecatcall.Session, messageID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runLLMIntermediateFlush",
		"pipecatcall_id": se.ID,
		"message_id":     messageID,
	})

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	defer close(se.LLMDoneChan)

	var llmFullText string  // accumulated from LLM tokens (used in text mode)
	var ttsFullText string  // accumulated from TTS chunks (used in TTS mode)
	var deltaBuffer string  // for timer-driven intermediates (text mode only)
	var sequence int
	var ttsReceived bool    // true once first bot-tts-text arrives

	for {
		select {
		case token := <-se.LLMTokenChan:
			llmFullText += token
			if !ttsReceived {
				deltaBuffer += token
			}

		case <-ticker.C:
			if !ttsReceived && deltaBuffer != "" {
				sequence++
				h.publishIntermediateEvent(se, messageID, deltaBuffer, sequence)
				log.Debugf("Published intermediate event (text mode). sequence: %d, delta_len: %d", sequence, len(deltaBuffer))
				deltaBuffer = ""
			}

		case ttsText := <-se.TTSTextChan:
			ttsReceived = true
			ttsFullText += ttsText
			sequence++
			h.publishIntermediateEvent(se, messageID, ttsText, sequence)
			log.Debugf("Published intermediate event (TTS mode). sequence: %d, tts_chunk_len: %d", sequence, len(ttsText))

		case <-se.LLMStopChan:
			if !ttsReceived {
				// Text mode: drain remaining tokens and publish final.
				for {
					select {
					case token := <-se.LLMTokenChan:
						llmFullText += token
						deltaBuffer += token
					default:
						goto llmDrained
					}
				}
			llmDrained:
				if deltaBuffer != "" {
					sequence++
					h.publishIntermediateEvent(se, messageID, deltaBuffer, sequence)
					log.Debugf("Published final intermediate event (text mode). sequence: %d, delta_len: %d", sequence, len(deltaBuffer))
				}

				h.publishFinalBotLLMEvent(se.Ctx, se, messageID, llmFullText)
				log.Debugf("Published final bot LLM event (text mode). full_text_len: %d", len(llmFullText))
				return
			}
			// TTS mode: note LLM done, keep running for TTS events.
			log.Debugf("LLM stopped in TTS mode, waiting for TTS stop.")

		case <-se.TTSStopChan:
			// TTS done: publish final with TTS-received text only.
			h.publishFinalBotLLMEvent(se.Ctx, se, messageID, ttsFullText)
			log.Debugf("Published final bot LLM event (TTS mode). tts_text_len: %d", len(ttsFullText))
			return

		case <-se.Ctx.Done():
			// Session teardown: drain both channels, publish partial final.
			for {
				select {
				case token := <-se.LLMTokenChan:
					llmFullText += token
				default:
					goto ctxLLMDrained
				}
			}
		ctxLLMDrained:
			for {
				select {
				case ttsText := <-se.TTSTextChan:
					ttsFullText += ttsText
					ttsReceived = true
				default:
					goto ctxTTSDrained
				}
			}
		ctxTTSDrained:
			finalText := llmFullText
			if ttsReceived {
				finalText = ttsFullText
			}
			if finalText != "" {
				h.publishFinalBotLLMEvent(context.Background(), se, messageID, finalText)
				log.Debugf("Context cancelled, published partial final bot LLM event. final_text_len: %d, tts_mode: %v", len(finalText), ttsReceived)
			} else {
				log.Debugf("Context cancelled, no text accumulated.")
			}
			return
		}
	}
}
```

**Step 2: Run all tests to verify text-mode tests still pass and TTS tests now pass**

Run: `cd bin-pipecat-manager && go test -v -timeout 60s -run Test_runLLMIntermediateFlush ./pkg/pipecatcallhandler/...`
Expected: All 15 tests PASS.

**Step 3: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go
git commit -m "NOJIRA-Sync-aimessage-intermediate-with-tts

- bin-pipecat-manager: Evolve flush goroutine to dual-mode (text/TTS) with auto-detection"
```

---

### Task 5: Add bot-tts-text and bot-tts-stopped handlers in read loop

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:515-555` (modify `receiveMessageFrameTypeMessage`)

**Step 1: Add TTSTextChan and TTSStopChan initialization in bot-llm-text handler**

In the `RTVIFrameTypeBotLLMText` case (around line 523-525), add the new channel creation alongside existing ones. Replace lines 522-527:

```go
		if !se.LLMFlushing {
			se.LLMMessageID = h.utilHandler.UUIDCreate()
			se.LLMTokenChan = make(chan string, 64)
			se.LLMStopChan = make(chan struct{})
			se.LLMDoneChan = make(chan struct{})
			se.TTSTextChan = make(chan string, 16)
			se.TTSStopChan = make(chan struct{})
			se.LLMFlushing = true
			go h.runLLMIntermediateFlush(se, se.LLMMessageID)
		}
```

**Step 2: Add bot-tts-text case**

After the `RTVIFrameTypeBotLLMStopped` case (after line 554), add the new `RTVIFrameTypeBotTTSText` case:

```go
	case pipecatframe.RTVIFrameTypeBotTTSText:
		msg := pipecatframe.RTVIBotTTSTextMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-tts-text message")
		}

		if se.LLMFlushing {
			select {
			case se.TTSTextChan <- msg.Data.Text:
			case <-se.LLMDoneChan:
				// Goroutine already exited (text mode finished first).
				log.Debugf("TTS text arrived but flush goroutine already exited.")
			}
		}

	case pipecatframe.RTVIFrameTypeBotTTSStopped:
		if se.LLMFlushing {
			close(se.TTSStopChan)
			<-se.LLMDoneChan
			se.LLMFlushing = false
		} else {
			log.Debugf("BotTTSStopped received but no flush goroutine is running.")
		}
```

**Step 3: Make bot-llm-stopped non-blocking**

Replace the `RTVIFrameTypeBotLLMStopped` case (lines 542-554) with the non-blocking version:

```go
	case pipecatframe.RTVIFrameTypeBotLLMStopped:
		msg := pipecatframe.RTVIBotLLMStoppedMessage{}
		if errUnmarshal := json.Unmarshal(m, &msg); errUnmarshal != nil {
			return errors.Wrapf(errUnmarshal, "could not unmarshal bot-llm-stopped message")
		}

		if se.LLMFlushing {
			close(se.LLMStopChan)
			// In text mode, the goroutine exits on LLMStopChan — wait for it.
			// In TTS mode, the goroutine keeps running for TTSStopChan — don't wait.
			// We use a non-blocking check: if LLMDoneChan is already closed, the
			// goroutine exited (text mode). Otherwise, it's in TTS mode.
			select {
			case <-se.LLMDoneChan:
				// Text mode: goroutine exited, reset state.
				se.LLMFlushing = false
			default:
				// TTS mode: goroutine still running, will exit on TTSStopChan.
				log.Debugf("BotLLMStopped in TTS mode, goroutine continues for TTS events.")
			}
		} else {
			log.Debugf("BotLLMStopped received but no tokens were received for this generation.")
		}
```

**Step 4: Run all tests**

Run: `cd bin-pipecat-manager && go test -v -timeout 60s ./pkg/pipecatcallhandler/...`
Expected: All tests PASS.

**Step 5: Run full verification workflow**

Run:
```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Sync-aimessage-intermediate-with-tts/bin-pipecat-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All pass.

**Step 6: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner.go
git commit -m "NOJIRA-Sync-aimessage-intermediate-with-tts

- bin-pipecat-manager: Add bot-tts-text and bot-tts-stopped handlers in read loop
- bin-pipecat-manager: Make bot-llm-stopped non-blocking for TTS mode compatibility"
```

---

### Task 6: Final verification and push

**Step 1: Run full verification one more time**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Sync-aimessage-intermediate-with-tts/bin-pipecat-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test -v ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Verify all tests pass with race detector**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Sync-aimessage-intermediate-with-tts/bin-pipecat-manager && \
go test -race -timeout 60s ./pkg/pipecatcallhandler/...
```

**Step 3: Check for conflicts with main**

```bash
cd /home/pchero/gitvoipbin/monorepo/.worktrees/NOJIRA-Sync-aimessage-intermediate-with-tts && \
git fetch origin main && \
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-Sync-aimessage-intermediate-with-tts
```

Then create PR with title `NOJIRA-Sync-aimessage-intermediate-with-tts`.

---

## Summary of Changes

| File | Change |
|------|--------|
| `bin-pipecat-manager/models/pipecatframe/helper.go` | Add `RTVIFrameTypeBotTTSText = "bot-tts-text"` constant |
| `bin-pipecat-manager/models/pipecatframe/helper_test.go` | Add test for new constant |
| `bin-pipecat-manager/models/pipecatcall/session.go` | Add `TTSTextChan chan string`, `TTSStopChan chan struct{}` |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | Dual-mode flush goroutine + `bot-tts-text`/`bot-tts-stopped` handlers + non-blocking `bot-llm-stopped` |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go` | 15 test cases: 5 text mode, 5 TTS mode, 2 mode switching, 4 integration |

**No changes needed:**
- Python side (`bot-tts-text` enabled by default in Pipecat >= 0.0.101)
- AI Manager (same event types and webhook structs)
- OpenAPI specs (no API-facing changes)
