package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func TestActionGet(t *testing.T) {

	tests := []struct {
		name string

		flowID   uuid.UUID
		actionID uuid.UUID

		responseFlow *flow.Flow
		expectedRes  *action.Action
	}{
		{
			name:     "test normal",
			flowID:   uuid.FromStringOrNil("3e25c306-0289-11f0-a190-f7382d5ab674"),
			actionID: uuid.FromStringOrNil("e377d304-0288-11f0-86a6-8f3c5f362761"),

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3e25c306-0289-11f0-a190-f7382d5ab674"),
				},
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("e34e40ac-0288-11f0-b112-83c58d92bece"),
					},
					{
						ID: uuid.FromStringOrNil("e377d304-0288-11f0-86a6-8f3c5f362761"),
					},
					{
						ID: uuid.FromStringOrNil("e3a0e564-0288-11f0-a3b9-1badd82bba32"),
					},
				},
			},
			expectedRes: &action.Action{
				ID: uuid.FromStringOrNil("e377d304-0288-11f0-86a6-8f3c5f362761"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &flowHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().FlowGet(ctx, tt.flowID.Return(tt.responseFlow, nil)
			res, err := h.ActionGet(ctx, tt.flowID, tt.actionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}
