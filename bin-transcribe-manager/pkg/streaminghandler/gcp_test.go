package streaminghandler

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// mockGCPClientStream implements grpc.ClientStream for testing
type mockGCPClientStream struct{}

func (m *mockGCPClientStream) Header() (metadata.MD, error) { return nil, nil }
func (m *mockGCPClientStream) Trailer() metadata.MD         { return nil }
func (m *mockGCPClientStream) CloseSend() error              { return nil }
func (m *mockGCPClientStream) Context() context.Context      { return context.Background() }
func (m *mockGCPClientStream) SendMsg(any) error             { return nil }
func (m *mockGCPClientStream) RecvMsg(any) error             { return nil }

// mockGCPStreamClient implements speechpb.Speech_StreamingRecognizeClient for testing
type mockGCPStreamClient struct {
	grpc.ClientStream
	responses []*speechpb.StreamingRecognizeResponse
	index     int
}

func (m *mockGCPStreamClient) Recv() (*speechpb.StreamingRecognizeResponse, error) {
	if m.index >= len(m.responses) {
		return nil, io.EOF
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *mockGCPStreamClient) Send(*speechpb.StreamingRecognizeRequest) error {
	return nil
}

func newMockGCPStreamClient(responses []*speechpb.StreamingRecognizeResponse) *mockGCPStreamClient {
	return &mockGCPStreamClient{
		ClientStream: &mockGCPClientStream{},
		responses:    responses,
		index:        0,
	}
}

// helper to create an interim response
func gcpInterimResponse(text string) *speechpb.StreamingRecognizeResponse {
	return &speechpb.StreamingRecognizeResponse{
		Results: []*speechpb.StreamingRecognitionResult{
			{
				IsFinal: false,
				Alternatives: []*speechpb.SpeechRecognitionAlternative{
					{Transcript: text},
				},
			},
		},
	}
}

// helper to create a final response
func gcpFinalResponse(text string) *speechpb.StreamingRecognizeResponse {
	return &speechpb.StreamingRecognizeResponse{
		Results: []*speechpb.StreamingRecognitionResult{
			{
				IsFinal: true,
				Alternatives: []*speechpb.SpeechRecognitionAlternative{
					{Transcript: text},
				},
			},
		},
	}
}

// helper to create an empty response (no results)
func gcpEmptyResponse() *speechpb.StreamingRecognizeResponse {
	return &speechpb.StreamingRecognizeResponse{
		Results: []*speechpb.StreamingRecognitionResult{},
	}
}

func Test_gcpProcessResult_InterimThenFinal(t *testing.T) {
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

	// Sequence: interim("hel"), interim("hello"), final("hello world")
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpInterimResponse("hel"),
		gcpInterimResponse("hello"),
		gcpFinalResponse("hello world"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	// Expected calls in order:
	gomock.InOrder(
		// First interim → speech_started + speech_interim
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Second interim → speech_interim only
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Final → speech_ended + transcript creation
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionIn, "hello world", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	h.gcpProcessResult(ctx, cancel, st, streamClient)
}

func Test_gcpProcessResult_FinalOnly(t *testing.T) {
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

	// Sequence: final only, no interim
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpFinalResponse("hello"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	// No speech_started or speech_ended expected (never entered speaking state)
	// Only transcript creation
	mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionOut, "hello", gomock.Any()).Return(&transcript.Transcript{}, nil)

	h.gcpProcessResult(ctx, cancel, st, streamClient)
}

func Test_gcpProcessResult_MultipleUtterances(t *testing.T) {
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

	// Two utterances: interim+final, then interim+final
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpInterimResponse("hi"),
		gcpFinalResponse("hi there"),
		gcpInterimResponse("bye"),
		gcpFinalResponse("bye now"),
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

	h.gcpProcessResult(ctx, cancel, st, streamClient)
}

func Test_gcpProcessResult_EmptyResults(t *testing.T) {
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

	// Only empty responses then EOF — no events expected
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpEmptyResponse(),
		gcpEmptyResponse(),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	// No mock expectations — no events should be published
	h.gcpProcessResult(ctx, cancel, st, streamClient)
}

func Test_gcpProcessResult_FinalWithEmptyTranscript(t *testing.T) {
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

	// Interim then final with empty transcript text — speech_ended fires but no transcript.Create
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpInterimResponse(""),
		gcpFinalResponse(""),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	gomock.InOrder(
		// Interim with empty text still triggers speech_started + speech_interim
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Final triggers speech_ended but empty transcript text → no transcript.Create
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
	)

	h.gcpProcessResult(ctx, cancel, st, streamClient)
}

func Test_gcpProcessResult_TranscriptCreateError(t *testing.T) {
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

	// Interim then final, but transcript.Create fails
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpInterimResponse("test"),
		gcpFinalResponse("test message"),
	})

	ctx := context.Background()
	_, cancel := context.WithCancel(ctx)

	gomock.InOrder(
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), customerID, transcribeID, transcript.DirectionIn, "test message", gomock.Any()).Return(nil, fmt.Errorf("db error")),
	)

	// Should not panic, just break out of loop
	h.gcpProcessResult(ctx, cancel, st, streamClient)
}

func Test_gcpProcessResult_ContextCanceled(t *testing.T) {
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

	// Provide responses but cancel context immediately
	streamClient := newMockGCPStreamClient([]*speechpb.StreamingRecognizeResponse{
		gcpInterimResponse("test"),
		gcpFinalResponse("test message"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// No events expected — context is already canceled
	// The function should return quickly. Give it a timeout to confirm.
	done := make(chan struct{})
	go func() {
		h.gcpProcessResult(ctx, cancel, st, streamClient)
		close(done)
	}()

	select {
	case <-done:
		// good, exited promptly
	case <-time.After(2 * time.Second):
		t.Fatal("gcpProcessResult did not exit after context cancellation")
	}
}
