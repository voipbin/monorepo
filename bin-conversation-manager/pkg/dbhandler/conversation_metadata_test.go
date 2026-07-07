package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

// Test_ConversationCreate_And_ConversationGet_Metadata verifies that
// Conversation.Metadata.ContactCaseID correctly round-trips through the
// DB (marshal on create, unmarshal on get), and that a Conversation with
// no Metadata set round-trips to a zero-value Metadata (ContactCaseID nil),
// not an error. This is Phase 1 Task 1.3 of the contact-case-management
// implementation plan.
func Test_ConversationCreate_And_ConversationGet_Metadata(t *testing.T) {
	caseID := uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000002")

	tests := []struct {
		name         string
		conversation *conversation.Conversation

		responseCurTime *time.Time
		expectRes       *conversation.Conversation
	}{
		{
			name: "metadata with contact_case_id set",
			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000003"),
					CustomerID: uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000004"),
				},
				Type: conversation.TypeMessage,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15551110000",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15552220000",
				},
				Metadata: &conversation.Metadata{ContactCaseID: &caseID},
			},

			responseCurTime: func() *time.Time { t := time.Date(2026, 7, 7, 9, 58, 0, 0, time.UTC); return &t }(),
			expectRes: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000003"),
					CustomerID: uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000004"),
				},
				Type: conversation.TypeMessage,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15551110000",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15552220000",
				},
				Metadata: &conversation.Metadata{ContactCaseID: &caseID},
				TMCreate: func() *time.Time { t := time.Date(2026, 7, 7, 9, 58, 0, 0, time.UTC); return &t }(),
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
		{
			name: "no metadata set (zero value)",
			conversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000005"),
					CustomerID: uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000004"),
				},
				Type: conversation.TypeMessage,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15551110001",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15552220001",
				},
			},

			responseCurTime: func() *time.Time { t := time.Date(2026, 7, 7, 9, 58, 0, 0, time.UTC); return &t }(),
			expectRes: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000005"),
					CustomerID: uuid.FromStringOrNil("f1b2c3d4-0002-0002-0002-000000000004"),
				},
				Type: conversation.TypeMessage,
				Self: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15551110001",
				},
				Peer: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+15552220001",
				},
				Metadata: nil,
				TMCreate: func() *time.Time { t := time.Date(2026, 7, 7, 9, 58, 0, 0, time.UTC); return &t }(),
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ConversationSet(ctx, gomock.Any())
			if err := h.ConversationCreate(ctx, tt.conversation); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// bypass cache to force a real DB read
			res, err := h.conversationGetFromDB(ctx, tt.conversation.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.expectRes, res)
			}
		})
	}
}
