package conferencecallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_Joined(t *testing.T) {

	tests := []struct {
		name string

		conferencecall *conferencecall.Conferencecall
	}{
		{
			"normal",

			&conferencecall.Conferencecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1ffe572-94d7-11ed-bee9-7f769a54345b"),
				},
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
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,

				conferenceHandler: mockConference,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConferencecallUpdate(ctx, tt.conferencecall.ID, map[conferencecall.Field]any{
				conferencecall.FieldStatus: conferencecall.StatusJoined,
			}.Return(nil)
			mockDB.EXPECT().ConferencecallGet(ctx, tt.conferencecall.ID.Return(tt.conferencecall, nil)
			mockNotify.EXPECT().PublishEvent(ctx, conferencecall.EventTypeConferencecallJoined, tt.conferencecall)

			mockConference.EXPECT().AddConferencecallID(ctx, tt.conferencecall.ConferenceID, tt.conferencecall.ID.Return(&conference.Conference{}, nil)

			res, err := h.Joined(ctx, tt.conferencecall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.conferencecall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.conferencecall, res)
			}
		})
	}
}
