package conversationhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	emmemail "monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

func Test_EmailEventSent(t *testing.T) {

	tests := []struct {
		name string

		email *emmemail.Email

		// per-destination existing-message lookups (in order); empty slice = no existing
		existingByDest [][]*message.Message
		// per-destination conversation returned by ConversationGetBySelfAndPeer
		conversationByDest []*conversation.Conversation

		expectCreateCount int
	}{
		{
			name: "single destination, new outgoing message",

			email: &emmemail.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "sender@voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{Type: commonaddress.TypeEmail, Target: "customer@example.com"},
				},
				Subject: "Hello",
				Content: "This is the body.",
			},

			existingByDest: [][]*message.Message{
				{},
			},
			conversationByDest: []*conversation.Conversation{
				{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")}},
			},

			expectCreateCount: 1,
		},
		{
			name: "multi destination fan-out produces N messages",

			email: &emmemail.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				},
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "sender@voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{Type: commonaddress.TypeEmail, Target: "a@example.com"},
					{Type: commonaddress.TypeEmail, Target: "b@example.com"},
				},
				Subject: "Bulk",
				Content: "Body",
			},

			existingByDest: [][]*message.Message{
				{},
				{},
			},
			conversationByDest: []*conversation.Conversation{
				{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000002")}},
				{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000003")}},
			},

			expectCreateCount: 2,
		},
		{
			name: "duplicate event is a no-op (existing message found)",

			email: &emmemail.Email{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
					CustomerID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeEmail,
					Target: "sender@voipbin.net",
				},
				Destinations: []commonaddress.Address{
					{Type: commonaddress.TypeEmail, Target: "customer@example.com"},
				},
				Subject: "Hello",
				Content: "Body",
			},

			existingByDest: [][]*message.Message{
				{
					{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000001")}},
				},
			},
			conversationByDest: nil,

			expectCreateCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{
				db:             mockDB,
				messageHandler: mockMessage,
			}

			ctx := context.Background()

			for i, dest := range tt.email.Destinations {
				normalizedPeer, _ := commonaddress.NormalizeTarget(dest.Type, dest.Target)
				txID := tt.email.ID.String() + ":" + normalizedPeer

				mockDB.EXPECT().
					MessageGetsByTransactionID(ctx, txID, "", uint64(1)).
					Return(tt.existingByDest[i], nil)

				if len(tt.existingByDest[i]) > 0 {
					// dedup hit: no conversation lookup, no create
					continue
				}

				// GetOrCreateBySelfAndPeer locates an existing conversation
				cv := tt.conversationByDest[i]
				mockDB.EXPECT().
					ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).
					Return(cv, nil)

				mockMessage.EXPECT().
					Create(
						ctx,
						messagehandler.MessageCreateArgs{
							ID:             uuid.Nil,
							CustomerID:     tt.email.CustomerID,
							ConversationID: cv.ID,
							Direction:      message.DirectionOutgoing,
							Status:         message.StatusProgressing,
							ReferenceType:  message.ReferenceTypeEmail,
							ReferenceID:    tt.email.ID,
							TransactionID:  txID,
							Text:           tt.email.Content,
							Subject:        tt.email.Subject,
							Medias:         []media.Media{},
						},
					).
					Return(&message.Message{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}}, nil)
			}

			if err := h.EmailEventSent(ctx, tt.email); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EmailEventSent_partialFailureContinues(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &conversationHandler{
		db:             mockDB,
		messageHandler: mockMessage,
	}

	ctx := context.Background()

	e := &emmemail.Email{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			CustomerID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
		},
		Source: &commonaddress.Address{
			Type:   commonaddress.TypeEmail,
			Target: "sender@voipbin.net",
		},
		Destinations: []commonaddress.Address{
			{Type: commonaddress.TypeEmail, Target: "fail@example.com"},
			{Type: commonaddress.TypeEmail, Target: "ok@example.com"},
		},
		Subject: "Partial",
		Content: "Body",
	}

	cvA := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000098")}}
	cvOK := &conversation.Conversation{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000099")}}

	// Both destinations are looked up for dedup; neither has an existing message.
	mockDB.EXPECT().MessageGetsByTransactionID(ctx, gomock.Any(), "", uint64(1)).Return([]*message.Message{}, nil).Times(2)

	// Both destinations resolve a conversation (GetOrCreateBySelfAndPeer finds one).
	gomock.InOrder(
		mockDB.EXPECT().
			ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).
			Return(cvA, nil),
		mockDB.EXPECT().
			ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).
			Return(cvOK, nil),
	)

	// First destination's message Create fails; the loop must continue to the
	// second destination rather than aborting the whole email. Second succeeds.
	normA, _ := commonaddress.NormalizeTarget(e.Destinations[0].Type, e.Destinations[0].Target)
	txA := e.ID.String() + ":" + normA
	normOK, _ := commonaddress.NormalizeTarget(e.Destinations[1].Type, e.Destinations[1].Target)
	txOK := e.ID.String() + ":" + normOK
	gomock.InOrder(
		mockMessage.EXPECT().
			Create(ctx, messagehandler.MessageCreateArgs{
				ID:             uuid.Nil,
				CustomerID:     e.CustomerID,
				ConversationID: cvA.ID,
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusProgressing,
				ReferenceType:  message.ReferenceTypeEmail,
				ReferenceID:    e.ID,
				TransactionID:  txA,
				Text:           e.Content,
				Subject:        e.Subject,
				Medias:         []media.Media{},
			}).
			Return(nil, fmt.Errorf("boom")),
		mockMessage.EXPECT().
			Create(ctx, messagehandler.MessageCreateArgs{
				ID:             uuid.Nil,
				CustomerID:     e.CustomerID,
				ConversationID: cvOK.ID,
				Direction:      message.DirectionOutgoing,
				Status:         message.StatusProgressing,
				ReferenceType:  message.ReferenceTypeEmail,
				ReferenceID:    e.ID,
				TransactionID:  txOK,
				Text:           e.Content,
				Subject:        e.Subject,
				Medias:         []media.Media{},
			}).
			Return(&message.Message{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}}, nil),
	)

	// A partial failure is surfaced as a non-nil error (observability only),
	// but the second destination was still processed.
	if err := h.EmailEventSent(ctx, e); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_EmailEventSent_nilSource(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)
	h := &conversationHandler{
		db:             mockDB,
		messageHandler: mockMessage,
	}

	ctx := context.Background()

	// nil Source: no DB or message-handler calls expected (mc.Finish asserts this).
	e := &emmemail.Email{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			CustomerID: uuid.FromStringOrNil("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		},
		Source: nil,
		Destinations: []commonaddress.Address{
			{Type: commonaddress.TypeEmail, Target: "customer@example.com"},
		},
	}

	if err := h.EmailEventSent(ctx, e); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
