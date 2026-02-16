package streaminghandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming/types"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// mockAWSEventStream implements awsEventStream for testing
type mockAWSEventStream struct {
	ch chan types.TranscriptResultStream
}

func (m *mockAWSEventStream) Events() <-chan types.TranscriptResultStream {
	return m.ch
}

func (m *mockAWSEventStream) Close() error {
	return nil
}

func newMockAWSEventStream(events []types.TranscriptResultStream) *mockAWSEventStream {
	ch := make(chan types.TranscriptResultStream, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return &mockAWSEventStream{ch: ch}
}

func awsStr(s string) *string {
	return &s
}

// helper to create a partial (interim) event
func awsPartialEvent(text string) *types.TranscriptResultStreamMemberTranscriptEvent {
	return &types.TranscriptResultStreamMemberTranscriptEvent{
		Value: types.TranscriptEvent{
			Transcript: &types.Transcript{
				Results: []types.Result{
					{
						IsPartial: true,
						Alternatives: []types.Alternative{
							{Transcript: awsStr(text)},
						},
					},
				},
			},
		},
	}
}

// helper to create a final event
func awsFinalEvent(text string) *types.TranscriptResultStreamMemberTranscriptEvent {
	return &types.TranscriptResultStreamMemberTranscriptEvent{
		Value: types.TranscriptEvent{
			Transcript: &types.Transcript{
				Results: []types.Result{
					{
						IsPartial: false,
						Alternatives: []types.Alternative{
							{Transcript: awsStr(text)},
						},
					},
				},
			},
		},
	}
}

func Test_awsProcessEvents_PartialThenFinal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	stream := newMockAWSEventStream([]types.TranscriptResultStream{
		awsPartialEvent("hel"),
		awsPartialEvent("hello"),
		awsFinalEvent("hello world"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	gomock.InOrder(
		// First partial → speech_started + speech_interim
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Second partial → speech_interim only
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Final → speech_ended + transcript creation
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionIn, "hello world", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	h.awsProcessEvents(ctx, cancel, st, stream)
}

func Test_awsProcessEvents_FinalOnly(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionOut,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	stream := newMockAWSEventStream([]types.TranscriptResultStream{
		awsFinalEvent("hello"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	// No speech_started or speech_ended expected (never entered speaking state)
	mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionOut, "hello", gomock.Any()).Return(&transcript.Transcript{}, nil)

	h.awsProcessEvents(ctx, cancel, st, stream)
}

func Test_awsProcessEvents_MultipleUtterances(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	stream := newMockAWSEventStream([]types.TranscriptResultStream{
		awsPartialEvent("hi"),
		awsFinalEvent("hi there"),
		awsPartialEvent("bye"),
		awsFinalEvent("bye now"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	gomock.InOrder(
		// First utterance
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionIn, "hi there", gomock.Any()).Return(&transcript.Transcript{}, nil),
		// Second utterance
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionIn, "bye now", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	h.awsProcessEvents(ctx, cancel, st, stream)
}

func Test_awsProcessEvents_EmptyStream(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	// Empty stream — no events
	stream := newMockAWSEventStream(nil)

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	// No mock expectations — no events should be published
	h.awsProcessEvents(ctx, cancel, st, stream)
}

func Test_awsProcessEvents_FinalWithEmptyTranscript(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	stream := newMockAWSEventStream([]types.TranscriptResultStream{
		awsPartialEvent(""),
		awsFinalEvent(""),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	gomock.InOrder(
		// Partial with empty text still triggers speech_started + speech_interim
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Final triggers speech_ended but empty transcript text → no transcript.Create
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
	)

	h.awsProcessEvents(ctx, cancel, st, stream)
}

func Test_awsProcessEvents_TranscriptCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	stream := newMockAWSEventStream([]types.TranscriptResultStream{
		awsPartialEvent("test"),
		awsFinalEvent("test message"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	gomock.InOrder(
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionIn, "test message", gomock.Any()).Return(nil, fmt.Errorf("db error")),
	)

	h.awsProcessEvents(ctx, cancel, st, stream)
}

func Test_awsProcessEvents_ContextCanceled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	transcribeID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	streamingID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         streamingID,
			CustomerID: customerID,
		},
		TranscribeID: transcribeID,
		Language:     "en-US",
		Direction:    transcript.DirectionIn,
	}

	h := &streamingHandler{
		notifyHandler:     mockNotify,
		transcriptHandler: mockTranscript,
	}

	// Open channel (never closed, no events) so only ctx.Done() fires
	stream := &mockAWSEventStream{ch: make(chan types.TranscriptResultStream)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	done := make(chan struct{})
	go func() {
		h.awsProcessEvents(ctx, cancel, st, stream)
		close(done)
	}()

	select {
	case <-done:
		// good, exited promptly
	case <-time.After(2 * time.Second):
		t.Fatal("awsProcessEvents did not exit after context cancellation")
	}
}
