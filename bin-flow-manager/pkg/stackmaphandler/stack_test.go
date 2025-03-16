package stackmaphandler

import (
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
	"reflect"
	"sort"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CreateStack(t *testing.T) {

	tests := []struct {
		name string

		stackID        uuid.UUID
		actions        []action.Action
		returnStackID  uuid.UUID
		returnActionID uuid.UUID

		responseUUID uuid.UUID

		expectedRes *stack.Stack
	}{
		{
			name: "normal",

			stackID: uuid.FromStringOrNil("e2d7bba0-fd6d-11ef-983e-9fec845d9958"),
			actions: []action.Action{
				{
					ID: uuid.FromStringOrNil("e2f2b25c-fd6d-11ef-a462-d73cddea8db0"),
				},
				{
					ID: uuid.FromStringOrNil("e31c33de-fd6d-11ef-8119-9bee28a1440d"),
				},
			},
			returnStackID:  uuid.FromStringOrNil("e34ab358-fd6d-11ef-9742-177d3a238163"),
			returnActionID: uuid.FromStringOrNil("e3736794-fd6d-11ef-a9b0-3701760a71d6"),

			expectedRes: &stack.Stack{
				ID: uuid.FromStringOrNil("e2d7bba0-fd6d-11ef-983e-9fec845d9958"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("e2f2b25c-fd6d-11ef-a462-d73cddea8db0"),
					},
					{
						ID: uuid.FromStringOrNil("e31c33de-fd6d-11ef-8119-9bee28a1440d"),
					},
				},
				ReturnStackID:  uuid.FromStringOrNil("e34ab358-fd6d-11ef-9742-177d3a238163"),
				ReturnActionID: uuid.FromStringOrNil("e3736794-fd6d-11ef-a9b0-3701760a71d6"),
			},
		},
		{
			name: "stack id is nil",

			stackID: uuid.Nil,
			actions: []action.Action{
				{
					ID: uuid.FromStringOrNil("c9bb3880-fd6e-11ef-a2cc-ff5b3670cbad"),
				},
				{
					ID: uuid.FromStringOrNil("c9e40526-fd6e-11ef-afbc-d343037a71b5"),
				},
			},
			returnStackID:  uuid.FromStringOrNil("ca08b650-fd6e-11ef-94af-1f7127f822ab"),
			returnActionID: uuid.FromStringOrNil("ca2d6bf8-fd6e-11ef-9add-4fe642b56405"),

			responseUUID: uuid.FromStringOrNil("ca521ab6-fd6e-11ef-aa1f-47246bdfa4e8"),
			expectedRes: &stack.Stack{
				ID: uuid.FromStringOrNil("ca521ab6-fd6e-11ef-aa1f-47246bdfa4e8"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("c9bb3880-fd6e-11ef-a2cc-ff5b3670cbad"),
					},
					{
						ID: uuid.FromStringOrNil("c9e40526-fd6e-11ef-afbc-d343037a71b5"),
					},
				},
				ReturnStackID:  uuid.FromStringOrNil("ca08b650-fd6e-11ef-94af-1f7127f822ab"),
				ReturnActionID: uuid.FromStringOrNil("ca2d6bf8-fd6e-11ef-9add-4fe642b56405"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &stackHandler{
				utilHandler: mockUtil,
			}

			if tt.responseUUID != uuid.Nil {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			}

			res := h.CreateStack(tt.stackID, tt.actions, tt.returnStackID, tt.returnActionID)
			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_deleteStack(t *testing.T) {

	tests := []struct {
		name string

		stackmap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectedRes map[uuid.UUID]*stack.Stack
	}{
		{
			name: "normal",

			stackmap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
				},
				uuid.FromStringOrNil("e2d7bba0-fd6d-11ef-983e-9fec845d9958"): {
					ID: uuid.FromStringOrNil("9c8a6682-fd6f-11ef-bc3d-23f8b2fdcb0b"),
				},
			},
			stackID: uuid.FromStringOrNil("e2d7bba0-fd6d-11ef-983e-9fec845d9958"),

			expectedRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
				},
			},
		},
		{
			name: "delete stack id is main stack id",

			stackmap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
				},
			},
			stackID: stack.IDMain,

			expectedRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &stackHandler{
				utilHandler: mockUtil,
			}

			h.DeleteStack(tt.stackmap, tt.stackID)
			if reflect.DeepEqual(tt.stackmap, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, tt.stackmap)
			}
		})
	}
}

