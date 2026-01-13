package streaminghandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_SayInit(t *testing.T) {

	tests := []struct {
		name       string
		streamings []*streaming.Streaming

		id        uuid.UUID
		messageID uuid.UUID

		expectRes *streaming.Streaming
	}{
		{
			name: "normal",
			streamings: []*streaming.Streaming{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("cfbd4912-87a2-11f0-b7c9-f7bb42a8c7db"),
					},
					VendorName:   streaming.VendorNameElevenlabs,
					VendorConfig: &ElevenlabsConfig{},
				},
			},

			id:        uuid.FromStringOrNil("cfbd4912-87a2-11f0-b7c9-f7bb42a8c7db"),
			messageID: uuid.FromStringOrNil("cfefc05e-87a2-11f0-8fcf-cf9eccb8babf"),

			expectRes: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cfbd4912-87a2-11f0-b7c9-f7bb42a8c7db"),
				},
				MessageID:    uuid.FromStringOrNil("cfefc05e-87a2-11f0-8fcf-cf9eccb8babf"),
				VendorName:   streaming.VendorNameElevenlabs,
				VendorConfig: &ElevenlabsConfig{},
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

			res, err := h.SayInit(ctx, tt.id, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SayAdd(t *testing.T) {

	tests := []struct {
		name       string
		streamings []*streaming.Streaming

		id        uuid.UUID
		messageID uuid.UUID
		text      string
	}{
		{
			name: "normal",
			streamings: []*streaming.Streaming{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d1d734fc-83d2-11f0-860b-13abcaf70fd9"),
					},
					MessageID:    uuid.FromStringOrNil("d2096954-83d2-11f0-93f4-6761531ebad2"),
					VendorName:   streaming.VendorNameElevenlabs,
					VendorConfig: &ElevenlabsConfig{},
				},
			},

			id:        uuid.FromStringOrNil("d1d734fc-83d2-11f0-860b-13abcaf70fd9"),
			messageID: uuid.FromStringOrNil("d2096954-83d2-11f0-93f4-6761531ebad2"),
			text:      "Hello, this is a test message.",
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

			mockEleven.EXPECT().SayAdd(gomock.Any(), tt.text.Return(nil).Times(1)

			if err := h.SayAdd(ctx, tt.id, tt.messageID, tt.text); err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
		})
	}
}
