package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func TestActionGet(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow
	}{
		{
			"test normal",
			&flow.Flow{
				ID: uuid.Must(uuid.NewV4()),
				Actions: []action.Action{
					{
						ID:   uuid.Must(uuid.NewV4()),
						Type: action.TypeEcho,
					},
					{
						ID:   uuid.Must(uuid.NewV4()),
						Type: action.TypeEcho,
					},
				},
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

			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)
			action, err := h.ActionGet(context.Background(), tt.flow.ID, tt.flow.Actions[0].ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*action, tt.flow.Actions[0]) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.flow.Actions[0], *action)
			}
		})
	}
}
