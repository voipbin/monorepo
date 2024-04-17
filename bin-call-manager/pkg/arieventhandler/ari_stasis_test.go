package arieventhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_EventHandlerStasisStart(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.StasisStart

		responseChannel *channel.Channel

		expectChannelID string
	}{
		{
			name: "normal",
			event: &ari.StasisStart{
				Event: ari.Event{
					Type:        ari.EventTypeStasisStart,
					Application: "voipbin",
					Timestamp:   "2020-04-25T00:27:18.342",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Args: ari.ArgsMap{
					"context_type": "call",
					"context":      "call-in",
					"domain":       "sip-service.voipbin.net",
					"source":       "213.127.79.161",
					"sip_call_id":  "8juJJyujlS",
					"sip_pai":      "",
					"sip_privacy":  "",
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
						AppData:  "voipbin,context_type=call,context=call-in,domain=sip-service.voipbin.net,source=213.127.79.161,sip_call_id=8juJJyujlS,sip_pai=,sip_privacy=",
					},
				},
			},

			responseChannel: &channel.Channel{
				ID:         "1587774438.2390",
				AsteriskID: "42:01:0a:a4:00:03",
				Name:       "PJSIP/in-voipbin-00000948",
				State:      ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					"context_type": "call",
					"context":      "call-in",
					"sip_call_id":  "8juJJyujlS",
					"sip_pai":      "",
					"sip_privacy":  "",
					"domain":       "sip-service.voipbin.net",
					"source":       "213.127.79.161",
				},
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectChannelID: "1587774438.2390",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockRequest := requesthandler.NewMockRequestHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := eventHandler{
				db:             mockDB,
				rabbitSock:     mockSock,
				reqHandler:     mockRequest,
				callHandler:    mockCall,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().Get(ctx, tt.expectChannelID).Return(tt.responseChannel, nil)
			mockChannel.EXPECT().ARIStasisStart(ctx, tt.event).Return(tt.responseChannel, nil)

			mockCall.EXPECT().ARIStasisStart(gomock.Any(), tt.responseChannel).Return(nil)

			if err := h.EventHandlerStasisStart(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerStasisEnd(t *testing.T) {
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockRequest := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			h := eventHandler{
				db:             mockDB,
				rabbitSock:     mockSock,
				reqHandler:     mockRequest,
				channelHandler: mockChannel,
				callHandler:    mockCall,
			}

			ctx := context.Background()

			mockChannel.EXPECT().UpdateStasisName(ctx, tt.expectChannelID, tt.expectStasis).Return(&channel.Channel{}, nil)

			if err := h.EventHandlerStasisEnd(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
