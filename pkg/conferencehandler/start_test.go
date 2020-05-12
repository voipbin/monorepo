package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
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
		call *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("59195730-934f-11ea-a50e-8f40de0b9810"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "e421f850-934f-11ea-b6d8-6f2393dd1cf0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstBridgeCreate(tt.call.AsteriskID, gomock.Any(), gomock.Any(), bridge.TypeMixing).Return(nil)
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockReq.EXPECT().AstChannelCreateSnoop(tt.call.AsteriskID, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirectionIn, channel.SnoopDirectionNone)
			mockReq.EXPECT().AstBridgeAddChannel(tt.call.AsteriskID, gomock.Any(), tt.call.ChannelID, "", false, false).Return(nil)
			mockReq.EXPECT().AstChannelAnswer(tt.call.AsteriskID, tt.call.ChannelID).Return(nil)

			h.startTypeEcho(tt.call)
		})
	}

}
