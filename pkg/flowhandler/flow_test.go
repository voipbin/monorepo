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

func TestFlowCreate(t *testing.T) {
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
			&flow.Flow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(&flow.Flow{}, nil)

			h.FlowCreate(context.Background(), &flow.Flow{}, true)

		})
	}
}

func TestFlowGet(t *testing.T) {
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
			&flow.Flow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(&flow.Flow{}, nil)

			h.FlowGet(context.Background(), uuid.Must(uuid.NewV4()))
		})
	}
}

func TestFlowCreatePersistTrue(t *testing.T) {
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
			"normal",
			&flow.Flow{
				ID: uuid.FromStringOrNil("8bf11004-ef06-11ea-91ed-0ba639a6618b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().FlowCreate(gomock.Any(), tt.flow).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)

			h.FlowCreate(ctx, tt.flow, true)
		})
	}
}

func TestFlowCreatePersistFalse(t *testing.T) {
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
			"normal",
			&flow.Flow{
				ID: uuid.FromStringOrNil("ebb1b7a0-ef06-11ea-900b-d7f31a9b7baa"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().FlowSetToCache(gomock.Any(), tt.flow).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)

			h.FlowCreate(ctx, tt.flow, false)
		})
	}
}

func TestFlowGetByUserID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name   string
		userID uint64
		token  string
		limit  uint64
	}

	tests := []test{
		{
			"test normal",
			1,
			"2020-10-10T03:30:17.000000",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().FlowGetsByUserID(ctx, tt.userID, tt.token, tt.limit).Return(nil, nil)

			h.FlowGetsByUserID(ctx, tt.userID, tt.token, tt.limit)
		})
	}
}

func TestFlowUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name         string
		flowID       uuid.UUID
		updateFlow   *flow.Flow
		responseFlow *flow.Flow
		expectRes    *flow.Flow
	}

	tests := []test{
		{
			"test normal",
			uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
			&flow.Flow{
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("445ad416-676d-11eb-bca9-1f9e07621368"),
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("445ad416-676d-11eb-bca9-1f9e07621368"),
						Type: action.TypeAnswer,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			mockDB.EXPECT().FlowUpdate(ctx, tt.flowID, tt.updateFlow).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			res, err := h.FlowUpdate(ctx, tt.flowID, tt.updateFlow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
