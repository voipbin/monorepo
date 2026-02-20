package streaminghandler

import (
	"context"
	"fmt"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Start(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		activeflowID  uuid.UUID
		referenceType streaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		gender        streaming.Gender
		direction     streaming.Direction

		responseUUID          uuid.UUID
		responseExternalMedia *cmexternalmedia.ExternalMedia

		// startExternalMedia now dials a WebSocket after creating external media,
		// so the "normal" case will fail at websocketConnect in unit tests.
		expectErr bool
	}{
		{
			name: "normal - verifies CallV1ExternalMediaStart is called with INCOMING host",

			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			activeflowID:  uuid.FromStringOrNil("dfe51622-87c4-11f0-9fbc-0be63c71e5fc"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			direction:     streaming.DirectionIncoming,

			responseUUID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			},

			// websocketConnect will fail because there is no real WebSocket server
			expectErr: true,
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
				utilHandler:    mockUtil,
				requestHandler: mockReq,
				notifyHandler:  mockNotify,
				mapStreaming:   make(map[uuid.UUID]*streaming.Streaming),
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingCreated, gomock.Any())
			mockReq.EXPECT().CallV1ExternalMediaStart(
				ctx,
				tt.responseUUID,
				cmexternalmedia.ReferenceType(tt.referenceType),
				tt.referenceID,
				"INCOMING",
				defaultEncapsulation,
				defaultTransport,
				"", // transportData
				defaultConnectionType,
				formatSlin16, // empty provider defaults to ElevenLabs (slin16)
				cmexternalmedia.DirectionNone,
				cmexternalmedia.Direction(tt.direction),
			).Return(tt.responseExternalMedia, nil)

			// websocketConnect fails in tests, so the cleanup path calls ExternalMediaStop
			mockReq.EXPECT().CallV1ExternalMediaStop(ctx, tt.responseExternalMedia.ID).Return(tt.responseExternalMedia, nil)

			_, err := h.Start(ctx, tt.customerID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.language, tt.gender, tt.direction)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StartWithID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		customerID    uuid.UUID
		referenceType streaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		provider      string
		voiceID       string
		direction     streaming.Direction

		responseExternalMedia    *cmexternalmedia.ExternalMedia
		responseExternalMediaErr error

		expectErr bool
	}{
		{
			name: "elevenlabs provider uses slin16 format",

			id:            uuid.FromStringOrNil("f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "voice123",
			direction:     streaming.DirectionIncoming,

			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			},

			// websocketConnect will fail because there is no real WebSocket server
			expectErr: true,
		},
		{
			name: "gcp provider uses ulaw format",

			id:            uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			provider:      "gcp",
			voiceID:       "en-US-Wavenet-D",
			direction:     streaming.DirectionIncoming,

			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			},

			expectErr: true,
		},
		{
			name: "aws provider uses slin format",

			id:            uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			provider:      "aws",
			voiceID:       "Joanna",
			direction:     streaming.DirectionIncoming,

			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			},

			expectErr: true,
		},
		{
			name: "create error duplicate ID",

			id:            uuid.FromStringOrNil("f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			expectErr: true,
		},
		{
			name: "external media error",

			id:            uuid.FromStringOrNil("f1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExternalMediaErr: fmt.Errorf("external media error"),

			expectErr: true,
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
				utilHandler:    mockUtil,
				requestHandler: mockReq,
				notifyHandler:  mockNotify,
				mapStreaming:   make(map[uuid.UUID]*streaming.Streaming),
			}
			ctx := context.Background()

			// For "create error duplicate ID" case, pre-populate the map
			if tt.name == "create error duplicate ID" {
				h.mapStreaming[tt.id] = &streaming.Streaming{}
			} else {
				mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingCreated, gomock.Any())
				mockReq.EXPECT().CallV1ExternalMediaStart(
					ctx,
					tt.id,
					cmexternalmedia.ReferenceType(tt.referenceType),
					tt.referenceID,
					"INCOMING",
					defaultEncapsulation,
					defaultTransport,
					"", // transportData
					defaultConnectionType,
					formatForProvider(tt.provider),
					cmexternalmedia.DirectionNone,
					cmexternalmedia.Direction(tt.direction),
				).Return(tt.responseExternalMedia, tt.responseExternalMediaErr)

				// websocketConnect fails in tests when ExternalMediaStart succeeds,
				// so the cleanup path calls ExternalMediaStop
				if tt.responseExternalMediaErr == nil && tt.responseExternalMedia != nil {
					mockReq.EXPECT().CallV1ExternalMediaStop(ctx, tt.responseExternalMedia.ID).Return(tt.responseExternalMedia, nil)
				}
			}

			_, err := h.StartWithID(ctx, tt.id, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.provider, tt.voiceID, tt.direction)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
