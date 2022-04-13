package activeflowhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

func TestAppendActions(t *testing.T) {

	tests := []struct {
		name         string
		action1      []action.Action
		action2      []action.Action
		expectAction []action.Action

		targetActionID uuid.UUID
	}{
		{
			"normal",
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
			},

			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			af := &activeflow.Activeflow{
				Actions: tt.action1,
			}
			if err := appendActions(af, tt.targetActionID, tt.action2); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(af.Actions, tt.expectAction) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, tt.action1)
			}
		})
	}
}

func TestReplaceActions(t *testing.T) {

	tests := []struct {
		name         string
		action1      []action.Action
		action2      []action.Action
		expectAction []action.Action

		targetActionID uuid.UUID
	}{
		{
			"normal",
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			[]action.Action{
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
			},

			[]action.Action{
				{
					ID: uuid.FromStringOrNil("c0a54954-0a96-11eb-80b2-8b6ef3a21db9"),
				},
				{
					ID: uuid.FromStringOrNil("e14c605c-0a96-11eb-9542-233abdd04f35"),
				},
				{
					ID: uuid.FromStringOrNil("e1858a8a-0a96-11eb-bf05-ab02488632d7"),
				},
				{
					ID: uuid.FromStringOrNil("e1b6e8d2-0a96-11eb-be8e-131d2f0bf1fe"),
				},
				{
					ID: uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
				},
			},
			uuid.FromStringOrNil("ce32b80e-0a96-11eb-9ca3-3f423a830f93"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			af := &activeflow.Activeflow{
				Actions: tt.action1,
			}
			if err := replaceActions(af, tt.targetActionID, tt.action2); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(af.Actions, tt.expectAction) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAction, tt.action1)
			}
		})
	}
}
