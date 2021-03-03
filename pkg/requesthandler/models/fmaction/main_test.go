package fmaction

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

func TestConvertAction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name         string
		action       *Action
		expectAction *models.Action
	}

	tests := []test{
		{
			"answer",
			&Action{
				ID:   uuid.FromStringOrNil("b3c0647e-6781-11eb-903a-f33daa2ee7c3"),
				Type: "answer",
			},
			&models.Action{
				ID:   uuid.FromStringOrNil("b3c0647e-6781-11eb-903a-f33daa2ee7c3"),
				Type: models.ActionTypeAnswer,
			},
		},
		{
			"conference_join",
			&Action{
				ID:     uuid.FromStringOrNil("456cf9fa-6782-11eb-9b4f-fbde8ed2192e"),
				Type:   "conference_join",
				Option: []byte(`{conference_id":"4b0d3e14-7701-4f59-944e-91f0e66cce22"}`),
			},
			&models.Action{
				ID:     uuid.FromStringOrNil("456cf9fa-6782-11eb-9b4f-fbde8ed2192e"),
				Type:   models.ActionTypeConferenceJoin,
				Option: []byte(`{conference_id":"4b0d3e14-7701-4f59-944e-91f0e66cce22"}`),
			},
		},
		{
			"echo",
			&Action{
				ID:   uuid.FromStringOrNil("ca9d8ffe-6787-11eb-94ca-bfb3a1122783"),
				Type: "echo",
			},
			&models.Action{
				ID:   uuid.FromStringOrNil("ca9d8ffe-6787-11eb-94ca-bfb3a1122783"),
				Type: models.ActionTypeEcho,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.action.ConvertAction()
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
		action       *models.Action
		expectAction *Action
	}

	tests := []test{
		{
			"answer",
			&models.Action{
				ID:   uuid.FromStringOrNil("654aa32a-6788-11eb-9894-a75cf57ce24a"),
				Type: models.ActionTypeAnswer,
			},
			&Action{
				ID:   uuid.FromStringOrNil("654aa32a-6788-11eb-9894-a75cf57ce24a"),
				Type: "answer",
			},
		},
		{
			"conference_join",
			&models.Action{
				ID:     uuid.FromStringOrNil("6571092a-6788-11eb-9bc3-ab00dcd060a0"),
				Type:   models.ActionTypeConferenceJoin,
				Option: []byte(`{conference_id":"659ed044-6788-11eb-8b58-3f997557f9df"}`),
			},
			&Action{
				ID:     uuid.FromStringOrNil("6571092a-6788-11eb-9bc3-ab00dcd060a0"),
				Type:   "conference_join",
				Option: []byte(`{conference_id":"659ed044-6788-11eb-8b58-3f997557f9df"}`),
			},
		},
		{
			"echo",
			&models.Action{
				ID:   uuid.FromStringOrNil("7021b888-6788-11eb-8a4e-a3a71029cdbd"),
				Type: models.ActionTypeEcho,
			},
			&Action{
				ID:   uuid.FromStringOrNil("7021b888-6788-11eb-8a4e-a3a71029cdbd"),
				Type: "echo",
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
