package stackhandler

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		stackID        uuid.UUID
		actions        []action.Action
		returnStrackID uuid.UUID
		returnActionID uuid.UUID

		expectRes *stack.Stack
	}{
		{
			"normal",

			uuid.FromStringOrNil("12725288-d3b1-11ec-bed2-efc5fa0d4d4d"),
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("3f7490a2-d3b1-11ec-841f-476ef3aa19b4"),
					Type: action.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("24dd1ad4-d3b1-11ec-af44-53536de24817"),
			uuid.FromStringOrNil("250395ce-d3b1-11ec-b2a1-5b16a7f48e78"),

			&stack.Stack{
				ID: uuid.FromStringOrNil("12725288-d3b1-11ec-bed2-efc5fa0d4d4d"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("3f7490a2-d3b1-11ec-841f-476ef3aa19b4"),
						Type: action.TypeAnswer,
					},
				},
				ReturnStackID:  uuid.FromStringOrNil("24dd1ad4-d3b1-11ec-af44-53536de24817"),
				ReturnActionID: uuid.FromStringOrNil("250395ce-d3b1-11ec-b2a1-5b16a7f48e78"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res := h.create(tt.stackID, tt.actions, tt.returnStrackID, tt.returnActionID)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectRes *stack.Stack
	}{
		{
			"normal",

			map[uuid.UUID]*stack.Stack{
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
			uuid.FromStringOrNil("3d983a34-d3b3-11ec-bf41-17fd979cd40f"),

			&stack.Stack{
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

			ctx := context.Background()

			res, err := h.Get(ctx, tt.stackMap, tt.stackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Push(t *testing.T) {

	tests := []struct {
		name string

		stackMap        map[uuid.UUID]*stack.Stack
		stackID         uuid.UUID
		actions         []action.Action
		currentStackID  uuid.UUID
		currentActionID uuid.UUID

		expectResAction   *action.Action
		expectResStackMap map[uuid.UUID]*stack.Stack
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			ctx := context.Background()

			_, resAction, err := h.Push(ctx, tt.stackMap, tt.stackID, tt.actions, tt.currentStackID, tt.currentActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// sort stackMap
			var tmpSort []string
			for stackID := range tt.stackMap {
				tmpSort = append(tmpSort, stackID.String())
			}
			sort.Strings(tmpSort)

			if reflect.DeepEqual(resAction, tt.expectResAction) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResAction, resAction)
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
