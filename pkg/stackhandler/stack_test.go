package stackhandler

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
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

			ctx := context.Background()

			res := h.create(ctx, tt.stackID, tt.actions, tt.returnStrackID, tt.returnActionID)
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
		actions         []action.Action
		currentStackID  uuid.UUID
		currentActionID uuid.UUID

		expectResAction   *action.Action
		expectResStackMap map[uuid.UUID]*stack.Stack
	}{
		{
			"empty stack",

			map[uuid.UUID]*stack.Stack{},
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("128f9aa0-d438-11ec-bdb8-279917da3e16"),
					Type: action.TypeAnswer,
				},
			},
			uuid.Nil,
			uuid.Nil,

			&action.Action{
				ID:   uuid.FromStringOrNil("128f9aa0-d438-11ec-bdb8-279917da3e16"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("128f9aa0-d438-11ec-bdb8-279917da3e16"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
		},
		{
			"stack 1",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("01d4de42-d45f-11ec-9ebc-7381a12a81fe"),
					Type: action.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),

			&action.Action{
				ID:   uuid.FromStringOrNil("01d4de42-d45f-11ec-9ebc-7381a12a81fe"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
				uuid.FromStringOrNil("0ff75fbe-d45e-11ec-b74d-e7cf7a495d7a"): {
					ID: uuid.FromStringOrNil("0ff75fbe-d45e-11ec-b74d-e7cf7a495d7a"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("01d4de42-d45f-11ec-9ebc-7381a12a81fe"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					ReturnActionID: uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
				},
			},
		},
		{
			"stack 1 and actions have 2 actions",

			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
			},
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("01d4de42-d45f-11ec-9ebc-7381a12a81fe"),
					Type: action.TypeAnswer,
				},
				{
					ID:   uuid.FromStringOrNil("e75e3d90-d460-11ec-a32b-6399bdc94bd7"),
					Type: action.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),

			&action.Action{
				ID:   uuid.FromStringOrNil("01d4de42-d45f-11ec-9ebc-7381a12a81fe"),
				Type: action.TypeAnswer,
			},
			map[uuid.UUID]*stack.Stack{
				uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"): {
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					ReturnActionID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
				},
				uuid.FromStringOrNil("0ff75fbe-d45e-11ec-b74d-e7cf7a495d7a"): {
					ID: uuid.FromStringOrNil("0ff75fbe-d45e-11ec-b74d-e7cf7a495d7a"),
					Actions: []action.Action{
						{
							ID:   uuid.FromStringOrNil("01d4de42-d45f-11ec-9ebc-7381a12a81fe"),
							Type: action.TypeAnswer,
						},
						{
							ID:   uuid.FromStringOrNil("e75e3d90-d460-11ec-a32b-6399bdc94bd7"),
							Type: action.TypeAnswer,
						},
					},
					ReturnStackID:  uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
					ReturnActionID: uuid.FromStringOrNil("5795b980-d3b2-11ec-91c7-1b9727f56f87"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := &stackHandler{}

			ctx := context.Background()

			_, resAction, err := h.Push(ctx, tt.stackMap, tt.actions, tt.currentStackID, tt.currentActionID)
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
