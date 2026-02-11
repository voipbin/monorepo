package flowhandler

import (
	"context"
	"reflect"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID       uuid.UUID
		flowType         flow.Type
		flowName         string
		detail           string
		persist          bool
		actions          []action.Action
		onCompleteFlowID uuid.UUID

		responseUUID uuid.UUID
		responseFlow *flow.Flow
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flowType:   flow.TypeFlow,
			flowName:   "test",
			detail:     "test detail",
			persist:    true,
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
			onCompleteFlowID: uuid.FromStringOrNil("9ccfb956-ce18-11f0-bdeb-af04faf83ec2"),

			responseUUID: uuid.FromStringOrNil("a29bcd2e-0295-11f0-a03b-bf8d2fff2101"),
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8b7c353e-e6e6-11ec-af5a-e70eb001a48b"),
					CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
				},
			},
		},
		{
			name: "test empty",

			customerID:       uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flowType:         flow.TypeFlow,
			flowName:         "test",
			detail:           "test detail",
			persist:          true,
			actions:          []action.Action{},
			onCompleteFlowID: uuid.Nil,

			responseUUID: uuid.FromStringOrNil("a2c051d0-0295-11f0-897c-0ffe1f3c6359"),
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("976d8e2e-e6e6-11ec-8da0-ef008343ebac"),
					CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
				},
			},
		},
		{
			name: "test empty with persist false",

			customerID:       uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			flowType:         flow.TypeFlow,
			flowName:         "test",
			detail:           "test detail",
			persist:          false,
			actions:          []action.Action{},
			onCompleteFlowID: uuid.Nil,

			responseUUID: uuid.FromStringOrNil("a2e4a45e-0295-11f0-b0d2-9b991bf4aa3d"),
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("97440572-e6e6-11ec-bcc6-73d296fdfdb7"),
					CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
				},
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
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &flowHandler{
				util:          mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				actionHandler: mockAction,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockReq.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, tt.customerID, bmaccount.ResourceTypeFlow).Return(true, nil)
			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.actions, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			if tt.persist == true {
				mockDB.EXPECT().FlowCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().FlowGet(ctx, gomock.Any()).Return(tt.responseFlow, nil)
			} else {
				mockDB.EXPECT().FlowSetToCache(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().FlowGet(ctx, gomock.Any()).Return(tt.responseFlow, nil)
			}
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowCreated, tt.responseFlow)

			res, err := h.Create(ctx, tt.customerID, tt.flowType, tt.flowName, tt.detail, tt.persist, tt.actions, tt.onCompleteFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFlow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFlow, res)
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
			name: "test normal",
			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("75d3c842-67c5-11eb-b8fe-0728b45d5ff1"),
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

		responseRes *flow.Flow
	}{
		{
			name: "test normal",

			flowID: uuid.FromStringOrNil("acb2d07e-67c5-11eb-a39d-6f0133ff0559"),
			responseRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("acb2d07e-67c5-11eb-a39d-6f0133ff0559"),
				},
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
			mockDB.EXPECT().FlowGet(ctx, tt.flowID).Return(tt.responseRes, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowDeleted, tt.responseRes)

			res, err := h.Delete(ctx, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseRes, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name    string
		token   string
		limit   uint64
		filters map[flow.Field]any

		responseFlows []*flow.Flow
	}{
		{
			name: "normal",

			token: "2020-10-10T03:30:17.000000Z",
			limit: 10,
			filters: map[flow.Field]any{
				flow.FieldCustomerID: uuid.FromStringOrNil("938cdf96-7f4c-11ec-94d3-8ba7d397d7fb"),
				flow.FieldDeleted:    false,
			},

			responseFlows: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2ce31ae8-028a-11f0-bc11-6f2efbd51bb2"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2d22a9b0-028a-11f0-9be2-c794607c2866"),
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

			ctx := context.Background()
			mockDB.EXPECT().FlowList(ctx, tt.token, tt.limit, tt.filters).Return(tt.responseFlows, nil)

			res, err := h.List(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFlows) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFlows, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id               uuid.UUID
		flowName         string
		detail           string
		actions          []action.Action
		onCompleteFlowID uuid.UUID

		responseFlow       *flow.Flow
		expectUpdateFiedls map[flow.Field]any
		expectedRes        *flow.Flow
	}{
		{
			name: "test normal",

			id:       uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
			flowName: "changed name",
			detail:   "changed detail",
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
			onCompleteFlowID: uuid.FromStringOrNil("9d74ce00-ce18-11f0-ba12-37c102015df6"),

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
				},
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("445ad416-676d-11eb-bca9-1f9e07621368"),
						Type: action.TypeAnswer,
					},
				},
			},
			expectUpdateFiedls: map[flow.Field]any{
				flow.FieldName:   "changed name",
				flow.FieldDetail: "changed detail",
				flow.FieldActions: []action.Action{
					{
						Type: action.TypeAnswer,
					},
				},
				flow.FieldOnCompleteFlowID: uuid.FromStringOrNil("9d74ce00-ce18-11f0-ba12-37c102015df6"),
			},
			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("728c58a6-676c-11eb-945b-e7ade6fd0b8d"),
				},
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

			mockDB.EXPECT().FlowUpdate(ctx, tt.id, tt.expectUpdateFiedls).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowUpdated, tt.responseFlow)

			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.actions, nil)
			res, err := h.Update(ctx, tt.id, tt.flowName, tt.detail, tt.actions, tt.onCompleteFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectedRes, res)
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

		expectedUpdateFields map[flow.Field]any
		expectedRes          *flow.Flow
	}{
		{
			name: "test normal",

			id: uuid.FromStringOrNil("a544c079-cf19-4111-a8ac-238791c4750d"),
			actions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("bbaa71de-5c9c-40fb-b8e7-28c331c28f73"),
					Type: action.TypeAnswer,
				},
			},

			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a544c079-cf19-4111-a8ac-238791c4750d"),
				},
				Name:   "changed name",
				Detail: "changed detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("bbaa71de-5c9c-40fb-b8e7-28c331c28f73"),
						Type: action.TypeAnswer,
					},
				},
			},

			expectedUpdateFields: map[flow.Field]any{
				flow.FieldActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("bbaa71de-5c9c-40fb-b8e7-28c331c28f73"),
						Type: action.TypeAnswer,
					},
				},
			},
			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a544c079-cf19-4111-a8ac-238791c4750d"),
				},
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

			mockDB.EXPECT().FlowUpdate(ctx, tt.id, tt.expectedUpdateFields).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, tt.id).Return(tt.responseFlow, nil)
			mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowUpdated, tt.responseFlow)

			mockAction.EXPECT().GenerateFlowActions(ctx, tt.actions).Return(tt.actions, nil)
			res, err := h.UpdateActions(ctx, tt.id, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectedRes, res)
			}
		})
	}
}
