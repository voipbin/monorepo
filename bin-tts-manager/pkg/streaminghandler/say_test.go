package streaminghandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Say(t *testing.T) {

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
						ID: uuid.FromStringOrNil("cd6ea4f8-5af2-11f0-9694-5fb9b4e116cb"),
					},
					VendorName:   streaming.VendorNameElevenlabs,
					VendorConfig: &ElevenlabsConfig{},
				},
			},

			id:        uuid.FromStringOrNil("cd6ea4f8-5af2-11f0-9694-5fb9b4e116cb"),
			messageID: uuid.FromStringOrNil("c796296e-83d0-11f0-9666-cb5cc4b71eb0"),
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

			mockEleven.EXPECT().AddText(gomock.Any(), tt.text).Return(nil).Times(1)

			if err := h.Say(ctx, tt.id, tt.messageID, tt.text); err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
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

			mockEleven.EXPECT().AddText(gomock.Any(), tt.text).Return(nil).Times(1)

			if err := h.SayAdd(ctx, tt.id, tt.messageID, tt.text); err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}
		})
	}
}
