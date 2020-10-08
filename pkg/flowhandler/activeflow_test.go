package flowhandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/activeflow"
)

func TestActiveFlowUpdateCurrentAction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name   string
		callID uuid.UUID
		act    *action.Action
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("f594ebd8-06ae-11eb-9bca-5757b3876041"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f916a6a2-06ae-11eb-a239-53802c6fbb36"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&activeflow.ActiveFlow{}, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.activeFlowUpdateCurrentAction(ctx, tt.callID, tt.act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestActiveFlowNextActionGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name         string
		callID       uuid.UUID
		actionID     uuid.UUID
		af           activeflow.ActiveFlow
		expectAction action.Action
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("0d276266-0737-11eb-808f-8f2856d44e29"),
			uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
			activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
					Type: action.TypeAnswer,
				},
				Actions: []action.Action{
					action.Action{
						ID:   uuid.FromStringOrNil("05e2c40a-0737-11eb-9134-5f9b578a4179"),
						Type: action.TypeAnswer,
					},
					action.Action{
						ID:   uuid.FromStringOrNil("c9fffcf4-0737-11eb-a28f-2bc0bae5eeaf"),
						Type: action.TypeAnswer,
					},
				},
			},
			action.Action{
				ID:   uuid.FromStringOrNil("c9fffcf4-0737-11eb-a28f-2bc0bae5eeaf"),
				Type: action.TypeAnswer,
			},
		},
		{
			"empty actions",
			uuid.FromStringOrNil("085f48fc-08a4-11eb-8ef3-675e25cbc25c"),
			action.IDStart,
			activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				Actions: []action.Action{},
			},
			action.Action{
				ID:   action.IDFinish,
				Type: action.TypeHangup,
			},
		},
		{
			"current id start",
			uuid.FromStringOrNil("950c810c-08a4-11eb-af93-93115c7f9c55"),
			action.IDStart,
			activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				Actions: []action.Action{
					action.Action{
						ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
						Type: action.TypeAnswer,
					},
					action.Action{
						ID:   uuid.FromStringOrNil("a9b365ee-08a4-11eb-87c5-e7b9e9ea9de3"),
						Type: action.TypeAnswer,
					},
				},
			},
			action.Action{
				ID:   uuid.FromStringOrNil("97f96f9c-08a4-11eb-8ea0-57d38a96eca3"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)
			mockDB.EXPECT().ActiveFlowGet(gomock.Any(), tt.callID).Return(&tt.af, nil)

			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)
			act, err := h.ActiveFlowNextActionGet(ctx, tt.callID, tt.actionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if act.ID != tt.expectAction.ID || act.Type != tt.expectAction.Type {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, act)
			}
		})
	}
}

func TestCreateActionHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name string
	}

	tests := []test{
		{
			"normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := *h.CreateActionHangup()

			marString, err := json.Marshal(res)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			var act action.Action
			if err := json.Unmarshal(marString, &act); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			var opt action.OptionHangup
			if err := json.Unmarshal(act.Option, &opt); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