func Test_GetStack(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectedRes *stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("3d983a34-d3b3-11ec-bf41-17fd979cd40f"): {
					ID: uuid.FromStringOrNil("3d983a34-d3b3-11ec-bf41-17fd979cd40f"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("60271cc4-d3b2-11ec-9885-57b3f9783b8b"),
					ReturnActionID: uuid.FromStringOrNil("6050d6e0-d3b2-11ec-baa1-ebbde6270e56"),
				},
			},
			stackID: uuid.FromStringOrNil("3d983a34-d3b3-11ec-bf41-17fd979cd40f"),

			expectedRes: &stack.Stack{
				ID: uuid.FromStringOrNil("3d983a34-d3b3-11ec-bf41-17fd979cd40f"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
						Type: action.TypeAnswer,
					},
				},
				ReturnStackID:  uuid.FromStringOrNil("60271cc4-d3b2-11ec-9885-57b3f9783b8b"),
				ReturnActionID: uuid.FromStringOrNil("6050d6e0-d3b2-11ec-baa1-ebbde6270e56"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res, err := h.GetStack(tt.stackMap, tt.stackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
func Test_PushStackByActions(t *testing.T) {

	tests := []struct {
		name string

		stackMap        map[uuid.UUID]*stack.Stack
		stackID         uuid.UUID
		actions         []action.Action
		currentStackID  uuid.UUID
		currentActionID uuid.UUID

		expectResAction   *action.Action
		expectResStackMap map[uuid.UUID]*stack.Stack
		expectRes         *stack.Stack
	}{
		{
			name: "empty stack",

			stackMap: map[uuid.UUID]*stack.Stack{},
			stackID:  stack.IDMain,
			actions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
					Type: action.TypeAnswer,
				},
			},
			currentStackID:  stack.IDEmpty,
			currentActionID: action.IDEmpty,

			expectResAction: &action.Action{
				ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
				Type: action.TypeAnswer,
			},
			expectResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			expectRes: &stack.Stack{
				ID: stack.IDMain,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
						Type: action.TypeAnswer,
					},
				},
				ReturnStackID:  stack.IDEmpty,
				ReturnActionID: action.IDEmpty,
			},
		},
		{
			name: "non-empty stack",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
							Type: action.TypeAnswer,
						},
					},
				},
			},
			stackID: uuid.FromStringOrNil("2976187e-f9cf-11ef-a24b-0379d7b40894"),
			actions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878df"),
					Type: action.TypeAnswer,
				},
			},
			currentStackID:  stack.IDMain,
			currentActionID: uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),

			expectResAction: &action.Action{
				ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878df"),
				Type: action.TypeAnswer,
			},
			expectResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("2976187e-f9cf-11ef-a24b-0379d7b40894"): {
					ID: uuid.FromStringOrNil("2976187e-f9cf-11ef-a24b-0379d7b40894"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878df"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
				},
			},
			expectRes: &stack.Stack{
				ID: uuid.FromStringOrNil("2976187e-f9cf-11ef-a24b-0379d7b40894"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878df"),
						Type: action.TypeAnswer,
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878de"),
			},
		},
		{
			name: "push multiple actions",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					Actions: []action.Action{},
				},
			},
			stackID: uuid.FromStringOrNil("074264c2-f9d1-11ef-9150-cbe1e547761d"),
			actions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e0"),
					Type: action.TypeAnswer,
				},
				{
					ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e1"),
					Type: action.TypeAnswer,
				},
			},
			currentStackID:  stack.IDMain,
			currentActionID: action.IDEmpty,

			expectResAction: &action.Action{
				ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e0"),
				Type: action.TypeAnswer,
			},
			expectResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID:             stack.IDMain,
					Actions:        []action.Action{},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("074264c2-f9d1-11ef-9150-cbe1e547761d"): {
					ID: uuid.FromStringOrNil("074264c2-f9d1-11ef-9150-cbe1e547761d"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e0"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e1"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: action.IDEmpty,
				},
			},
			expectRes: &stack.Stack{
				ID: uuid.FromStringOrNil("074264c2-f9d1-11ef-9150-cbe1e547761d"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e0"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("98207410-f928-11ef-aaab-3bd53b1878e1"),
						Type: action.TypeAnswer,
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: action.IDEmpty,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res, err := h.PushStackByActions(tt.stackMap, tt.stackID, tt.actions, tt.currentStackID, tt.currentActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// sort stackMap
			var tmpSort []string
			for stackID := range tt.stackMap {
				tmpSort = append(tmpSort, stackID.String())
			}
			sort.Strings(tmpSort)

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResAction, res)
			}

			i := 0
			for _, key := range tmpSort {
				s := tt.stackMap[uuid.FromStringOrNil(key)]

				tmp := getItemByIndex(tt.expectResStackMap, i)
				if tmp == nil {
					t.Errorf("Wrong match. expect: not nil, got: nil")
					continue
				}

				tmp.ID = s.ID
				if reflect.DeepEqual(tmp, s) != true {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tmp, s)
				}

				i++
			}
		})
	}
}

func Test_PopStack(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectResStackMap map[uuid.UUID]*stack.Stack
		expectResStack    *stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("690901cc-f9d9-11ef-b47a-43629eae889c"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("694418b6-f9d9-11ef-9b48-fb6368584463"): {
					ID: uuid.FromStringOrNil("694418b6-f9d9-11ef-9b48-fb6368584463"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("697628e2-f9d9-11ef-a7ac-5784b43997cc"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("690901cc-f9d9-11ef-b47a-43629eae889c"),
				},
			},
			stackID: uuid.FromStringOrNil("694418b6-f9d9-11ef-9b48-fb6368584463"),

			expectResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("690901cc-f9d9-11ef-b47a-43629eae889c"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			expectResStack: &stack.Stack{
				ID: uuid.FromStringOrNil("694418b6-f9d9-11ef-9b48-fb6368584463"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("697628e2-f9d9-11ef-a7ac-5784b43997cc"),
						Type: action.TypeAnswer,
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("690901cc-f9d9-11ef-b47a-43629eae889c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res, err := h.PopStack(tt.stackMap, tt.stackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.stackMap, tt.expectResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResStackMap, tt.stackMap)
			}

			if !reflect.DeepEqual(res, tt.expectResStack) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResStack, res)
			}
		})
	}
}
