package contacthandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func resTimePtr(t time.Time) *time.Time { return &t }

func Test_ResolutionCreate(t *testing.T) {
	customerID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")
	interactionID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000003")
	agentID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000004")
	resID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000005")
	now := resTimePtr(time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC))

	validInteraction := &interaction.Interaction{
		ID:         interactionID,
		CustomerID: customerID,
	}
	validContact := &contact.Contact{
		Identity: commonidentity.Identity{
			ID:         contactID,
			CustomerID: customerID,
		},
	}

	tests := []struct {
		name string

		customerID     uuid.UUID
		contactID      uuid.UUID
		interactionID  uuid.UUID
		resolutionType string
		resolvedByType string
		resolvedByID   uuid.UUID

		mockInteraction    *interaction.Interaction
		mockInteractionErr error
		mockContact        *contact.Contact
		mockContactErr     error
		mockExisting       []*resolution.Resolution
		expectErr          bool
	}{
		{
			name:           "normal",
			customerID:     customerID,
			contactID:      contactID,
			interactionID:  interactionID,
			resolutionType: resolution.ResolutionTypePositive,
			resolvedByType: resolution.ResolvedByTypeAgent,
			resolvedByID:   agentID,

			mockInteraction: validInteraction,
			mockContact:     validContact,
			mockExisting:    []*resolution.Resolution{},
			expectErr:       false,
		},
		{
			name:           "interaction not found",
			customerID:     customerID,
			contactID:      contactID,
			interactionID:  interactionID,
			resolutionType: resolution.ResolutionTypePositive,
			resolvedByType: resolution.ResolvedByTypeAgent,
			resolvedByID:   agentID,

			mockInteraction:    nil,
			mockInteractionErr: dbhandler.ErrNotFound,
			expectErr:          true,
		},
		{
			name:           "wrong interaction customer",
			customerID:     customerID,
			contactID:      contactID,
			interactionID:  interactionID,
			resolutionType: resolution.ResolutionTypePositive,
			resolvedByType: resolution.ResolvedByTypeAgent,
			resolvedByID:   agentID,

			mockInteraction: &interaction.Interaction{
				ID:         interactionID,
				CustomerID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001"), // different customer
			},
			expectErr: true,
		},
		{
			name:           "contact not found",
			customerID:     customerID,
			contactID:      contactID,
			interactionID:  interactionID,
			resolutionType: resolution.ResolutionTypePositive,
			resolvedByType: resolution.ResolvedByTypeAgent,
			resolvedByID:   agentID,

			mockInteraction: validInteraction,
			mockContact:     nil,
			mockContactErr:  dbhandler.ErrNotFound,
			expectErr:       true,
		},
		{
			name:           "cross-tenant contactID",
			customerID:     customerID,
			contactID:      contactID,
			interactionID:  interactionID,
			resolutionType: resolution.ResolutionTypePositive,
			resolvedByType: resolution.ResolvedByTypeAgent,
			resolvedByID:   agentID,

			mockInteraction: validInteraction,
			mockContact: &contact.Contact{
				Identity: commonidentity.Identity{
					ID:         contactID,
					CustomerID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001"), // different customer
				},
			},
			expectErr: true,
		},
		{
			name:           "duplicate resolution same type",
			customerID:     customerID,
			contactID:      contactID,
			interactionID:  interactionID,
			resolutionType: resolution.ResolutionTypePositive,
			resolvedByType: resolution.ResolvedByTypeAgent,
			resolvedByID:   agentID,

			mockInteraction: validInteraction,
			mockContact:     validContact,
			mockExisting: []*resolution.Resolution{
				{
					ContactID:      contactID,
					InteractionID:  interactionID,
					ResolutionType: resolution.ResolutionTypePositive,
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &contactHandler{
				utilHandler: mockUtil,
				db:          mockDB,
			}
			ctx := context.Background()

			// Always expect InteractionGet.
			mockDB.EXPECT().InteractionGet(ctx, tt.interactionID).Return(tt.mockInteraction, tt.mockInteractionErr)

			// ContactGet only called if interaction ownership passes.
			if tt.mockInteractionErr == nil && tt.mockInteraction != nil && tt.mockInteraction.CustomerID == tt.customerID {
				mockDB.EXPECT().ContactGet(ctx, tt.contactID).Return(tt.mockContact, tt.mockContactErr)
			}

			// ResolutionListByInteraction + create only on full success path.
			if !tt.expectErr && tt.mockExisting != nil {
				mockDB.EXPECT().ResolutionListByInteraction(ctx, tt.customerID, tt.interactionID).Return(tt.mockExisting, nil)
				mockUtil.EXPECT().UUIDCreate().Return(resID)
				mockUtil.EXPECT().TimeNow().Return(now)
				mockDB.EXPECT().ResolutionCreate(ctx, gomock.Any()).Return(nil)
			}

			// Also set up ResolutionListByInteraction for duplicate check on duplicate case.
			if tt.expectErr && tt.mockExisting != nil && tt.mockContactErr == nil && tt.mockContact != nil && tt.mockContact.CustomerID == tt.customerID {
				mockDB.EXPECT().ResolutionListByInteraction(ctx, tt.customerID, tt.interactionID).Return(tt.mockExisting, nil)
			}

			_, err := h.ResolutionCreate(ctx, tt.customerID, tt.contactID, tt.interactionID, tt.resolutionType, tt.resolvedByType, tt.resolvedByID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
