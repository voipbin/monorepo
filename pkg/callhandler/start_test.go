package callhandler

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestGetService(t *testing.T) {
	type test struct {
		name          string
		channel       *channel.Channel
		expectService service
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				Data: map[string]interface{}{
					"CONTEXT": contextIncomingVoip,
					"DOMAIN":  domainEcho,
				},
			},
			svcEcho,
		},
		{
			"normal",
			&channel.Channel{
				Data: map[string]interface{}{
					"CONTEXT": contextIncomingCall,
					"DOMAIN":  domainEcho,
				},
			},
			svcEcho,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := getService(tt.channel)
			if service != tt.expectService {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectService, service)
			}
		})
	}
}

func TestServiceEchoStart(t *testing.T) {
	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				AsteriskID: "80:fa:5b:5e:da:81",
				Name:       "PJSIP/in-voipbin-00000948",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("6611bf7e-92e4-11ea-b658-8313e9bd28f8"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "f82007c4-92e2-11ea-a3e2-138ed7e90501",
				Type:       call.TypeEcho,
				Direction:  call.DirectionIncoming,
			},
		},
	}

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			option := action.OptionEcho{
				Duration: 180,
				DTMF:     true,
			}
			opt, err := json.Marshal(option)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			action := &action.Action{
				ID:     actionBegin,
				Type:   action.TypeEcho,
				Option: opt,
				Next:   actionEnd,
			}

			mockReq.EXPECT().AstChannelVariableSet(tt.channel.AsteriskID, tt.channel.ID, "TIMEOUT(absolute)", defaultMaxTimeoutEcho).Return(nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetFlowID(gomock.Any(), gomock.Any(), uuid.Nil).Return(nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), gomock.Any(), action).Return(nil)
			mockConf.EXPECT().Start(conference.TypeEcho, gomock.Any())

			h.serviceEchoStart(tt.channel)
		})
	}
}
