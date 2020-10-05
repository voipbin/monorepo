package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

func TestActionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name string
		flow *flow.Flow
	}

	tests := []test{
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

func TestActionPatchGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name   string
		act    *action.Action
		callID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			&action.Action{
				ID:     uuid.FromStringOrNil("6e2a0cee-fba2-11ea-a469-a350f2dad844"),
				Option: []byte(`{"event_url": "https://webhook.site/e47c9b40-662c-4d20-a288-6777360fa211"}`),
			},
			uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := h.actionPatchGet(tt.act, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}

}
