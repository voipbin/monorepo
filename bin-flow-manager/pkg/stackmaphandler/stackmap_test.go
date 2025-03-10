package stackmaphandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		actions []action.Action

		expectRes map[uuid.UUID]*stack.Stack
	}{
		{
			name: "normal",

			actions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("bb18cf48-fd61-11ef-a6ce-13ea271274d7"),
					Type: action.TypeAnswer,
				},
			},

			expectRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("bb18cf48-fd61-11ef-a6ce-13ea271274d7"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res := h.Create(tt.actions)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Pop(t *testing.T) {

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
