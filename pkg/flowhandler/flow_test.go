package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
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

		userID     uint64
		flowType   flow.Type
		flowName   string
		detail     string
		persist    bool
		webhookURI string
		actions    []action.Action
	}

	tests := []test{
		{
			"normal",

			1,
			flow.TypeFlow,
			"test",
			"test detail",
			true,
			"test.com",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
		},
		{
			"test empty",
			1,
			flow.TypeFlow,
			"test",
			"test detail",
			true,
			"test.com",
			[]action.Action{},
		},
		{
			"test empty with persist false",
			1,
			flow.TypeFlow,
			"test",
			"test detail",
			false,
			"test.com",
			[]action.Action{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.persist == true {
				mockDB.EXPECT().FlowCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(&flow.Flow{}, nil)
			} else {
				mockDB.EXPECT().FlowSetToCache(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(&flow.Flow{}, nil)
			}

			_, err := h.FlowCreate(ctx, tt.userID, tt.flowType, tt.flowName, tt.detail, tt.persist, tt.webhookURI, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
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
			&flow.Flow{
				ID: uuid.FromStringOrNil("75d3c842-67c5-11eb-b8fe-0728b45d5ff1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(tt.flow, nil)

			_, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestFlowDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &flowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	type test struct {
		name   string
		flowID uuid.UUID
	}

	tests := []test{
		{
			"test normal",
			uuid.FromStringOrNil("acb2d07e-67c5-11eb-a39d-6f0133ff0559"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().FlowDelete(gomock.Any(), tt.flowID).Return(nil)
			mockReq.EXPECT().NMV1NumberFlowDelete(ctx, tt.flowID).Return(nil)

			if err := h.FlowDelete(context.Background(), tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// func TestFlowCreatePersistTrue(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockDB := dbhandler.NewMockDBHandler(mc)

// 	h := &flowHandler{
// 		db: mockDB,
// 	}

// 	type test struct {
// 		name string
// 		flow *flow.Flow
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&flow.Flow{
// 				ID:      uuid.FromStringOrNil("8bf11004-ef06-11ea-91ed-0ba639a6618b"),
// 				Persist: true,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := context.Background()

// 			mockDB.EXPECT().FlowCreate(gomock.Any(), gomock.Any()).Return(nil)
// 			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(tt.flow, nil)

// 			_, err := h.FlowCreate(ctx, tt.flow)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

// func TestFlowCreatePersistFalse(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockDB := dbhandler.NewMockDBHandler(mc)

// 	h := &flowHandler{
// 		db: mockDB,
// 	}

// 	type test struct {
// 		name string
// 		flow *flow.Flow
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&flow.Flow{
// 				Persist: false,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := context.Background()

// 			mockDB.EXPECT().FlowSetToCache(gomock.Any(), gomock.Any()).Return(nil)
// 			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(tt.flow, nil)

// 			_, err := h.FlowCreate(ctx, tt.flow)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

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

			_, err := h.FlowGetsByUserID(ctx, tt.userID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
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
		updateFlow   *flow.Flow
		responseFlow *flow.Flow
		expectRes    *flow.Flow
	}

	tests := []test{
		{
			"test normal",
			&flow.Flow{
				ID:     uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
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

			mockDB.EXPECT().FlowUpdate(ctx, tt.updateFlow).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.updateFlow.ID).Return(tt.responseFlow, nil)
			res, err := h.FlowUpdate(ctx, tt.updateFlow)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
