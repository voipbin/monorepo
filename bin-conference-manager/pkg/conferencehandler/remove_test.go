package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-call-manager/models/confbridge"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_RemoveConferencecallID(t *testing.T) {

	tests := []struct {
		name string

		conferenceID     uuid.UUID
		conferencecallID uuid.UUID

		responseConference *conference.Conference
	}{
		{
			"conference type and not terminating",

			uuid.FromStringOrNil("db1133c4-149a-11ed-be62-d3681a989fb4"),
			uuid.FromStringOrNil("5dd38eba-149b-11ed-a715-17c3951b2c26"),

			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("db1133c4-149a-11ed-be62-d3681a989fb4"),
				},
				Type: conference.TypeConference,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceRemoveConferencecallID(ctx, tt.responseConference.ID, tt.conferencecallID).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.responseConference.ID).Return(tt.responseConference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConference.CustomerID, conference.EventTypeConferenceUpdated, tt.responseConference)

			res, err := h.RemoveConferencecallID(ctx, tt.conferenceID, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_removeConferencecallIDTypeConference(t *testing.T) {

	tests := []struct {
		name string

		conference *conference.Conference
	}{
		{
			"status is not terminating",

			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9fa895c2-14c5-11ed-8f28-43e3f54564e5"),
				},
			},
		},
		{
			"status is terminating and have conferencecalls",

			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2d48b8c-14c5-11ed-a71a-b3ec4bcdf2eb"),
				},
				Status: conference.StatusTerminating,
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("d6ea1330-14c5-11ed-a48e-471aed8a9cee"),
				},
			},
		},
		{
			"status is terminating and have no conferencecalls",

			&conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2d48b8c-14c5-11ed-a71a-b3ec4bcdf2eb"),
				},
				Status:            conference.StatusTerminating,
				ConferencecallIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			if tt.conference.Status == conference.StatusTerminating && len(tt.conference.ConferencecallIDs) == 0 {
				mockReq.EXPECT().CallV1ConfbridgeDelete(ctx, tt.conference.ConfbridgeID).Return(&confbridge.Confbridge{}, nil)
				mockDB.EXPECT().ConferenceEnd(ctx, tt.conference.ID).Return(nil)
				mockDB.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(tt.conference, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.conference.CustomerID, conference.EventTypeConferenceDeleted, tt.conference)
			}

			if err := h.removeConferencecallIDTypeConference(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
