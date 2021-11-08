package action

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

func TestConvertAction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name         string
		action       *fmaction.Action
		expectAction *Action
	}

	tests := []test{
		{
			"answer",
			&fmaction.Action{
				ID:   uuid.FromStringOrNil("b3c0647e-6781-11eb-903a-f33daa2ee7c3"),
				Type: fmaction.TypeAnswer,
			},
			&Action{
				Type: "answer",
			},
		},
		{
			"conference_join",
			&action.Action{
				ID:     uuid.FromStringOrNil("456cf9fa-6782-11eb-9b4f-fbde8ed2192e"),
				Type:   fmaction.TypeConferenceJoin,
				Option: []byte(`{conference_id":"4b0d3e14-7701-4f59-944e-91f0e66cce22"}`),
			},
			&Action{
				Type:   "conference_join",
				Option: []byte(`{conference_id":"4b0d3e14-7701-4f59-944e-91f0e66cce22"}`),
			},
		},
		{
			"echo",
			&action.Action{
				ID:   uuid.FromStringOrNil("ca9d8ffe-6787-11eb-94ca-bfb3a1122783"),
				Type: fmaction.TypeEcho,
			},
			&Action{
				Type: "echo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := ConvertAction(tt.action)
			if reflect.DeepEqual(tt.expectAction, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectAction, res)
			}
		})
	}
}

func TestCreateAction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name         string
		action       *Action
		expectAction *fmaction.Action
	}

	tests := []test{
		{
			"answer",
			&Action{
				Type: "answer",
			},
			&fmaction.Action{
				Type: fmaction.TypeAnswer,
			},
		},
		{
			"conference_join",
			&Action{
				Type:   "conference_join",
				Option: []byte(`{conference_id":"659ed044-6788-11eb-8b58-3f997557f9df"}`),
			},
			&fmaction.Action{
				Type:   fmaction.TypeConferenceJoin,
				Option: []byte(`{conference_id":"659ed044-6788-11eb-8b58-3f997557f9df"}`),
			},
		},
		{
			"echo",
			&Action{
				Type: "echo",
			},
			&fmaction.Action{
				Type: fmaction.TypeEcho,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := CreateAction(tt.action)
			if reflect.DeepEqual(tt.expectAction, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectAction, res)
			}
		})
	}
}
