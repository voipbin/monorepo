package stackmaphandler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

var (
	testBoundaryActionA = uuid.FromStringOrNil("5c0b6a10-7d2e-4a51-8c01-1a2b3c4d5e01")
	testBoundaryActionB = uuid.FromStringOrNil("5c0b6a10-7d2e-4a51-8c01-1a2b3c4d5e02")
	testBoundaryActionC = uuid.FromStringOrNil("5c0b6a10-7d2e-4a51-8c01-1a2b3c4d5e03")
	testBoundaryMissing = uuid.FromStringOrNil("5c0b6a10-7d2e-4a51-8c01-1a2b3c4d5e04")
	testBoundarySubID   = uuid.FromStringOrNil("5c0b6a10-7d2e-4a51-8c01-1a2b3c4d5e05")
)

func Test_GetAction_boundary(t *testing.T) {
	tests := []struct {
		name string

		stackMap       map[uuid.UUID]*stack.Stack
		startStackID   uuid.UUID
		targetActionID uuid.UUID
		releaseStack   bool

		expectedErrPart string
		expectedStackID uuid.UUID
		expectedAction  *action.Action
		expectedRemain  []uuid.UUID
	}{
		{
			name:            "empty stack id",
			stackMap:        map[uuid.UUID]*stack.Stack{},
			startStackID:    stack.IDEmpty,
			targetActionID:  testBoundaryActionA,
			expectedErrPart: "invalid stack id",
			expectedStackID: stack.IDEmpty,
		},
		{
			name:            "empty action id",
			stackMap:        map[uuid.UUID]*stack.Stack{},
			startStackID:    stack.IDMain,
			targetActionID:  action.IDEmpty,
			expectedErrPart: "invalid action id",
			expectedStackID: stack.IDEmpty,
		},
		{
			name:            "start stack does not exist",
			stackMap:        map[uuid.UUID]*stack.Stack{},
			startStackID:    testBoundarySubID,
			targetActionID:  testBoundaryActionA,
			expectedErrPart: "could not find stack. stack_id: " + testBoundarySubID.String(),
			expectedStackID: stack.IDEmpty,
		},
		{
			name: "main stack has no actions",
			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {ID: stack.IDMain},
			},
			startStackID:    stack.IDMain,
			targetActionID:  testBoundaryActionA,
			expectedErrPart: "actions are empty",
			expectedStackID: stack.IDEmpty,
		},
		{
			name: "start action of the main stack is the first action",
			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{ID: testBoundaryActionA, Type: action.TypeAnswer},
						{ID: testBoundaryActionB, Type: action.TypeHangup},
					},
				},
			},
			startStackID:    stack.IDMain,
			targetActionID:  action.IDStart,
			expectedStackID: stack.IDMain,
			expectedAction:  &action.Action{ID: testBoundaryActionA, Type: action.TypeAnswer},
			expectedRemain:  []uuid.UUID{stack.IDMain},
		},
		{
			name: "return stack loops to itself without releasing exceeds the stack count",
			stackMap: map[uuid.UUID]*stack.Stack{
				testBoundarySubID: {
					ID:            testBoundarySubID,
					Actions:       []action.Action{{ID: testBoundaryActionA, Type: action.TypeAnswer}},
					ReturnStackID: testBoundarySubID,
				},
			},
			startStackID:    testBoundarySubID,
			targetActionID:  testBoundaryMissing,
			releaseStack:    false,
			expectedErrPart: "exceed mac stack count",
			expectedStackID: stack.IDEmpty,
			expectedRemain:  []uuid.UUID{testBoundarySubID},
		},
		{
			name: "return stack loops to itself with releasing removes the stack",
			stackMap: map[uuid.UUID]*stack.Stack{
				testBoundarySubID: {
					ID:            testBoundarySubID,
					Actions:       []action.Action{{ID: testBoundaryActionA, Type: action.TypeAnswer}},
					ReturnStackID: testBoundarySubID,
				},
			},
			startStackID:    testBoundarySubID,
			targetActionID:  testBoundaryMissing,
			releaseStack:    true,
			expectedErrPart: "could not find stack. stack_id: " + testBoundarySubID.String(),
			expectedStackID: stack.IDEmpty,
			expectedRemain:  []uuid.UUID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &stackHandler{}

			resStackID, resAction, err := h.GetAction(tt.stackMap, tt.startStackID, tt.targetActionID, tt.releaseStack)

			if tt.expectedErrPart == "" && err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.expectedErrPart != "" && (err == nil || !strings.Contains(err.Error(), tt.expectedErrPart)) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectedErrPart, err)
			}
			if resStackID != tt.expectedStackID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectedStackID, resStackID)
			}
			if !reflect.DeepEqual(resAction, tt.expectedAction) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedAction, resAction)
			}
			if tt.expectedRemain != nil && len(tt.stackMap) != len(tt.expectedRemain) {
				t.Errorf("Wrong match. expect stacks: %v, got: %v", tt.expectedRemain, tt.stackMap)
			}
			for _, id := range tt.expectedRemain {
				if _, ok := tt.stackMap[id]; !ok {
					t.Errorf("Wrong match. expect the stack to remain. stack_id: %s", id)
				}
			}
		})
	}
}

