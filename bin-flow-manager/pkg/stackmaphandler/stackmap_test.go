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
