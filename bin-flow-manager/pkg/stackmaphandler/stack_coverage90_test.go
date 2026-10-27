package stackmaphandler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

func Test_stack_boundary_coverage90(t *testing.T) {
	subStackID := uuid.FromStringOrNil("c9a0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")
	otherStackID := uuid.FromStringOrNil("c9a0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")
	actionID := uuid.FromStringOrNil("c9a0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c03")

	newMap := func() map[uuid.UUID]*stack.Stack {
		return map[uuid.UUID]*stack.Stack{
			stack.IDMain: {ID: stack.IDMain, Actions: []action.Action{{ID: actionID, Type: action.TypeAnswer}}},
			subStackID:   {ID: subStackID},
			otherStackID: nil,
		}
	}

	h := NewStackmapHandler().(*stackHandler)

	t.Run("deleting a stack that is not in the map or is nil leaves the map as it is", func(t *testing.T) {
		stackMap := newMap()
		h.DeleteStack(stackMap, uuid.FromStringOrNil("c9a0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c99"))
		h.DeleteStack(stackMap, otherStackID)

		if !reflect.DeepEqual(stackMap, newMap()) {
			t.Errorf("The map must not change.\ngot: %v", stackMap)
		}
	})

	t.Run("pushing an existing stack is refused and keeps the stack", func(t *testing.T) {
		stackMap := newMap()
		res, err := h.PushStackByActions(stackMap, subStackID, []action.Action{{ID: actionID}}, stack.IDMain, actionID)
		if err == nil || !strings.Contains(err.Error(), "stack already exists") {
			t.Fatalf("expected the exists failure, got: %v", err)
		}
		if res != nil {
			t.Errorf("no stack is expected with the error. got: %v", res)
		}
		if len(stackMap[subStackID].Actions) != 0 {
			t.Errorf("The existing stack must not be replaced. got: %v", stackMap[subStackID])
		}
	})

	t.Run("popping the main stack is refused", func(t *testing.T) {
		stackMap := newMap()
		res, err := h.PopStack(stackMap, stack.IDMain)
		if err == nil || !strings.Contains(err.Error(), "cannot pop main stack") {
			t.Fatalf("expected the main stack failure, got: %v", err)
		}
		if res != nil {
			t.Errorf("no stack is expected with the error. got: %v", res)
		}
		if _, ok := stackMap[stack.IDMain]; !ok {
			t.Errorf("The main stack must stay in the map")
		}
	})

	t.Run("popping a stack that is not in the map is refused", func(t *testing.T) {
		stackMap := newMap()
		res, err := h.PopStack(stackMap, uuid.FromStringOrNil("c9a0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c99"))
		if err == nil || !strings.Contains(err.Error(), "could not get the stack") {
			t.Fatalf("expected the lookup failure, got: %v", err)
		}
		if res != nil {
			t.Errorf("no stack is expected with the error. got: %v", res)
		}
	})
}
