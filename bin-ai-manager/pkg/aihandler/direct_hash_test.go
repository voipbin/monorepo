package aihandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_DirectHashRegenerate_ExistingDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseAI        *ai.AI
		responseDirect    *dmdirect.Direct
		responseUpdatedAI *ai.AI

		expectUpdateFields map[ai.Field]any
	}

	tests := []test{
		{
			name: "ai with existing direct_id regenerates",

			id: uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "oldhash123",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				ResourceType: dmdirect.ResourceTypeAI,
				ResourceID:   uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
				Hash:         "newhash456",
			},
			responseUpdatedAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "newhash456",
			},

			expectUpdateFields: map[ai.Field]any{
				ai.FieldDirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				ai.FieldDirectHash: "newhash456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseAI, nil)
			mockReq.EXPECT().DirectV1DirectRegenerate(ctx, tt.responseAI.DirectID).Return(tt.responseDirect, nil)
			mockDB.EXPECT().AIUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseUpdatedAI, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedAI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAI, res)
			}
		})
	}
}

func Test_DirectHashRegenerate_NoDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseAI        *ai.AI
		responseDirect    *dmdirect.Direct
		responseUpdatedAI *ai.AI

		expectUpdateFields map[ai.Field]any
	}

	tests := []test{
		{
			name: "ai with no direct_id creates new direct",

			id: uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.Nil,
				DirectHash: "",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				ResourceType: dmdirect.ResourceTypeAI,
				ResourceID:   uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
				Hash:         "createdhash789",
			},
			responseUpdatedAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				DirectHash: "createdhash789",
			},

			expectUpdateFields: map[ai.Field]any{
				ai.FieldDirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				ai.FieldDirectHash: "createdhash789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseAI, nil)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.responseAI.CustomerID, dmdirect.ResourceTypeAI, tt.id).Return(tt.responseDirect, nil)
			mockDB.EXPECT().AIUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseUpdatedAI, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedAI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedAI, res)
			}
		})
	}
}
