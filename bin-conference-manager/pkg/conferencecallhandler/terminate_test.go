package conferencecallhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

func Test_Terminate_kickable(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseConferencecall *conferencecall.Conferencecall
		responseConference     *conference.Conference

		expectRetryCount int
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("335a95ec-94da-11ed-8016-9317e1c7c8e7"),
			retryCount: defaultHealthCheckRetryMax + 1,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:          uuid.FromStringOrNil("335a95ec-94da-11ed-8016-9317e1c7c8e7"),
				Status:      conferencecall.StatusJoined,
				ReferenceID: uuid.FromStringOrNil("338cffc8-94da-11ed-95e4-2754365ed8a0"),
			},
			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("33db117c-94da-11ed-aa96-ab85ca5dc13a"),
				ConfbridgeID: uuid.FromStringOrNil("33b512ba-94da-11ed-8a65-27c4c9eb7665"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockConference := conferencehandler.NewMockConferenceHandler(mc)

			h := conferencecallHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				conferenceHandler: mockConference,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
			mockConference.EXPECT().Get(ctx, tt.responseConferencecall.ConferenceID).Return(tt.responseConference, nil)
			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.id, conferencecall.StatusLeaving).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, gomock.Any(), gomock.Any())
			mockReq.EXPECT().CallV1ConfbridgeCallKick(ctx, tt.responseConference.ConfbridgeID, tt.responseConferencecall.ReferenceID).Return(nil)

			res, err := h.Terminate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConferencecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_Terminate_unkickable(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseConferencecall *conferencecall.Conferencecall
		responseConference     *conference.Conference

		expectRetryCount int
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("bfa810e6-94db-11ed-a904-4f44913cdbb2"),
			retryCount: defaultHealthCheckRetryMax + 1,

			responseConferencecall: &conferencecall.Conferencecall{
				ID:          uuid.FromStringOrNil("bfa810e6-94db-11ed-a904-4f44913cdbb2"),
				Status:      conferencecall.StatusLeaving,
				ReferenceID: uuid.FromStringOrNil("bfc9be3a-94db-11ed-939f-d72160afc80c"),
			},
			responseConference: &conference.Conference{
				ID:           uuid.FromStringOrNil("c0219010-94db-11ed-b38d-13f659f67758"),
				ConfbridgeID: uuid.FromStringOrNil("bff40334-94db-11ed-8a5f-3f6b77b599f1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockConference := conferencehandler.NewMockConferenceHandler(mc)

			h := conferencecallHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				conferenceHandler: mockConference,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallGet(ctx, tt.id).Return(tt.responseConferencecall, nil)
			mockConference.EXPECT().Get(ctx, tt.responseConferencecall.ConferenceID).Return(tt.responseConference, nil)

			res, err := h.Terminate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConferencecall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_isKickable(t *testing.T) {

	tests := []struct {
		name string

		conferencecall *conferencecall.Conferencecall

		expectRes bool
	}{
		{
			name: "pass all validation. must return true",

			conferencecall: &conferencecall.Conferencecall{
				Status: conferencecall.StatusJoined,
			},

			expectRes: true,
		},
		{
			name: "conferencecall has invalid status. leaved",

			conferencecall: &conferencecall.Conferencecall{
				Status: conferencecall.StatusLeaved,
			},

			expectRes: false,
		},
		{
			name: "conferencecall has invalid status. leaving",

			conferencecall: &conferencecall.Conferencecall{
				Status: conferencecall.StatusLeaving,
			},

			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferencecallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			res := h.isKickable(ctx, tt.conferencecall)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Terminated(t *testing.T) {

	tests := []struct {
		name string

		conferencecall *conferencecall.Conferencecall

		responseConference *conference.Conference
		expectRes          *conferencecall.Conferencecall
	}{
		{
			name: "normal",

			conferencecall: &conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("cf8520fc-94dc-11ed-bacf-13ff11224cdb"),
				ConferenceID: uuid.FromStringOrNil("dcae37aa-94dc-11ed-a462-07f286738971"),
			},

			responseConference: &conference.Conference{
				ConfbridgeID: uuid.FromStringOrNil("dcae37aa-94dc-11ed-a462-07f286738971"),
			},
			expectRes: &conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("cf8520fc-94dc-11ed-bacf-13ff11224cdb"),
				ConferenceID: uuid.FromStringOrNil("dcae37aa-94dc-11ed-a462-07f286738971"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockConference := conferencehandler.NewMockConferenceHandler(mc)

			h := conferencecallHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				conferenceHandler: mockConference,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdateStatus(ctx, tt.conferencecall.ID, conferencecall.StatusLeaved).Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.conferencecall.ID).Return(tt.conferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, gomock.Any(), gomock.Any())
			mockConference.EXPECT().RemoveConferencecallID(ctx, tt.conferencecall.ConferenceID, tt.conferencecall.ID).Return(tt.responseConference, nil)

			res, err := h.Terminated(ctx, tt.conferencecall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
