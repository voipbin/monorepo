package streaminghandler

import (
	"context"
	"testing"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Start(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		transcribeID  uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcript.Direction

		responseUUID          uuid.UUID
		responseExternalMedia *cmexternalmedia.ExternalMedia
	}{
		{
			name: "normal - verifies ExternalMediaStart params and cleanup on WebSocket failure",

			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			transcribeID:  uuid.FromStringOrNil("e210a336-e9df-11ef-b5e9-bbbc7edb0445"),
			referenceType: transcribe.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			direction:     transcript.DirectionIn,

			responseUUID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID:       uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
				MediaURI: "ws://127.0.0.1:0/invalid", // invalid URI so websocketConnect fails
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &streamingHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, gomock.Any())

			// Verify ExternalMediaStart is called with WebSocket parameters
			mockReq.EXPECT().CallV1ExternalMediaStart(
				ctx,
				tt.responseUUID,
				cmexternalmedia.ReferenceType(tt.referenceType),
				tt.referenceID,
				"INCOMING",            // WebSocket: service dials out
				defaultEncapsulation,  // "none"
				defaultTransport,      // "websocket"
				"",                    // transportData
				defaultConnectionType, // "server"
				defaultFormat,         // "slin"
				cmexternalmedia.Direction(tt.direction),
				cmexternalmedia.DirectionNone,
			).Return(tt.responseExternalMedia, nil)

			// WebSocket connect will fail → expect cleanup of orphaned external media and streaming record
			mockReq.EXPECT().CallV1ExternalMediaStop(ctx, tt.responseExternalMedia.ID).Return(tt.responseExternalMedia, nil)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStopped, gomock.Any())

			// Start should return error because websocketConnect fails
			_, err := h.Start(ctx, tt.customerID, tt.transcribeID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err == nil {
				t.Error("Expected error from Start (WebSocket connect should fail), got nil")
			}

			// Verify streaming record was cleaned up from the map
			if _, errGet := h.Get(ctx, tt.responseUUID); errGet == nil {
				t.Error("Expected streaming record to be deleted after WebSocket connect failure")
			}
		})
	}
}
