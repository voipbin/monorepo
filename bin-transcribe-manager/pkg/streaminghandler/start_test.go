package streaminghandler

import (
	"context"
	"monorepo/bin-call-manager/models/externalmedia"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Start(t *testing.T) {

	tests := []struct {
		name          string
		listenAddress string

		customerID    uuid.UUID
		transcribeID  uuid.UUID
		referenceType transcribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     transcript.Direction

		responseUUID          uuid.UUID
		responseExternalMedia *cmexternalmedia.ExternalMedia

		expectExternalMediaID uuid.UUID
		expectRes             *streaming.Streaming
	}{
		{
			name:          "normal",
			listenAddress: "localhost:8080",

			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			transcribeID:  uuid.FromStringOrNil("e210a336-e9df-11ef-b5e9-bbbc7edb0445"),
			referenceType: transcribe.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			direction:     transcript.DirectionIn,

			responseUUID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			responseExternalMedia: &cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			},

			expectExternalMediaID: uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
			expectRes: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633"),
					CustomerID: uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
				},
				TranscribeID: uuid.FromStringOrNil("e210a336-e9df-11ef-b5e9-bbbc7edb0445"),
				Language:     "en-US",
				Direction:    transcript.DirectionIn,
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

				listenAddress: tt.listenAddress,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, gomock.Any())
			mockReq.EXPECT().CallV1ExternalMediaStart(
				ctx,
				tt.responseUUID,
				cmexternalmedia.ReferenceType(tt.referenceType),
				tt.referenceID,
				tt.listenAddress,
				defaultEncapsulation,
				defaultTransport,
				defaultConnectionType,
				defaultFormat,
				externalmedia.Direction(tt.direction),
				externalmedia.DirectionNone,
			).Return(tt.responseExternalMedia, nil)

			res, err := h.Start(ctx, tt.customerID, tt.transcribeID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
