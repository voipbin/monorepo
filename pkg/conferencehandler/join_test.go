package conferencehandler

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
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

func Test_Join(t *testing.T) {

	tests := []struct {
		name string

		conferenceID  uuid.UUID
		referenceType conferencecall.ReferenceType
		referenceID   uuid.UUID

		responseConference     *conference.Conference
		responseConferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("aec64aa8-1340-11ed-b710-ef3b76998e5f"),
			conferencecall.ReferenceTypeCall,
			uuid.FromStringOrNil("af9a6c84-1340-11ed-89e7-270a42c938ad"),

			&conference.Conference{
				ID:           uuid.FromStringOrNil("aec64aa8-1340-11ed-b710-ef3b76998e5f"),
				Type:         conference.TypeConference,
				ConfbridgeID: uuid.FromStringOrNil("7d0bb11c-3e69-11ec-a38a-7b47fb83fb56"),
			},
			&conferencecall.Conferencecall{
				ID:            uuid.FromStringOrNil("d32d0170-1340-11ed-9e34-e703c37fefd3"),
				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("af9a6c84-1340-11ed-89e7-270a42c938ad"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := conferenceHandler{
				reqHandler:            mockReq,
				db:                    mockDB,
				cache:                 mockCache,
				notifyHandler:         mockNotify,
				conferencecallHandler: mockConferencecall,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.responseConference.ID).Return(tt.responseConference, nil)
			mockConferencecall.EXPECT().Create(ctx, tt.responseConference.CustomerID, tt.responseConference.ID, tt.referenceType, tt.referenceID).Return(tt.responseConferencecall, nil)
			mockReq.EXPECT().CMV1ConfbridgeCallAdd(ctx, tt.responseConference.ConfbridgeID, tt.referenceID).Return(nil)

			res, err := h.Join(ctx, tt.conferenceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConferencecall, res)
			}
		})
	}
}

func Test_JoinedConfbridge(t *testing.T) {

	tests := []struct {
		name string

		conferenceID uuid.UUID
		callID       uuid.UUID

		responseConferencecall *conferencecall.Conferencecall
		responseConference     *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("67c2dcb8-14c9-11ed-a5d0-2f904a6e3049"),
			uuid.FromStringOrNil("68004986-14c9-11ed-a7a1-679de3f88fb4"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("b37e5b6e-14c9-11ed-9e1d-bfc9e40ac768"),
			},
			&conference.Conference{
				ID:           uuid.FromStringOrNil("aec64aa8-1340-11ed-b710-ef3b76998e5f"),
				Type:         conference.TypeConference,
				ConfbridgeID: uuid.FromStringOrNil("7d0bb11c-3e69-11ec-a38a-7b47fb83fb56"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := conferenceHandler{
				reqHandler:            mockReq,
				db:                    mockDB,
				cache:                 mockCache,
				notifyHandler:         mockNotify,
				conferencecallHandler: mockConferencecall,
			}

			ctx := context.Background()

			mockConferencecall.EXPECT().UpdateStatusJoinedByReferenceID(ctx, tt.callID).Return(tt.responseConferencecall, nil)
			mockDB.EXPECT().ConferenceAddConferencecallID(ctx, tt.conferenceID, tt.responseConferencecall.ID).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.conferenceID).Return(tt.responseConference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConference.CustomerID, conference.EventTypeConferenceUpdated, tt.responseConference)

			if err := h.JoinedConfbridge(ctx, tt.conferenceID, tt.callID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