func Test_GetNextAction_boundary(t *testing.T) {
	finish := &action.ActionFinish

	tests := []struct {
		name string

		stackMap        map[uuid.UUID]*stack.Stack
		currentStackID  uuid.UUID
		currentActionID uuid.UUID

		expectedStackID uuid.UUID
	}{
		{
			name:            "empty current stack id finishes in the main stack",
			stackMap:        map[uuid.UUID]*stack.Stack{},
			currentStackID:  stack.IDEmpty,
			currentActionID: testBoundaryActionA,
			expectedStackID: stack.IDMain,
		},
		{
			name:            "current stack does not exist",
			stackMap:        map[uuid.UUID]*stack.Stack{},
			currentStackID:  testBoundarySubID,
			currentActionID: testBoundaryActionA,
			expectedStackID: stack.IDEmpty,
		},
		{
			name: "current action is not in the stack",
			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID:      stack.IDMain,
					Actions: []action.Action{{ID: testBoundaryActionA, Type: action.TypeAnswer}},
				},
			},
			currentStackID:  stack.IDMain,
			currentActionID: testBoundaryMissing,
			expectedStackID: stack.IDEmpty,
		},
		{
			name: "next_id points to an action that does not exist",
			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{ID: testBoundaryActionA, Type: action.TypeAnswer, NextID: testBoundaryMissing},
						{ID: testBoundaryActionB, Type: action.TypeHangup},
					},
				},
			},
			currentStackID:  stack.IDMain,
			currentActionID: testBoundaryActionA,
			expectedStackID: stack.IDEmpty,
		},
		{
			name: "the following action has an empty id",
			stackMap: map[uuid.UUID]*stack.Stack{
				stack.IDMain: {
					ID: stack.IDMain,
					Actions: []action.Action{
						{ID: testBoundaryActionA, Type: action.TypeAnswer},
						{ID: action.IDEmpty, Type: action.TypeHangup},
					},
				},
			},
			currentStackID:  stack.IDMain,
			currentActionID: testBoundaryActionA,
			expectedStackID: stack.IDEmpty,
		},
		{
			name: "return address loops to the last action of the same stack exceeds the stack count",
			stackMap: map[uuid.UUID]*stack.Stack{
				testBoundarySubID: {
					ID: testBoundarySubID,
					Actions: []action.Action{
						{ID: testBoundaryActionA, Type: action.TypeAnswer},
						{ID: testBoundaryActionC, Type: action.TypeHangup},
					},
					ReturnStackID:  testBoundarySubID,
					ReturnActionID: testBoundaryActionC,
				},
			},
			currentStackID:  testBoundarySubID,
			currentActionID: testBoundaryActionC,
			expectedStackID: stack.IDMain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &stackHandler{}

			resStackID, resAction := h.GetNextAction(tt.stackMap, tt.currentStackID, tt.currentActionID, false)

			if resStackID != tt.expectedStackID {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectedStackID, resStackID)
			}
			if !reflect.DeepEqual(resAction, finish) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", finish, resAction)
			}
		})
	}
}

func Test_AddActions_boundary(t *testing.T) {
	newStackMap := func() map[uuid.UUID]*stack.Stack {
		return map[uuid.UUID]*stack.Stack{
			stack.IDMain: {
				ID:      stack.IDMain,
				Actions: []action.Action{{ID: testBoundaryActionA, Type: action.TypeAnswer}},
			},
		}
	}
	added := []action.Action{{ID: testBoundaryActionB, Type: action.TypeHangup}}

	tests := []struct {
		name string

		targetStackID  uuid.UUID
		targetActionID uuid.UUID

		expectedErrPart string
	}{
		{"empty stack id", stack.IDEmpty, testBoundaryActionA, "invalid stack id"},
		{"empty target action id", stack.IDMain, action.IDEmpty, "invalid target action id"},
		{"stack does not exist", testBoundarySubID, testBoundaryActionA, "could not get stack. stack_id: " + testBoundarySubID.String()},
		{"target action does not exist", stack.IDMain, testBoundaryMissing, "could not find action. action_id: " + testBoundaryMissing.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &stackHandler{}
			stackMap := newStackMap()

			err := h.AddActions(stackMap, tt.targetStackID, tt.targetActionID, added)
			if err == nil || !strings.Contains(err.Error(), tt.expectedErrPart) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectedErrPart, err)
			}

			// a rejected request must not change the stack
			expectActions := newStackMap()[stack.IDMain].Actions
			if !reflect.DeepEqual(stackMap[stack.IDMain].Actions, expectActions) {
				t.Errorf("Wrong match. the stack was changed.\nexpect: %v\ngot: %v", expectActions, stackMap[stack.IDMain].Actions)
			}
		})
	}
}

func Test_NewStackmapHandler_pushStackGeneratesID(t *testing.T) {
	h := NewStackmapHandler()
	stackMap := map[uuid.UUID]*stack.Stack{}

	res, err := h.PushStackByActions(stackMap, uuid.Nil, []action.Action{{ID: testBoundaryActionA, Type: action.TypeAnswer}}, stack.IDMain, testBoundaryActionA)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID == uuid.Nil {
		t.Fatalf("Wrong match. expect: a generated stack id, got: nil uuid")
	}
	if stackMap[res.ID] != res {
		t.Errorf("Wrong match. expect the pushed stack in the map. stack_id: %s", res.ID)
	}
	if res.ReturnStackID != stack.IDMain || res.ReturnActionID != testBoundaryActionA {
		t.Errorf("Wrong match. expect return address: %s %s, got: %s %s", stack.IDMain, testBoundaryActionA, res.ReturnStackID, res.ReturnActionID)
	}
}
