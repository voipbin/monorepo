package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

func Test_Leaved(t *testing.T) {

	tests := []struct {
		name string

		conference  *conference.Conference
		referenceID uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"conference type and not terminating",

			&conference.Conference{
				ID:   uuid.FromStringOrNil("db1133c4-149a-11ed-be62-d3681a989fb4"),
				Type: conference.TypeConference,
			},
			uuid.FromStringOrNil("e41b141c-149a-11ed-bf7f-7b9a8e34e993"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("5dd38eba-149b-11ed-a715-17c3951b2c26"),
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
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,

				conferencecallHandler: mockConferencecall,
			}
			ctx := context.Background()

			mockConferencecall.EXPECT().GetByReferenceID(ctx, tt.referenceID).Return(tt.responseConferencecall, nil)
			mockConferencecall.EXPECT().UpdateStatusLeaved(ctx, tt.responseConferencecall.ID).Return(&conferencecall.Conferencecall{}, nil)
			mockDB.EXPECT().ConferenceRemoveConferencecallID(ctx, tt.conference.ID, tt.responseConferencecall.ID).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(tt.conference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.conference.CustomerID, conference.EventTypeConferenceUpdated, tt.conference)

			if err := h.Leaved(ctx, tt.conference, tt.referenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_leavedTypeConference(t *testing.T) {

	tests := []struct {
		name string

		conference *conference.Conference
	}{
		{
			"status is not terminating",

			&conference.Conference{
				ID: uuid.FromStringOrNil("9fa895c2-14c5-11ed-8f28-43e3f54564e5"),
			},
		},
		{
			"status is terminating and have conferencecalls",

			&conference.Conference{
				ID:     uuid.FromStringOrNil("c2d48b8c-14c5-11ed-a71a-b3ec4bcdf2eb"),
				Status: conference.StatusTerminating,
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("d6ea1330-14c5-11ed-a48e-471aed8a9cee"),
				},
			},
		},
		{
			"status is terminating and have no conferencecalls",

			&conference.Conference{
				ID:                uuid.FromStringOrNil("c2d48b8c-14c5-11ed-a71a-b3ec4bcdf2eb"),
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
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,

				conferencecallHandler: mockConferencecall,
			}
			ctx := context.Background()

			if tt.conference.Status == conference.StatusTerminating && len(tt.conference.ConferencecallIDs) == 0 {
				mockReq.EXPECT().CMV1ConfbridgeDelete(ctx, tt.conference.ConfbridgeID).Return(nil)
				mockDB.EXPECT().ConferenceEnd(ctx, tt.conference.ID).Return(nil)
				mockDB.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(tt.conference, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.conference.CustomerID, conference.EventTypeConferenceDeleted, tt.conference)
			}

			if err := h.leavedTypeConference(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_leavedTypeConnect(t *testing.T) {

	tests := []struct {
		name string

		conference *conference.Conference
	}{
		{
			"have no conferencecall",

			&conference.Conference{
				ID: uuid.FromStringOrNil("d2d98860-14c6-11ed-90d3-77d48833d689"),
			},
		},
		{
			"conferencecall",

			&conference.Conference{
				ID: uuid.FromStringOrNil("d2d98860-14c6-11ed-90d3-77d48833d689"),
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("4c33128e-14c8-11ed-b20a-275a81f394dd"),
				},
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
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,

				conferencecallHandler: mockConferencecall,
			}
			ctx := context.Background()

			if len(tt.conference.ConferencecallIDs) == 0 {
				// destroy
				mockReq.EXPECT().CMV1ConfbridgeDelete(ctx, tt.conference.ConfbridgeID).Return(nil)
				mockDB.EXPECT().ConferenceEnd(ctx, tt.conference.ID).Return(nil)
				mockDB.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(tt.conference, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.conference.CustomerID, conference.EventTypeConferenceDeleted, tt.conference)
			} else {
				// terminating
				mockDB.EXPECT().ConferenceGet(ctx, tt.conference.ID).Return(tt.conference, nil)
				mockDB.EXPECT().ConferenceSetStatus(ctx, tt.conference.ID, conference.StatusTerminating).Return(nil)
				mockReq.EXPECT().FMV1FlowDelete(ctx, tt.conference.FlowID).Return(&fmflow.Flow{}, nil)
				for _, conferencecallID := range tt.conference.ConferencecallIDs {
					mockConferencecall.EXPECT().Get(ctx, conferencecallID).Return(&conferencecall.Conferencecall{}, nil)
					mockReq.EXPECT().CMV1ConfbridgeCallKick(ctx, tt.conference.ConfbridgeID, gomock.Any()).Return(nil)
				}
			}

			if err := h.leavedTypeConnect(ctx, tt.conference); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
