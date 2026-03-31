package flowhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_DirectHashRegenerate_FlowNotFound(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			name: "flow not found returns error",

			id: uuid.FromStringOrNil("f1f1f1f1-1111-1111-1111-111111111111"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &flowHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(nil, fmt.Errorf("flow not found"))

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_DirectHashRegenerate_ExistingDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseFlow        *flow.Flow
		responseDirect      *dmdirect.Direct
		responseUpdatedFlow *flow.Flow

		expectUpdateFields map[flow.Field]any
	}

	tests := []test{
		{
			name: "flow with existing direct_id regenerates",

			id: uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),

			responseFlow: &flow.Flow{
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
				ResourceType: "flow",
				ResourceID:   uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
				Hash:         "newhash456",
			},
			responseUpdatedFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111"),
					CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
				},
				DirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				DirectHash: "newhash456",
			},

			expectUpdateFields: map[flow.Field]any{
				flow.FieldDirectID:   uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
				flow.FieldDirectHash: "newhash456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &flowHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockReq.EXPECT().DirectV1DirectRegenerate(ctx, tt.responseFlow.DirectID).Return(tt.responseDirect, nil)
			mockDB.EXPECT().FlowUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseUpdatedFlow, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedFlow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedFlow, res)
			}
		})
	}
}

func Test_DirectHashRegenerate_NoDirect(t *testing.T) {

	type test struct {
		name string

		id uuid.UUID

		responseFlow        *flow.Flow
		responseDirect      *dmdirect.Direct
		responseUpdatedFlow *flow.Flow

		expectUpdateFields map[flow.Field]any
	}

	tests := []test{
		{
			name: "flow with no direct_id creates new direct",

			id: uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),

			responseFlow: &flow.Flow{
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
				ResourceType: "flow",
				ResourceID:   uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
				Hash:         "createdhash789",
			},
			responseUpdatedFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222"),
					CustomerID: uuid.FromStringOrNil("c2c2c2c2-2222-2222-2222-222222222222"),
				},
				DirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				DirectHash: "createdhash789",
			},

			expectUpdateFields: map[flow.Field]any{
				flow.FieldDirectID:   uuid.FromStringOrNil("d2d2d2d2-2222-2222-2222-222222222222"),
				flow.FieldDirectHash: "createdhash789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &flowHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, tt.responseFlow.CustomerID, "flow", tt.id).Return(tt.responseDirect, nil)
			mockDB.EXPECT().FlowUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseUpdatedFlow, nil)

			res, err := h.DirectHashRegenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseUpdatedFlow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseUpdatedFlow, res)
			}
		})
	}
}
