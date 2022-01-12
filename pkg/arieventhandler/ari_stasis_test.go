package arieventhandler

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	channel "gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestEventHandlerStasisStart(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockRequest,
		callHandler: mockCall,
	}

	tests := []struct {
		name  string
		event *ari.StasisStart

		expectChannelID  string
		expactStasisData map[string]string
		expectStasisname string
	}{
		{
			"normal",
			&ari.StasisStart{
				Event: ari.Event{
					Type:        ari.EventTypeStasisStart,
					Application: "voipbin",
					Timestamp:   "2020-04-25T00:27:18.342",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Args: ari.ArgsMap{
					"context": "call-in",
					"domain":  "sip-service.voipbin.net",
					"source":  "213.127.79.161",
				},
				Channel: ari.Channel{
					ID:           "1587774438.2390",
					Name:         "PJSIP/in-voipbin-00000948",
					Language:     "en",
					CreationTime: "2020-04-25T00:27:18.341",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "1234234324",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=8juJJyujlS,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
			},

			"1587774438.2390",
			map[string]string{
				"context": "call-in",
				"domain":  "sip-service.voipbin.net",
				"source":  "213.127.79.161",
			},
			"voipbin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			channel := &channel.Channel{
				Data: map[string]interface{}{},
			}
			mockDB.EXPECT().ChannelIsExist(tt.expectChannelID, gomock.Any()).Return(true)
			mockDB.EXPECT().ChannelSetStasisNameAndStasisData(gomock.Any(), tt.expectChannelID, tt.expectStasisname, tt.expactStasisData).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.expectChannelID).Return(channel, nil)
			mockCall.EXPECT().ARIStasisStart(gomock.Any(), channel, tt.expactStasisData).Return(nil)

			if err := h.EventHandlerStasisStart(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerStasisEnd(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockRequest,
		callHandler: mockCall,
	}

	tests := []struct {
		name  string
		event *ari.StasisEnd

		expectChannelID string
		expectStasis    string
	}{
		{
			"normal",
			&ari.StasisEnd{
				Event: ari.Event{
					Type:        ari.EventTypeStasisEnd,
					Application: "voipbin",
					Timestamp:   "2020-05-06T15:36:26.406",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Channel: ari.Channel{
					ID:           "1588779386.6019",
					Name:         "PJSIP/in-voipbin-00000bcb",
					Language:     "en",
					CreationTime: "2020-05-06T15:36:26.003",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Number: "287",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "0111049442037691478",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=522514233-794783407-1635452815,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=103.145.12.63",
					},
				},
			},
			"1588779386.6019",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetStasis(gomock.Any(), tt.expectChannelID, tt.expectStasis).Return(nil)

			if err := h.EventHandlerStasisEnd(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
