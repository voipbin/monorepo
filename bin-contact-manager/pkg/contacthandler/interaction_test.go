package contacthandler

import (
	"context"
	"testing"
	"time"

	callmodel "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	convmsg "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_EventCallCreated(t *testing.T) {
	tests := []struct {
		name string

		message *callmodel.WebhookMessage

		responseUUID uuid.UUID
		responseCurTime *time.Time

		expectInteraction *interaction.Interaction
	}{
		{
			name: "incoming call - peer=source, local=destination",

			message: &callmodel.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000002"),
				},
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "peerTelIncoming",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "localTelIncoming",
				},
				Direction: callmodel.DirectionIncoming,
			},

			responseUUID:    uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
			responseCurTime: func() *time.Time { t := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC); return &t }(),

			expectInteraction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("bb000001-0000-0000-0000-000000000001"),
				CustomerID:    uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000002"),
				Direction:     "incoming",
				PeerType:      "tel",
				PeerTarget:    "peerTelIncoming",
				LocalType:     "tel",
				LocalTarget:   "localTelIncoming",
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("aa000001-0000-0000-0000-000000000001"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC); return &t }(),
			},
		},
		{
			name: "outgoing call - peer=destination, local=source",

			message: &callmodel.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000002"),
				},
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "localTelOutgoing",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "peerTelOutgoing",
				},
				Direction: callmodel.DirectionOutgoing,
			},

			responseUUID:    uuid.FromStringOrNil("bb000002-0000-0000-0000-000000000001"),
			responseCurTime: func() *time.Time { t := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC); return &t }(),

			expectInteraction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("bb000002-0000-0000-0000-000000000001"),
				CustomerID:    uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000002"),
				Direction:     "outgoing",
				PeerType:      "tel",
				PeerTarget:    "peerTelOutgoing",
				LocalType:     "tel",
				LocalTarget:   "localTelOutgoing",
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("aa000002-0000-0000-0000-000000000001"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC); return &t }(),
			},
		},
		{
			name: "unknown direction - zero peer/local, row still created",

			message: &callmodel.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000002"),
				},
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "someSrc",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "someDst",
				},
				Direction: "", // unknown direction (DirectionNond)
			},

			responseUUID:    uuid.FromStringOrNil("bb000003-0000-0000-0000-000000000001"),
			responseCurTime: func() *time.Time { t := time.Date(2026, 6, 28, 13, 0, 0, 0, time.UTC); return &t }(),

			expectInteraction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("bb000003-0000-0000-0000-000000000001"),
				CustomerID:    uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000002"),
				Direction:     "",
				PeerType:      "",
				PeerTarget:    "",
				LocalType:     "",
				LocalTarget:   "",
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("aa000003-0000-0000-0000-000000000001"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 13, 0, 0, 0, time.UTC); return &t }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := contactHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockDB.EXPECT().InteractionCreate(ctx, tt.expectInteraction).Return(nil)

			if err := h.EventCallCreated(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventConversationMessageCreated(t *testing.T) {
	tests := []struct {
		name string

		message *convmsg.WebhookMessage

		responseUUID    uuid.UUID
		responseCurTime *time.Time

		expectInteraction *interaction.Interaction
	}{
		{
			name: "incoming LINE message - peer=source, local=destination",

			message: &convmsg.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cc000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("cc000001-0000-0000-0000-000000000002"),
				},
				Direction: convmsg.DirectionIncoming,
				Source: commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "",
				},
			},

			responseUUID:    uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000001"),
			responseCurTime: func() *time.Time { t := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC); return &t }(),

			expectInteraction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000001"),
				CustomerID:    uuid.FromStringOrNil("cc000001-0000-0000-0000-000000000002"),
				Direction:     "incoming",
				PeerType:      "line",
				PeerTarget:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				LocalType:     "line",
				LocalTarget:   "",
				ReferenceType: "conversation_message",
				ReferenceID:   uuid.FromStringOrNil("cc000001-0000-0000-0000-000000000001"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC); return &t }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := contactHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockDB.EXPECT().InteractionCreate(ctx, tt.expectInteraction).Return(nil)

			if err := h.EventConversationMessageCreated(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
