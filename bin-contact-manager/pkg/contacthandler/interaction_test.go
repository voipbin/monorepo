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
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_EventCallCreated(t *testing.T) {
	tests := []struct {
		name string

		message *callmodel.WebhookMessage

		responseUUID    uuid.UUID
		responseCurTime *time.Time
		expectCaseID    uuid.UUID

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
			expectCaseID:    uuid.FromStringOrNil("bb000001-0000-0000-0000-0000000000ca"),

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
			expectCaseID:    uuid.FromStringOrNil("bb000002-0000-0000-0000-0000000000ca"),

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
			expectCaseID:    uuid.FromStringOrNil("bb000003-0000-0000-0000-0000000000ca"),

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
		{
			name: "incoming call - peer is agent extension - projection skipped",

			message: &callmodel.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa000004-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("aa000004-0000-0000-0000-000000000002"),
				},
				Source: commonaddress.Address{
					Type:   commonaddress.TypeExtension,
					Target: "extensionSrc",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "localTelIncoming",
				},
				Direction: callmodel.DirectionIncoming,
			},

			expectInteraction: nil,
		},
		{
			name: "outgoing call - peer is direct-SIP leg - projection skipped",

			message: &callmodel.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aa000005-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("aa000005-0000-0000-0000-000000000002"),
				},
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "localTelOutgoing",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:peer@example.com",
				},
				Direction: callmodel.DirectionOutgoing,
			},

			expectInteraction: nil,
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
			mockCase := casehandler.NewMockCaseHandler(mc)
			h := contactHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   mockUtil,
				caseHandler:   mockCase,
			}
			ctx := context.Background()

			if tt.expectInteraction != nil {
				mockCase.EXPECT().GetOrCreate(ctx, tt.expectInteraction.CustomerID, gomock.Any(), commonaddress.Type(tt.expectInteraction.PeerType), tt.expectInteraction.PeerTarget, "call", gomock.Any()).Return(&kase.Case{ID: tt.expectCaseID}, nil)
				expected := *tt.expectInteraction
				expected.CaseID = &tt.expectCaseID
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockDB.EXPECT().InteractionCreate(ctx, &expected).Return(nil)
			}

			if err := h.EventCallCreated(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventConversationMessageCreated(t *testing.T) {
	hintCaseID := uuid.FromStringOrNil("ee000001-0000-0000-0000-000000000099")

	tests := []struct {
		name string

		message *convmsg.WebhookMessage

		responseUUID    uuid.UUID
		responseCurTime *time.Time
		expectCaseID    uuid.UUID

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
				CaseID: &hintCaseID,
			},

			responseUUID:    uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000001"),
			responseCurTime: func() *time.Time { t := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC); return &t }(),
			expectCaseID:    uuid.FromStringOrNil("dd000001-0000-0000-0000-0000000000ca"),

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
		{
			name: "outgoing web_session message - peer is synthetic web session type - projection skipped",

			message: &convmsg.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cc000002-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("cc000002-0000-0000-0000-000000000002"),
				},
				Direction: convmsg.DirectionIncoming,
				Source: commonaddress.Address{
					Type:   "web_session",
					Target: "web_session:xyz",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeAI,
					Target: "",
				},
			},

			expectInteraction: nil,
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
			mockCase := casehandler.NewMockCaseHandler(mc)
			h := contactHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				utilHandler:   mockUtil,
				caseHandler:   mockCase,
			}
			ctx := context.Background()

			if tt.expectInteraction != nil {
				// Regression guard for the round-1 review defect: the
				// message's CaseID hint MUST be forwarded verbatim to
				// GetOrCreate's caseIDHint parameter, not silently
				// dropped as a hardcoded nil. gomock.Eq on the actual
				// pointer VALUE (not gomock.Any()) is required here --
				// gomock.Any() would pass identically whether the hint
				// were forwarded or discarded, which is exactly how this
				// regression escaped detection in round 1.
				mockCase.EXPECT().GetOrCreate(ctx, tt.expectInteraction.CustomerID, gomock.Any(), commonaddress.Type(tt.expectInteraction.PeerType), tt.expectInteraction.PeerTarget, "conversation_message", tt.message.CaseID).Return(&kase.Case{ID: tt.expectCaseID}, nil)
				expected := *tt.expectInteraction
				expected.CaseID = &tt.expectCaseID
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockDB.EXPECT().InteractionCreate(ctx, &expected).Return(nil)
			}

			if err := h.EventConversationMessageCreated(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_InteractionList_unfiltered covers the new since-based unfiltered branch
// added alongside PR #1054/§3.2 of the design doc: when peerType/peerTarget/
// contactID/addressID are all zero but since is non-zero, InteractionList must
// route to h.db.InteractionList with an empty peer/addressSet and the since
// value forwarded verbatim (not the interactionListByContact/ByAddress paths).
func Test_InteractionList_unfiltered(t *testing.T) {
	customerID := uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000001")
	since := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		customerID uuid.UUID
		since      time.Time

		responseItems []*interaction.Interaction
		expectErr     bool
	}{
		{
			name:       "normal - since forwarded, unfiltered branch reached",
			customerID: customerID,
			since:      since,

			responseItems: []*interaction.Interaction{
				{
					ID:         uuid.FromStringOrNil("dd000001-0000-0000-0000-000000000002"),
					CustomerID: customerID,
				},
			},
			expectErr: false,
		},
		{
			name:       "bad request - since is zero-value, no filter provided",
			customerID: customerID,
			since:      time.Time{},

			expectErr: true,
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

			if !tt.expectErr {
				mockDB.EXPECT().
					InteractionList(ctx, tt.customerID, uint64(21), "", "", "", nil, tt.since).
					Return(tt.responseItems, nil)
			}

			items, _, err := h.InteractionList(ctx, tt.customerID, 20, "", "", "", uuid.Nil, uuid.Nil, tt.since)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if len(items) != len(tt.responseItems) {
				t.Errorf("Wrong match. expect: %d items, got: %d", len(tt.responseItems), len(items))
			}
		})
	}
}
