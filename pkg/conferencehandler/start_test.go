package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestStartTypeEcho(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name string
		cf   *conference.Conference
		call *call.Call
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				Type: conference.TypeEcho,
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("59195730-934f-11ea-a50e-8f40de0b9810"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "e421f850-934f-11ea-b6d8-6f2393dd1cf0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstBridgeCreate(tt.call.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing}).Return(nil)
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionIn, channel.SnoopDirectionNone)
			mockReq.EXPECT().AstBridgeAddChannel(tt.call.AsteriskID, gomock.Any(), tt.call.ChannelID, "", false, false).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)

			h.startTypeEcho(tt.cf, tt.call)
		})
	}
}

func TestStartTypeConference(t *testing.T) {
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
		reqConf    *conference.Conference
		call       *call.Call
		conference *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				Type: conference.TypeConference,
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("0f6dcf3e-a412-11ea-8197-f7feaeb4c806"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "13fecf8a-a412-11ea-9d1b-3b97be1ef739",
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("325b9d9a-a413-11ea-a6d7-ef53544faeb3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstBridgeCreate(requesthandler.AsteriskIDConference, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeVideoSFU}).Return(nil)
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.conference, nil)

			if tt.reqConf.Timeout > 0 {
				mockReq.EXPECT().CallConferenceTerminate(gomock.Any(), "timeout", tt.reqConf.Timeout*1000).Return(nil)
			}

			h.startTypeConference(tt.reqConf, tt.call)
		})
	}
}
