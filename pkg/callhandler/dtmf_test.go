package callhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestDTMFReceived(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name     string
		channel  *channel.Channel
		call     *call.Call
		digit    string
		duration int
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("b2a45cf6-9ace-11ea-9354-4baa7f3ad331"),
				ChannelID:  "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
				Action: action.Action{
					Type: action.TypeEcho,
				},
			},
			"4",
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstChannelDTMF(tt.call.AsteriskID, tt.call.ChannelID, tt.digit, tt.duration, 0, 0, 0)

			h.DTMFReceived(tt.channel, tt.digit, tt.duration)
		})
	}

}
