package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_Terminating(t *testing.T) {

	tests := []struct {
		name string
		id   uuid.UUID

		responseConference *conference.Conference
	}{
		{
			name: "normal",
			id:   uuid.FromStringOrNil("9f5001a6-9482-11eb-956e-f7ead445bb7a"),

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9f5001a6-9482-11eb-956e-f7ead445bb7a"),
				},
				Type:              conference.TypeConference,
				Status:            conference.StatusProgressing,
				ConfbridgeID:      uuid.FromStringOrNil("4649cc0a-2086-11ec-8439-af4c561e87eb"),
				ConferencecallIDs: []uuid.UUID{},
			},
		},
		{
			name: "have 1 call",
			id:   uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("af79b3bc-9233-11ea-9b6f-2351dfdaf227"),
				},
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				ConfbridgeID: uuid.FromStringOrNil("3b5c6712-4368-11ec-a76b-0fcdde373728"),
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("2c4eaf4a-9482-11eb-9c2a-57de7ce9aed1"),
				},
			},
		},
		{
			name: "2 calls in the conference",
			id:   uuid.FromStringOrNil("fbf41954-0ab4-11eb-a22f-671a43bddb11"),

			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fbf41954-0ab4-11eb-a22f-671a43bddb11"),
				},
				Type:         conference.TypeConference,
				Status:       conference.StatusProgressing,
				ConfbridgeID: uuid.FromStringOrNil("3b8d65f6-4368-11ec-95eb-9751947b5cae"),
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("33a1af9a-9482-11eb-90d1-d7f2cf2288cb"),
					uuid.FromStringOrNil("6dfae364-9482-11eb-b11c-0f47944e2c54"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := conferenceHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockDB.EXPECT().ConferenceSetStatus(ctx, tt.id, conference.StatusTerminating).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConference.CustomerID, conference.EventTypeConferenceUpdated, tt.responseConference)

			for _, ccID := range tt.responseConference.ConferencecallIDs {
				mockReq.EXPECT().ConferenceV1ConferencecallKick(ctx, ccID).Return(&conferencecall.Conferencecall{}, nil)
			}

			res, err := h.Terminating(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseConference, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})

	}
}
