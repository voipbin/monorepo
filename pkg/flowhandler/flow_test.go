package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

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

		customerID uuid.UUID
		flowType   flow.Type
		flowName   string
		detail     string
		persist    bool
		actions    []action.Action
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flow.TypeFlow,
			"test",
			"test detail",
			true,
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
		},
		{
			"test empty",
			uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flow.TypeFlow,
			"test",
			"test detail",
			true,
			[]action.Action{},
		},
		{
			"test empty with persist false",
			uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flow.TypeFlow,
			"test",
			"test detail",
			false,
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

			_, err := h.FlowCreate(ctx, tt.customerID, tt.flowType, tt.flowName, tt.detail, tt.persist, tt.actions)
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
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &flowHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name   string
		flowID uuid.UUID

		expectRes *flow.Flow
	}{
		{
			"test normal",
			uuid.FromStringOrNil("acb2d07e-67c5-11eb-a39d-6f0133ff0559"),
			&flow.Flow{
				ID: uuid.FromStringOrNil("acb2d07e-67c5-11eb-a39d-6f0133ff0559"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().FlowDelete(gomock.Any(), tt.flowID).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), tt.flowID).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), flow.EventTypeFlowDeleted, gomock.Any())

			mockReq.EXPECT().NMV1NumberFlowDelete(ctx, tt.flowID).Return(nil)

			res, err := h.FlowDelete(context.Background(), tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
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

func TestFlowGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &flowHandler{
		db: mockDB,
	}

	type test struct {
		name       string
		customerID uuid.UUID
		token      string
		limit      uint64
	}

	tests := []test{
		{
			"test normal",
			uuid.FromStringOrNil("938cdf96-7f4c-11ec-94d3-8ba7d397d7fb"),
			"2020-10-10T03:30:17.000000",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().FlowGets(ctx, tt.customerID, tt.token, tt.limit).Return(nil, nil)

			_, err := h.FlowGets(ctx, tt.customerID, tt.token, tt.limit)
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
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &flowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name string

		id       uuid.UUID
		flowName string
		detail   string
		actions  []action.Action

		responseFlow *flow.Flow
		expectRes    *flow.Flow
	}{
		{
			"test normal",

			uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
			"changed name",
			"changed detail",
			[]action.Action{
				{
					Type: action.TypeAnswer,
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

			mockDB.EXPECT().FlowUpdate(ctx, tt.id, tt.flowName, tt.detail, gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowUpdated, tt.responseFlow)

			res, err := h.FlowUpdate(ctx, tt.id, tt.flowName, tt.detail, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestGenerateFlowForAgentCall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &flowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name string

		customerID   uuid.UUID
		confbridgeID uuid.UUID

		responseFlow *flow.Flow
		expectRes    *flow.Flow
	}{
		{
			"test normal",

			uuid.FromStringOrNil("e8d81018-8ca5-11ec-99e0-6ff2cca2a2d9"),
			uuid.FromStringOrNil("e926b54c-8ca5-11ec-84bf-036e13d83721"),

			&flow.Flow{
				ID: uuid.FromStringOrNil("4abf1d80-8ca6-11ec-b130-7b0a22a773f8"),
			},
			&flow.Flow{
				ID: uuid.FromStringOrNil("4abf1d80-8ca6-11ec-b130-7b0a22a773f8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			mockDB.EXPECT().FlowSetToCache(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(gomock.Any(), gomock.Any()).Return(tt.responseFlow, nil)

			res, err := h.generateFlowForAgentCall(ctx, tt.customerID, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
