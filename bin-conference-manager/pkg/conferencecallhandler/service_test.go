package conferencecallhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ServiceStart(t *testing.T) {

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("335a95ec-94da-11ed-8016-9317e1c7c8e7"),
				},
				Status:      conferencecall.StatusJoined,
				ReferenceID: uuid.FromStringOrNil("338cffc8-94da-11ed-95e4-2754365ed8a0"),
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("33db117c-94da-11ed-aa96-ab85ca5dc13a"),
				},
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
			mockDB.EXPECT().ConferencecallUpdate(ctx, tt.id, map[conferencecall.Field]any{
				conferencecall.FieldStatus: conferencecall.StatusLeaving,
			}).Return(nil)
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
