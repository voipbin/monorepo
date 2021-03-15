package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestJoined(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := conferenceHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		cache:         mockCache,
	}

	type test struct {
		name             string
		channel          *channel.Channel
		bridge           *bridge.Bridge
		conference       *conference.Conference
		call             *call.Call
		bridgeJoining    *bridge.Bridge
		bridgeConference *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "13622bea-e58e-11ea-a93b-6f527ab3ee66",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				Type:       channel.TypeJoin,
			},
			&bridge.Bridge{
				ID:           "16cf5d02-e58e-11ea-a53e-479ddc675923",
				ConferenceID: uuid.FromStringOrNil("1a73b610-e58e-11ea-8307-9b9c2504b9bc"),
			},

			&conference.Conference{
				ID:       uuid.FromStringOrNil("a955a958-85be-11eb-bbe8-1382c0f7f7a0"),
				Type:     conference.TypeConference,
				BridgeID: "a97ac6d4-85be-11eb-ba4e-274dce825ac3",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("a99555ee-85be-11eb-9386-ef072600ebe3"),
				AsteriskID: "00:11:22:33:44:55",
				ChannelID:  "a9ad4a50-85be-11eb-9220-efd91279c796",
				Type:       call.TypeConference,
			},
			&bridge.Bridge{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "aa2d287e-85be-11eb-8866-3f7f4da14ca7",
			},
			&bridge.Bridge{
				AsteriskID:   "00:11:22:33:44:66",
				ID:           "a97ac6d4-85be-11eb-ba4e-274dce825ac3",
				ConferenceID: uuid.FromStringOrNil("bcbd248a-85be-11eb-a8c3-af32f458bca0"),
			},
		},
	}

	for _, tt := range tests {
		mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
		mockDB.EXPECT().CallSetConferenceID(gomock.Any(), tt.call.ID, tt.bridge.ConferenceID)
		mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
		mockNotify.EXPECT().CallUpdated(tt.call)
		mockDB.EXPECT().ConferenceAddCallID(gomock.Any(), tt.bridge.ConferenceID, tt.call.ID)

		if err := h.joined(tt.channel, tt.bridge); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}
