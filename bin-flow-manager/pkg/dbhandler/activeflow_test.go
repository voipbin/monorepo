package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

func Test_ActiveflowCreate(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseCurTime string

		expectedRes *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2752907c-ace3-11ec-8fa5-b7c9abbb778b"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},
				FlowID: uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),
				ReferenceActiveflowID: uuid.FromStringOrNil("c5ffef10-07d2-11f0-901d-dfa7036e5fa5"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("99e1dab2-d588-11ec-abff-93cc8e1d3e49"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("99e1dab2-d588-11ec-abff-93cc8e1d3e49"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  uuid.FromStringOrNil("9852ca58-d588-11ec-93cb-d7c113ec56d7"),
				ForwardActionID: uuid.FromStringOrNil("99be3620-d588-11ec-9239-439980b8dcd2"),

				ExecuteCount: 3,
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9a1524d0-d588-11ec-9bdc-73a17f559ad8"),
						Type: action.TypeAnswer,
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2752907c-ace3-11ec-8fa5-b7c9abbb778b"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},
				FlowID: uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),
				ReferenceActiveflowID: uuid.FromStringOrNil("c5ffef10-07d2-11f0-901d-dfa7036e5fa5"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("99e1dab2-d588-11ec-abff-93cc8e1d3e49"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("99e1dab2-d588-11ec-abff-93cc8e1d3e49"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  uuid.FromStringOrNil("9852ca58-d588-11ec-93cb-d7c113ec56d7"),
				ForwardActionID: uuid.FromStringOrNil("99be3620-d588-11ec-9239-439980b8dcd2"),

				ExecuteCount: 3,
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9a1524d0-d588-11ec-9bdc-73a17f559ad8"),
						Type: action.TypeAnswer,
					},
				},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowCreate(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowGet(gomock.Any(), tt.activeflow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.ActiveflowGet(ctx, tt.activeflow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_ActiveflowUpdate(t *testing.T) {

	tests := []struct {
		name string

		activeflow       *activeflow.Activeflow
		updateActiveflow *activeflow.Activeflow

		responseCurTime string

		expectedRes *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b55d582-ace6-11ec-a6de-b7dda3562854"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},
				FlowID: uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("a5567ece-d589-11ec-a7ed-f7c002ca2172"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				ExecuteCount:    0,
				ExecutedActions: []action.Action{},
			},
			updateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b55d582-ace6-11ec-a6de-b7dda3562854"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("a5567ece-d589-11ec-a7ed-f7c002ca2172"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("a5567ece-d589-11ec-a7ed-f7c002ca2172"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				ExecuteCount: 1,
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
						Type: action.TypeAnswer,
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b55d582-ace6-11ec-a6de-b7dda3562854"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},
				FlowID: uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("a5567ece-d589-11ec-a7ed-f7c002ca2172"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("a5567ece-d589-11ec-a7ed-f7c002ca2172"),
					Type: action.TypeAnswer,
				},

				ForwardStackID:  stack.IDEmpty,
				ForwardActionID: action.IDEmpty,

				ExecuteCount: 1,
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
						Type: action.TypeAnswer,
					},
				},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowCreate(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowUpdate(ctx, tt.updateActiveflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowGet(gomock.Any(), tt.activeflow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.ActiveflowGet(ctx, tt.activeflow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_ActiveflowDelete(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseCurTime string

		expectedRes *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("56d3aa20-aced-11ec-bcec-e3508fe4f7e1"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},
				FlowID: uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("56fed452-aced-11ec-8281-73c867112f40"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("7b2e5e1c-ace6-11ec-b3e3-c71fd637fde9"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("791c55d4-ace6-11ec-9fd6-032afea377bc"),
					Type: action.TypeAnswer,
				},
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("7b08f758-ace6-11ec-a13c-5b27004a9376"),
				ExecuteCount:    1,
				ExecutedActions: []action.Action{},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("56d3aa20-aced-11ec-bcec-e3508fe4f7e1"),
					CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				},
				FlowID: uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("56fed452-aced-11ec-8281-73c867112f40"),

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("7b2e5e1c-ace6-11ec-b3e3-c71fd637fde9"),
								Type: action.TypeAnswer,
							},
						},
					},
				},

				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("791c55d4-ace6-11ec-9fd6-032afea377bc"),
					Type: action.TypeAnswer,
				},
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("7b08f758-ace6-11ec-a13c-5b27004a9376"),
				ExecuteCount:    1,
				ExecutedActions: []action.Action{},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ActiveflowSet(ctx, gomock.Any())
			if err := h.ActiveflowCreate(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ActiveflowSet(ctx, gomock.Any())
			if err := h.ActiveflowDelete(ctx, tt.activeflow.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowGet(ctx, tt.activeflow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ActiveflowSet(ctx, gomock.Any()).Return(nil)
			res, err := h.ActiveflowGet(ctx, tt.activeflow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_ActiveflowGets(t *testing.T) {

	tests := []struct {
		name        string
		activeflows []activeflow.Activeflow

		size    uint64
		filters map[string]string

		responseCurTime string

		expectedRes []*activeflow.Activeflow
	}{
		{
			name: "have no actions",
			activeflows: []activeflow.Activeflow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b9c89d28-ecda-11ee-a4c3-3f9069ec91c9"),
						CustomerID: uuid.FromStringOrNil("c3419d78-ecda-11ee-96fd-276b944569e9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ba4c00d2-ecda-11ee-9b4e-efecfed060d2"),
						CustomerID: uuid.FromStringOrNil("c3419d78-ecda-11ee-96fd-276b944569e9"),
					},
				},
			},

			size: 10,
			filters: map[string]string{
				"customer_id": "c3419d78-ecda-11ee-96fd-276b944569e9",
				"deleted":     "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: []*activeflow.Activeflow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b9c89d28-ecda-11ee-a4c3-3f9069ec91c9"),
						CustomerID: uuid.FromStringOrNil("c3419d78-ecda-11ee-96fd-276b944569e9"),
					},
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ba4c00d2-ecda-11ee-9b4e-efecfed060d2"),
						CustomerID: uuid.FromStringOrNil("c3419d78-ecda-11ee-96fd-276b944569e9"),
					},
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			for _, activeflow := range tt.activeflows {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
				if err := h.ActiveflowCreate(ctx, &activeflow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.ActiveflowGets(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes[0], res[0])
			}
		})
	}
}
