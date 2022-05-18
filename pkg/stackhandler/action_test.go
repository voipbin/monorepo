package stackhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
)

func Test_GetAction(t *testing.T) {

	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		currentStackID uuid.UUID
		targetActionID uuid.UUID

		expectResStackID  uuid.UUID
		epxectResAction   *action.Action
		expectResStackMap map[uuid.UUID]*stack.Stack
	}{
		{
			"action exist in the given stack",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
		},
		{
			"action exist in the other stack depth 1",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"): {
					ID: uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					ReturnActionID: uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				},
			},
			uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
			uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
		},
		{
			"action exist in the other stack depth 2",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5edb555c-d3b6-11ec-8c3d-43092e3123e7"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("f725f016-d3b5-11ec-9897-eb58118fdc21"): {
					ID: uuid.FromStringOrNil("f725f016-d3b5-11ec-9897-eb58118fdc21"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("fb79a806-d3b5-11ec-8418-bfbfa0a8638e"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
					ReturnActionID: uuid.FromStringOrNil("fb79a806-d3b5-11ec-8418-bfbfa0a8638e"),
				},
				uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"): {
					ID: uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					ReturnActionID: uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				},
			},
			uuid.FromStringOrNil("f725f016-d3b5-11ec-9897-eb58118fdc21"),
			uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5edb555c-d3b6-11ec-8c3d-43092e3123e7"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
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

			ctx := context.Background()

			resStackID, resAction, err := h.GetAction(ctx, tt.stackMap, tt.currentStackID, tt.targetActionID, true)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(resStackID, tt.expectResStackID) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResStackID, resStackID)
			}

			if !reflect.DeepEqual(resAction, tt.epxectResAction) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.epxectResAction, resAction)
			}

			if !reflect.DeepEqual(tt.stackMap, tt.expectResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.epxectResAction, resAction)
			}
		})
	}
}

func Test_GetActionReference(t *testing.T) {

	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		currentStackID uuid.UUID
		targetActionID uuid.UUID

		updateOption json.RawMessage

		expectResStackMap map[uuid.UUID]*stack.Stack
	}{
		{
			name: "goto action update loop",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:     uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type:   action.TypeGoto,
							Option: []byte(`{"target_id":"6b4ecab6-d57e-11ec-9593-1b54970a3f8c","loop_count":3}`),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			currentStackID: stack.IDMain,
			targetActionID: uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),

			updateOption: []byte(`{"target_id":"6b4ecab6-d57e-11ec-9593-1b54970a3f8c","loop_count":2}`),

			expectResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:     uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type:   action.TypeGoto,
							Option: []byte(`{"target_id":"6b4ecab6-d57e-11ec-9593-1b54970a3f8c","loop_count":2}`),
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

			ctx := context.Background()

			_, resAction, err := h.GetAction(ctx, tt.stackMap, tt.currentStackID, tt.targetActionID, false)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			resAction.Option = tt.updateOption

			if !reflect.DeepEqual(tt.stackMap, tt.expectResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResStackMap[stack.IDMain], tt.stackMap[stack.IDMain])
			}
		})
	}
}

func Test_GetNextAction(t *testing.T) {

	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		currentStackID uuid.UUID
		currentAction  *action.Action

		expectResStackID  uuid.UUID
		epxectResAction   *action.Action
		expectResStackMap map[uuid.UUID]*stack.Stack
	}{
		{
			"next action exist in the same stack",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
				Type: action.TypeAnswer,
			},

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
		},
		{
			"the current action has next id and the next action exist in the same stack",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:     uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type:   action.TypeAnswer,
							NextID: uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
						},
						{
							ID:   uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:     uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
				Type:   action.TypeAnswer,
				NextID: uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
			},

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:     uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type:   action.TypeAnswer,
							NextID: uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
						},
						{
							ID:   uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
		},
		{
			"the current action in the end of actions",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"): {
					ID: uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("f566b6be-d466-11ec-a0f6-3791ed281646"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f595e6e6-d466-11ec-ba65-8babefff694b"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f5c379b2-d466-11ec-af74-6fee9d235883"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					ReturnActionID: uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
				},
			},
			uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f5c379b2-d466-11ec-af74-6fee9d235883"),
				Type: action.TypeAnswer,
			},

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
		},
		{
			"the current action in the end of actions and retrun action has next id",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:     uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type:   action.TypeAnswer,
							NextID: uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
						},
						{
							ID:   uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"): {
					ID: uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("f566b6be-d466-11ec-a0f6-3791ed281646"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f595e6e6-d466-11ec-ba65-8babefff694b"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f5c379b2-d466-11ec-af74-6fee9d235883"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					ReturnActionID: uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
				},
			},
			uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f5c379b2-d466-11ec-af74-6fee9d235883"),
				Type: action.TypeAnswer,
			},

			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			&action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:     uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
							Type:   action.TypeAnswer,
							NextID: uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
						},
						{
							ID:   uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
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

			ctx := context.Background()

			resStackID, resAction := h.GetNextAction(ctx, tt.stackMap, tt.currentStackID, tt.currentAction, true)

			if !reflect.DeepEqual(resStackID, tt.expectResStackID) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectResStackID, resStackID)
			}

			if !reflect.DeepEqual(resAction, tt.epxectResAction) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.epxectResAction, resAction)
			}

			if !reflect.DeepEqual(tt.stackMap, tt.expectResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.epxectResAction, resAction)
			}

		})
	}
}
