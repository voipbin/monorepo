package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

func TestConferenceCreate(t *testing.T) {
	type test struct {
		name string

		conference       *conference.Conference
		expectConference *conference.Conference
	}

	tests := []test{
		{
			"type echo",
			&conference.Conference{
				ID: uuid.FromStringOrNil("eb8e31ec-9162-11ea-ba76-cbd8f42249bd"),
				Type: conference.TypeEcho,
				Name: "test type echo",
				Detail: "test type echo detail",

				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("eb8e31ec-9162-11ea-ba76-cbd8f42249bd"),
				Type: conference.TypeEcho,
				Name: "test type echo",
				Detail: "test type echo detail",

				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
		{
			"type conference",
			&conference.Conference{
				ID: uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				Type: conference.TypeConference,
				Name: "test type conference",
				Detail: "test type conference detail",

				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("26a42912-9163-11ea-93ca-bf5915635f88"),
				Type: conference.TypeConference,
				Name: "test type conference",
				Detail: "test type conference detail",

				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
		{
			"type transfer",
			&conference.Conference{
				ID: uuid.FromStringOrNil("483a5dee-9163-11ea-95c5-cbea96d71f7b"),
				Type: conference.TypeTransfer,
				Name: "test type transfer",
				Detail: "test type transfer detail",

				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("483a5dee-9163-11ea-95c5-cbea96d71f7b"),
				Type: conference.TypeTransfer,
				Name: "test type transfer",
				Detail: "test type transfer detail",

				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			if err := h.ConferenceCreate(context.Background(), tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ConferenceGet(context.Background(), tt.conference.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectConference, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectConference, res)
			}
		})
	}
}
