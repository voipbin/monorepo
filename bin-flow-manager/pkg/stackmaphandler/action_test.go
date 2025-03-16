package stackmaphandler

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

func Test_GetAction(t *testing.T) {

	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		startStackID   uuid.UUID
		targetActionID uuid.UUID

		expectedResStackID  uuid.UUID
		epxectedResAction   *action.Action
		expectedResStackMap map[uuid.UUID]*stack.Stack
	}{
		{
			name: "action exist in the given stack",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("b1afd956-a8ae-11ed-a7fb-fba3920318fd"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("b1f14db4-a8ae-11ed-90e2-d3acbfa567b3"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("b22763b8-a8ae-11ed-8d8a-27edae05150c"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
			startStackID:   stack.IDMain,
			targetActionID: uuid.FromStringOrNil("b1afd956-a8ae-11ed-a7fb-fba3920318fd"),

			expectedResStackID: stack.IDMain,
			epxectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("b1afd956-a8ae-11ed-a7fb-fba3920318fd"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("b1afd956-a8ae-11ed-a7fb-fba3920318fd"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("b1f14db4-a8ae-11ed-90e2-d3acbfa567b3"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("b22763b8-a8ae-11ed-8d8a-27edae05150c"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
		},
		{
			name: "action exist in the other stack depth 1",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				},
			},
			startStackID:   uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
			targetActionID: uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),

			expectedResStackID: stack.IDMain,
			epxectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			name: "action exist in the other stack depth 2",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
				uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"): {
					ID: uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("127eb93f-3d03-4583-83c0-bd4a23c66f03"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
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
					ReturnActionID: uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
				},
				uuid.FromStringOrNil("79d1a532-f446-470f-bdc6-45542750d9cd"): {
					ID: uuid.FromStringOrNil("79d1a532-f446-470f-bdc6-45542750d9cd"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("2549adfe-e7c5-4c10-ba5b-fb29bbeac441"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("f725f016-d3b5-11ec-9897-eb58118fdc21"),
					ReturnActionID: uuid.FromStringOrNil("fb79a806-d3b5-11ec-8418-bfbfa0a8638e"),
				},
			},
			startStackID:   uuid.FromStringOrNil("79d1a532-f446-470f-bdc6-45542750d9cd"),
			targetActionID: uuid.FromStringOrNil("127eb93f-3d03-4583-83c0-bd4a23c66f03"),

			expectedResStackID: uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
			epxectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("127eb93f-3d03-4583-83c0-bd4a23c66f03"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
				uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"): {
					ID: uuid.FromStringOrNil("93def85e-d3b5-11ec-b6e5-6751b01de122"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("127eb93f-3d03-4583-83c0-bd4a23c66f03"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("9453b73e-d3b5-11ec-b636-0fcf55d52956"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			resStackID, resAction, err := h.GetAction(tt.stackMap, tt.startStackID, tt.targetActionID, true)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(resStackID, tt.expectedResStackID) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedResStackID, resStackID)
			}

			if !reflect.DeepEqual(resAction, tt.epxectedResAction) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.epxectedResAction, resAction)
			}

			if !reflect.DeepEqual(tt.stackMap, tt.expectedResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.epxectedResAction, resAction)
			}
		})
	}
}

func Test_GetAction_error(t *testing.T) {

	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		startStackID   uuid.UUID
		targetActionID uuid.UUID
	}{
		{
			name: "action id does not exist",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("51e04796-d3b5-11ec-a41b-1fb38082327f"),
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
			startStackID:   stack.IDMain,
			targetActionID: uuid.FromStringOrNil("5fe2fada-fd80-11ef-9907-03656d336254"),
		},
		{
			name: "action id does exist but the stack is not in the same stack",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("68053b06-fd80-11ef-9ce5-532418cfaa6e"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
				uuid.FromStringOrNil("682ebb34-fd80-11ef-98f3-f35d931f2253"): {
					ID: uuid.FromStringOrNil("682ebb34-fd80-11ef-98f3-f35d931f2253"),
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("685c7e2a-fd80-11ef-9ba2-ebad010387c0"),
						},
					},
				},
			},
			startStackID:   stack.IDMain,
			targetActionID: uuid.FromStringOrNil("685c7e2a-fd80-11ef-9ba2-ebad010387c0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			_, _, err := h.GetAction(tt.stackMap, tt.startStackID, tt.targetActionID, true)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
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

		expectedResStackMap map[uuid.UUID]*stack.Stack
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

			expectedResStackMap: map[uuid.UUID]*stack.Stack{
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

			_, resAction, err := h.GetAction(tt.stackMap, tt.currentStackID, tt.targetActionID, false)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			resAction.Option = tt.updateOption

			if !reflect.DeepEqual(tt.stackMap, tt.expectedResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedResStackMap[stack.IDMain], tt.stackMap[stack.IDMain])
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

		expectedResStackID  uuid.UUID
		expectedResAction   *action.Action
		expectedResStackMap map[uuid.UUID]*stack.Stack
	}{
		{
			name: "next action exist in the same stack",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			currentStackID: stack.IDMain,
			currentAction: &action.Action{
				ID:   uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
				Type: action.TypeAnswer,
			},

			expectedResStackID: stack.IDMain,
			expectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			name: "the current action has next id and the next action exist in the same stack",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			currentStackID: stack.IDMain,
			currentAction: &action.Action{
				ID:     uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
				Type:   action.TypeAnswer,
				NextID: uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
			},

			expectedResStackID: stack.IDMain,
			expectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			name: "the current action in the end of actions",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("342f0e56-d3b7-11ec-a9a6-1b8c6f3600ee"),
				},
			},
			currentStackID: uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"),
			currentAction: &action.Action{
				ID:   uuid.FromStringOrNil("f5c379b2-d466-11ec-af74-6fee9d235883"),
				Type: action.TypeAnswer,
			},

			expectedResStackID: stack.IDMain,
			expectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			name: "the current action in the end of actions and retrun action has next id",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
					ReturnStackID:  stack.IDMain,
					ReturnActionID: uuid.FromStringOrNil("ff9c0da6-d3b6-11ec-b1e7-e35b7eafa103"),
				},
			},
			currentStackID: uuid.FromStringOrNil("e9d59e50-d466-11ec-a214-7795c33e5df4"),
			currentAction: &action.Action{
				ID:   uuid.FromStringOrNil("f5c379b2-d466-11ec-af74-6fee9d235883"),
				Type: action.TypeAnswer,
			},

			expectedResStackID: stack.IDMain,
			expectedResAction: &action.Action{
				ID:   uuid.FromStringOrNil("f75cdd64-d3b6-11ec-9ef6-af4e5a66b496"),
				Type: action.TypeAnswer,
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
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
			name: "the current action is start action",

			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("502a96ba-fda4-11ef-85b7-e7df9e7f94c2"),
						},
						{
							ID: uuid.FromStringOrNil("5073826c-fda4-11ef-a5f0-f78b1e53c254"),
						},
						{
							ID: uuid.FromStringOrNil("509ceeb8-fda4-11ef-b102-434dca711dbc"),
						},
					},
					ReturnStackID:  stack.IDEmpty,
					ReturnActionID: action.IDEmpty,
				},
			},
			currentStackID: stack.IDMain,
			currentAction: &action.Action{
				ID: action.IDStart,
			},

			expectedResStackID: stack.IDMain,
			expectedResAction: &action.Action{
				ID: uuid.FromStringOrNil("502a96ba-fda4-11ef-85b7-e7df9e7f94c2"),
			},
			expectedResStackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{
							ID: uuid.FromStringOrNil("502a96ba-fda4-11ef-85b7-e7df9e7f94c2"),
						},
						{
							ID: uuid.FromStringOrNil("5073826c-fda4-11ef-a5f0-f78b1e53c254"),
						},
						{
							ID: uuid.FromStringOrNil("509ceeb8-fda4-11ef-b102-434dca711dbc"),
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

			resStackID, resAction := h.GetNextAction(tt.stackMap, tt.currentStackID, tt.currentAction, true)

			if !reflect.DeepEqual(resStackID, tt.expectedResStackID) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedResStackID, resStackID)
			}

			if !reflect.DeepEqual(resAction, tt.expectedResAction) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedResAction, resAction)
			}

			if !reflect.DeepEqual(tt.stackMap, tt.expectedResStackMap) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedResAction, resAction)
			}

		})
	}
}

func Test_findAction(t *testing.T) {

	tests := []struct {
		name string

		actions  []action.Action
		actionID uuid.UUID

		expectedResIdx    int
		expectedResAction *action.Action
	}{
		{
			name: "goto action update loop",

			actions: []action.Action{
				{
					ID: uuid.FromStringOrNil("554e80da-f81f-11ec-a9e8-67b197e30dbe"),
				},
				{
					ID: uuid.FromStringOrNil("48b6b536-f81f-11ec-8b52-7b901ad4635d"),
				},
				{
					ID: uuid.FromStringOrNil("63a3a480-f81f-11ec-9545-d7c6f39069a3"),
				},
			},
			actionID: uuid.FromStringOrNil("48b6b536-f81f-11ec-8b52-7b901ad4635d"),

			expectedResIdx: 1,
			expectedResAction: &action.Action{
				ID: uuid.FromStringOrNil("48b6b536-f81f-11ec-8b52-7b901ad4635d"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			idx, res := h.findAction(tt.actions, tt.actionID)

			if idx != tt.expectedResIdx {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedResIdx, idx)
			}
			if !reflect.DeepEqual(tt.expectedResAction, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedResAction, res)
			}
		})
	}
}
