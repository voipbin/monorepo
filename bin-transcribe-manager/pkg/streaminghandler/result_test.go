package streaminghandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func newTestResultProcessor(mockNotify *notifyhandler.MockNotifyHandler, mockTranscript *transcripthandler.MockTranscriptHandler) (*resultProcessor, *streaming.Streaming) {
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

	rp := newResultProcessor(st, mockNotify, mockTranscript)
	return rp, st
}

func Test_resultProcessor_InterimThenFinal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	rp, st := newTestResultProcessor(mockNotify, mockTranscript)
	ctx := context.Background()

	gomock.InOrder(
		// First interim → speech_started + speech_interim
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Second interim → speech_interim only
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		// Final → speech_ended + transcript creation
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "hello world", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	rp.process(ctx, sttResult{isFinal: false, message: "hel"})
	rp.process(ctx, sttResult{isFinal: false, message: "hello"})
	rp.process(ctx, sttResult{isFinal: true, message: "hello world"})
}

func Test_resultProcessor_FinalOnly(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	rp, st := newTestResultProcessor(mockNotify, mockTranscript)
	ctx := context.Background()

	// No speech_started or speech_ended — never entered speaking state
	mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "hello", gomock.Any()).Return(&transcript.Transcript{}, nil)

	rp.process(ctx, sttResult{isFinal: true, message: "hello"})
}

func Test_resultProcessor_MultipleUtterances(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	rp, st := newTestResultProcessor(mockNotify, mockTranscript)
	ctx := context.Background()

	gomock.InOrder(
		// First utterance
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "hi there", gomock.Any()).Return(&transcript.Transcript{}, nil),
		// Second utterance
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "bye now", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	rp.process(ctx, sttResult{isFinal: false, message: "hi"})
	rp.process(ctx, sttResult{isFinal: true, message: "hi there"})
	rp.process(ctx, sttResult{isFinal: false, message: "bye"})
	rp.process(ctx, sttResult{isFinal: true, message: "bye now"})
}

func Test_resultProcessor_FinalWithEmptyTranscript(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	rp, st := newTestResultProcessor(mockNotify, mockTranscript)
	ctx := context.Background()

	gomock.InOrder(
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
	)

	rp.process(ctx, sttResult{isFinal: false, message: ""})
	rp.process(ctx, sttResult{isFinal: true, message: ""})
}

func Test_resultProcessor_TranscriptCreateError_Continues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	rp, st := newTestResultProcessor(mockNotify, mockTranscript)
	ctx := context.Background()

	gomock.InOrder(
		// First utterance — create fails
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "test message", gomock.Any()).Return(nil, fmt.Errorf("db error")),
		// Second utterance — still processes (continues on error)
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechStarted, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechInterim, gomock.Any()),
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), st.CustomerID, streaming.EventTypeSpeechEnded, gomock.Any()),
		mockTranscript.EXPECT().Create(gomock.Any(), st.CustomerID, st.TranscribeID, transcript.DirectionIn, "second message", gomock.Any()).Return(&transcript.Transcript{}, nil),
	)

	rp.process(ctx, sttResult{isFinal: false, message: "test"})
	rp.process(ctx, sttResult{isFinal: true, message: "test message"})
	rp.process(ctx, sttResult{isFinal: false, message: "second"})
	rp.process(ctx, sttResult{isFinal: true, message: "second message"})
}
