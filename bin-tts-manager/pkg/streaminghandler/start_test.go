package streaminghandler

import (
	"context"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Start(t *testing.T) {

	tests := []struct {
		name          string
		listenAddress string

		customerID    uuid.UUID
		transcribeID  uuid.UUID
		referenceType streaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		gender        streaming.Gender
		direction     streaming.Direction

		responseUUID          uuid.UUID
		responseExternalMedia *cmexternalmedia.ExternalMedia

		expectExternalMediaID uuid.UUID
		expectRes             *streaming.Streaming
	}{
		{
			name:          "normal",
			listenAddress: "localhost:8090",

			customerID:    uuid.FromStringOrNil("e1d034f4-e9df-11ef-990b-2f91a795184b"),
			transcribeID:  uuid.FromStringOrNil("e210a336-e9df-11ef-b5e9-bbbc7edb0445"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
			language:      "en-US",
			direction:     streaming.DirectionIncoming,

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
				ReferenceType: streaming.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e24d0934-e9df-11ef-9193-e30e5103f5bd"),
				Language:      "en-US",
				Direction:     streaming.DirectionIncoming,
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
				utilHandler:    mockUtil,
				requestHandler: mockReq,
				notifyHandler:  mockNotify,
				mapStreaming:   make(map[uuid.UUID]*streaming.Streaming),

				listenAddress: tt.listenAddress,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingCreated, gomock.Any())
			mockReq.EXPECT().CallV1ExternalMediaStart(ctx, tt.responseUUID, cmexternalmedia.ReferenceType(tt.referenceType), tt.referenceID, true, tt.listenAddress, defaultEncapsulation, defaultTransport, defaultConnectionType, defaultFormat, string(tt.direction)).Return(tt.responseExternalMedia, nil)

			res, err := h.Start(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.gender, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			tt.expectRes.ChanDone = res.ChanDone
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Say(t *testing.T) {

	tests := []struct {
		name       string
		streamings []*streaming.Streaming

		id   uuid.UUID
		text string
	}{
		{
			name: "normal",
			streamings: []*streaming.Streaming{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cd6ea4f8-5af2-11f0-9694-5fb9b4e116cb"),
					},
					Vendor: streaming.VendorElevenlabs,
				},
			},

			id:   uuid.FromStringOrNil("cd6ea4f8-5af2-11f0-9694-5fb9b4e116cb"),
			text: "Hello, this is a test message.",
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
			mockEleven := NewMockstreamer(mc)

			h := &streamingHandler{
				utilHandler:    mockUtil,
				requestHandler: mockReq,
				notifyHandler:  mockNotify,
				mapStreaming:   make(map[uuid.UUID]*streaming.Streaming),

				elevenlabsHandler: mockEleven,
			}
			ctx := context.Background()

			for _, s := range tt.streamings {
				h.mapStreaming[s.ID] = s
			}

			mockEleven.EXPECT().AddText(ctx, gomock.Any(), tt.text).Return(nil).Times(1)

			if err := h.Say(ctx, tt.id, tt.text); err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
		})
	}
}
