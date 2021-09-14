package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestLeave(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name       string
		conference *conference.Conference
		call       *call.Call
		channel    *channel.Channel
		bridge     *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				ID: uuid.FromStringOrNil("1b0f9e5e-9246-11ea-a764-53e61c9fef34"),
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("2dde2e70-9245-11ea-a1e5-1b4f44d33983"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "14ddefea-9246-11ea-bcc6-4bbba9c0b195",
				BridgeID:   "e6370b1c-9246-11ea-9fc8-533a3618523b",
			},
			&channel.Channel{
				ID:         "14ddefea-9246-11ea-bcc6-4bbba9c0b195",
				AsteriskID: "80:fa:5b:5e:da:81",
				BridgeID:   "e6370b1c-9246-11ea-9fc8-533a3618523b",
				Type:       channel.TypeJoin,
			},
			&bridge.Bridge{
				ID:         "e6370b1c-9246-11ea-9fc8-533a3618523b",
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelIDs: []string{
					"14ddefea-9246-11ea-bcc6-4bbba9c0b195",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.call.BridgeID).Return(tt.bridge, nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.call.ChannelID).Return(tt.channel, nil)
			mockReq.EXPECT().AstChannelHangup(tt.channel.AsteriskID, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(nil)

			h.Leave(tt.conference.ID, tt.call.ID)
		})
	}
}
