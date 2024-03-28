package flowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		flowType   flow.Type
		flowName   string
		detail     string
		persist    bool
		actions    []action.Action

		responseFlow *flow.Flow
	}{
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

			&flow.Flow{
				ID:         uuid.FromStringOrNil("8b7c353e-e6e6-11ec-af5a-e70eb001a48b"),
				CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
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

			&flow.Flow{
				ID:         uuid.FromStringOrNil("976d8e2e-e6e6-11ec-8da0-ef008343ebac"),
				CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			},
		},
		{
			"test empty with persist false",
			uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flow.TypeFlow,
			"test",
			"test detail",
			false,
			[]action.Action{},

			&flow.Flow{
				ID:         uuid.FromStringOrNil("97440572-e6e6-11ec-bcc6-73d296fdfdb7"),
				CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &flowHandler{
				util:          mockUtil,
				db:            mockDB,
				actionHandler: mockAction,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.actions, nil)

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			if tt.persist == true {
				mockDB.EXPECT().FlowCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().FlowGet(ctx, gomock.Any()).Return(tt.responseFlow, nil)
			} else {
				mockDB.EXPECT().FlowSetToCache(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().FlowGet(ctx, gomock.Any()).Return(tt.responseFlow, nil)
			}
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowCreated, tt.responseFlow)

			_, err := h.Create(ctx, tt.customerID, tt.flowType, tt.flowName, tt.detail, tt.persist, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_FlowGet(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow
	}{
		{
			"test normal",
			&flow.Flow{
				ID: uuid.FromStringOrNil("75d3c842-67c5-11eb-b8fe-0728b45d5ff1"),
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
			mockDB.EXPECT().FlowGet(ctx, tt.flow.ID).Return(tt.flow, nil)

			_, err := h.Get(ctx, tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

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

			ctx := context.Background()
			mockDB.EXPECT().FlowDelete(ctx, tt.flowID).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.flowID).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowDeleted, gomock.Any())

			res, err := h.Delete(ctx, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name    string
		token   string
		limit   uint64
		filters map[string]string
	}{
		{
			"normal",
			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"customer_id": "938cdf96-7f4c-11ec-94d3-8ba7d397d7fb",
				"deleted":     "false",
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
			mockDB.EXPECT().FlowGets(ctx, tt.token, tt.limit, tt.filters).Return(nil, nil)

			_, err := h.Gets(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Update(t *testing.T) {

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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			h := &flowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				actionHandler: mockAction,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowUpdate(ctx, tt.id, tt.flowName, tt.detail, gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowUpdated, tt.responseFlow)

			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.actions, nil)
			res, err := h.Update(ctx, tt.id, tt.flowName, tt.detail, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowUpdateActions(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		actions []action.Action

		responseFlow *flow.Flow
		expectRes    *flow.Flow
	}{
		{
			"test normal",

			uuid.FromStringOrNil("a544c079-cf19-4111-a8ac-238791c4750d"),
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("bbaa71de-5c9c-40fb-b8e7-28c331c28f73"),
					Type: action.TypeAnswer,
				},
			},

			&flow.Flow{
				ID:     uuid.FromStringOrNil("a544c079-cf19-4111-a8ac-238791c4750d"),
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("bbaa71de-5c9c-40fb-b8e7-28c331c28f73"),
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("a544c079-cf19-4111-a8ac-238791c4750d"),
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("bbaa71de-5c9c-40fb-b8e7-28c331c28f73"),
						Type: action.TypeAnswer,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			h := &flowHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				actionHandler: mockAction,
			}

			ctx := context.Background()

			mockDB.EXPECT().FlowUpdateActions(ctx, tt.id, gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowUpdated, tt.responseFlow)

			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.actions, nil)
			res, err := h.UpdateActions(ctx, tt.id, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
