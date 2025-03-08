package stackhandler

import (
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_StackMapInit(t *testing.T) {

	tests := []struct {
		name string

		actions []action.Action

		expectRes map[uuid.UUID]*stack.Stack
	}{
		{
			name: "normal",

			actions: []action.Action{
				{
					ID: uuid.FromStringOrNil("f21f4a3a-fc0d-11ef-b6e3-bfaf66248e80"),
				},
				{
					ID: uuid.FromStringOrNil("f26a47e2-fc0d-11ef-8a64-2b3827f68d54"),
				},
			},

			expectRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("f21f4a3a-fc0d-11ef-b6e3-bfaf66248e80"),
						},
						{
							ID: uuid.FromStringOrNil("f26a47e2-fc0d-11ef-8a64-2b3827f68d54"),
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

			res := h.StackMapInit(tt.actions)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StackMapGet(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectRes *stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("9b1bbe20-fc0e-11ef-84f5-8f461d8f19a1"),
						},
						{
							ID: uuid.FromStringOrNil("9b54d89a-fc0e-11ef-87d6-cf7690c6af89"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			stackID: stack.IDMain,

			expectRes: &stack.Stack{
				ID: stack.IDMain,
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("9b1bbe20-fc0e-11ef-84f5-8f461d8f19a1"),
					},
					{
						ID: uuid.FromStringOrNil("9b54d89a-fc0e-11ef-87d6-cf7690c6af89"),
					},
				},
				ReturnStackID:  stack.IDEmpty,
				ReturnActionID: action.IDEmpty,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res, err := h.StackMapGet(tt.stackMap, tt.stackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StackMapPush(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stack    *stack.Stack

		expectRes map[uuid.UUID]*stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("e3e494ba-fc0e-11ef-8822-7bc32b05ecde"),
						},
						{
							ID: uuid.FromStringOrNil("e4128596-fc0e-11ef-b2d6-4b2dcc17c26e"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			stack: &stack.Stack{
				ID: uuid.FromStringOrNil("e33a3eac-fc0e-11ef-9c6a-97ef8c5d4031"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("e3afb146-fc0e-11ef-bcdd-5723414879dc"),
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("e4128596-fc0e-11ef-b2d6-4b2dcc17c26e"),
			},

			expectRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("e3e494ba-fc0e-11ef-8822-7bc32b05ecde"),
						},
						{
							ID: uuid.FromStringOrNil("e4128596-fc0e-11ef-b2d6-4b2dcc17c26e"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("e33a3eac-fc0e-11ef-9c6a-97ef8c5d4031"): {
					ID: uuid.FromStringOrNil("e33a3eac-fc0e-11ef-9c6a-97ef8c5d4031"),
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("e3afb146-fc0e-11ef-bcdd-5723414879dc"),
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("e4128596-fc0e-11ef-b2d6-4b2dcc17c26e"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			if err := h.StackMapPush(tt.stackMap, tt.stack); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, tt.stackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, tt.stackMap)
			}
		})
	}
}

func Test_stackMapRemove(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectRes map[uuid.UUID]*stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("61aa1e2e-fc0f-11ef-ae4a-53f9be36fe2d"),
						},
						{
							ID: uuid.FromStringOrNil("61e9fd96-fc0f-11ef-81b3-fb31e11d07df"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("62216272-fc0f-11ef-a037-2b3235971a21"): {
					ID: uuid.FromStringOrNil("62216272-fc0f-11ef-a037-2b3235971a21"),
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("625727ae-fc0f-11ef-b67d-1b3f0a0abdda"),
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("61aa1e2e-fc0f-11ef-ae4a-53f9be36fe2d"),
				},
			},
			stackID: uuid.FromStringOrNil("62216272-fc0f-11ef-a037-2b3235971a21"),

			expectRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("61aa1e2e-fc0f-11ef-ae4a-53f9be36fe2d"),
						},
						{
							ID: uuid.FromStringOrNil("61e9fd96-fc0f-11ef-81b3-fb31e11d07df"),
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

			h.stackMapRemove(tt.stackMap, tt.stackID)

			if !reflect.DeepEqual(tt.expectRes, tt.stackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, tt.stackMap)
			}
		})
	}
}

func Test_StackMapPop(t *testing.T) {

	tests := []struct {
		name string

		stackMap map[uuid.UUID]*stack.Stack
		stackID  uuid.UUID

		expectStackMap map[uuid.UUID]*stack.Stack
		expectRes      *stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("0f8e2e72-fc10-11ef-8dd2-d78169b92244"),
						},
						{
							ID: uuid.FromStringOrNil("0fc8db44-fc10-11ef-9b7b-7b10db076db4"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("0ff44f40-fc10-11ef-aca1-7b5eb93d1fef"): {
					ID: uuid.FromStringOrNil("0ff44f40-fc10-11ef-aca1-7b5eb93d1fef"),
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("625727ae-fc0f-11ef-b67d-1b3f0a0abdda"),
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("0f8e2e72-fc10-11ef-8dd2-d78169b92244"),
				},
			},
			stackID: uuid.FromStringOrNil("0ff44f40-fc10-11ef-aca1-7b5eb93d1fef"),

			expectStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("0f8e2e72-fc10-11ef-8dd2-d78169b92244"),
						},
						{
							ID: uuid.FromStringOrNil("0fc8db44-fc10-11ef-9b7b-7b10db076db4"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			expectRes: &stack.Stack{
				ID: uuid.FromStringOrNil("0ff44f40-fc10-11ef-aca1-7b5eb93d1fef"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("625727ae-fc0f-11ef-b67d-1b3f0a0abdda"),
					},
				},
				ReturnStackID:  stack.IDMain,
				ReturnActionID: uuid.FromStringOrNil("0f8e2e72-fc10-11ef-8dd2-d78169b92244"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			res, err := h.StackMapPop(tt.stackMap, tt.stackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectStackMap, tt.stackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectStackMap, tt.stackMap)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StackMapPushActions(t *testing.T) {

	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		stackID        uuid.UUID
		targetActionID uuid.UUID
		actions        []action.Action

		expectRes map[uuid.UUID]*stack.Stack
	}{
		{
			name: "normal",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("ff5b8474-fbcd-11ef-b2d8-635dcecb71e6"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			stackID:        stack.IDMain,
			targetActionID: uuid.FromStringOrNil("ff5b8474-fbcd-11ef-b2d8-635dcecb71e6"),
			actions: []action.Action{
				{
					ID: uuid.FromStringOrNil("ff9a8480-fbcd-11ef-b111-3bb24eb836d9"),
				},
				{
					ID: uuid.FromStringOrNil("ffc0c6fe-fbcd-11ef-a6b7-93565861c0ba"),
				},
			},

			expectRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("ff5b8474-fbcd-11ef-b2d8-635dcecb71e6"),
						},
						{
							ID: uuid.FromStringOrNil("ff9a8480-fbcd-11ef-b111-3bb24eb836d9"),
						},
						{
							ID: uuid.FromStringOrNil("ffc0c6fe-fbcd-11ef-a6b7-93565861c0ba"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
		},
		{
			name: "push in the middle",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("982baf32-fbd1-11ef-86d7-bf6ab6ee0323"),
						},
						{
							ID: uuid.FromStringOrNil("991b1c8e-fbd1-11ef-8fdd-eb909f840bd9"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("9888e026-fbd1-11ef-b1e5-c7c42b317c7f"): {
					ID: uuid.FromStringOrNil("9888e026-fbd1-11ef-b1e5-c7c42b317c7f"),
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("98a98704-fbd1-11ef-9ccb-bfa46af0d721"),
						},
						{
							ID: uuid.FromStringOrNil("98ce6452-fbd1-11ef-866b-d3f4e1bec4fc"),
						},
						{
							ID: uuid.FromStringOrNil("98f5b156-fbd1-11ef-b5e6-9b611231d004"),
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("982baf32-fbd1-11ef-86d7-bf6ab6ee0323"),
				},
			},
			stackID:        uuid.FromStringOrNil("9888e026-fbd1-11ef-b1e5-c7c42b317c7f"),
			targetActionID: uuid.FromStringOrNil("98ce6452-fbd1-11ef-866b-d3f4e1bec4fc"),
			actions: []action.Action{
				{
					ID: uuid.FromStringOrNil("99419134-fbd1-11ef-9aad-dfa75b8ff53e"),
				},
				{
					ID: uuid.FromStringOrNil("99633c4e-fbd1-11ef-9b36-77dc4559204e"),
				},
			},

			expectRes: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("982baf32-fbd1-11ef-86d7-bf6ab6ee0323"),
						},
						{
							ID: uuid.FromStringOrNil("991b1c8e-fbd1-11ef-8fdd-eb909f840bd9"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("9888e026-fbd1-11ef-b1e5-c7c42b317c7f"): {
					ID: uuid.FromStringOrNil("9888e026-fbd1-11ef-b1e5-c7c42b317c7f"),
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("98a98704-fbd1-11ef-9ccb-bfa46af0d721"),
						},
						{
							ID: uuid.FromStringOrNil("98ce6452-fbd1-11ef-866b-d3f4e1bec4fc"),
						},
						{
							ID: uuid.FromStringOrNil("99419134-fbd1-11ef-9aad-dfa75b8ff53e"),
						},
						{
							ID: uuid.FromStringOrNil("99633c4e-fbd1-11ef-9b36-77dc4559204e"),
						},
						{
							ID: uuid.FromStringOrNil("98f5b156-fbd1-11ef-b5e6-9b611231d004"),
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("982baf32-fbd1-11ef-86d7-bf6ab6ee0323"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			if err := h.StackMapPushActions(tt.stackMap, tt.stackID, tt.targetActionID, tt.actions); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, tt.stackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, tt.stackMap)
			}
		})
	}
}
