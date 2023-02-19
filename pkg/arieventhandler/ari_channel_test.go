package arieventhandler

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_EventHandlerChannelCreated(t *testing.T) {
	tests := []struct {
		name  string
		event *ari.ChannelCreated

		expectID                string
		expectAsteriskID        string
		expectName              string
		expectChannelType       channel.Type
		expectTech              channel.Tech
		expectSIPCallID         string
		expectSIPTransport      channel.SIPTransport
		expectSourceName        string
		expectSourceNumber      string
		expectDestinationName   string
		expectDestinationNumber string
		expectState             ari.ChannelState
		expectData              map[string]interface{}
		expectStasisName        string
		expectStasisData        map[string]string
		expectBridgeID          string
		expectPlaybackID        string
		expectDialResult        string
		expectHangupCause       ari.ChannelCause
		expectDirection         channel.Direction

		responseChannel *channel.Channel
	}{
		{
			"normal",

			&ari.ChannelCreated{
				Event: ari.Event{
					Type:        ari.EventTypeChannelCreated,
					Application: "voipbin",
					Timestamp:   "2020-04-19T14:38:00.363",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1587307080.49",
					Name:         "PJSIP/in-voipbin-00000030",
					Language:     "en",
					CreationTime: "2020-04-19T14:38:00.363",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Number: "68025",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "011441332323027",
						Priority: 1,
					},
				},
			},

			"1587307080.49",
			"42:01:0a:a4:00:05",
			"PJSIP/in-voipbin-00000030",
			channel.TypeNone,
			channel.TechPJSIP,

			"",
			channel.SIPTransportNone,

			"",
			"68025",
			"",
			"011441332323027",

			ari.ChannelStateRing,
			map[string]interface{}{},

			"",
			map[string]string{},

			"",
			"",
			"",
			ari.ChannelCauseUnknown,
			channel.DirectionNone,

			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "1587307080.49",
			},
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

			mockChannel.EXPECT().Create(
				ctx,
				tt.expectID,
				tt.expectAsteriskID,
				tt.expectName,
				tt.expectChannelType,
				tt.expectTech,
				tt.expectSourceName,
				tt.expectSourceNumber,
				tt.expectDestinationName,
				tt.expectDestinationNumber,
				tt.expectState,
			).Return(tt.responseChannel, nil)

			if err := h.EventHandlerChannelCreated(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelDestroyed(t *testing.T) {

	tests := []struct {
		name string

		event *ari.ChannelDestroyed

		responseChannel *channel.Channel
		expectChannelID string
		expectHangup    ari.ChannelCause
	}{
		{
			"normal",
			&ari.ChannelDestroyed{
				Event: ari.Event{
					Type:        ari.EventTypeChannelDestroyed,
					Application: "voipbin",
					Timestamp:   "2020-04-19T17:02:58.651",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Channel: ari.Channel{
					ID:           "1587315778.885",
					Name:         "PJSIP/in-voipbin-00000370",
					Language:     "en",
					CreationTime: "2020-04-19T17:02:58.651",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Number: "804",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "00048323395006",
						Priority: 1,
					},
				},
				CauseTxt: "Switching equipment congestion",
				Cause:    42,
			},

			&channel.Channel{
				ID:   "1587315778.885",
				Type: channel.TypeCall,
			},
			"1587315778.885",
			ari.ChannelCauseSwitchCongestion,
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

			mockChannel.EXPECT().Delete(ctx, tt.expectChannelID, tt.expectHangup).Return(tt.responseChannel, nil)
			mockCall.EXPECT().ARIChannelDestroyed(ctx, tt.responseChannel).Return(nil)

			if err := h.EventHandlerChannelDestroyed(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelStateChange(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.ChannelStateChange

		responseChannel *channel.Channel

		expectChannelID string
		expectState     ari.ChannelState
	}{
		{
			"normal",
			&ari.ChannelStateChange{
				Event: ari.Event{
					Type:        ari.EventTypeChannelStateChange,
					Application: "voipbin",
					Timestamp:   "2020-04-25T19:17:13.786",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1587842233.10218",
					Name:         "PJSIP/in-voipbin-000026ee",
					Language:     "en",
					CreationTime: "2020-04-25T19:17:13.585",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Number: "586737682",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "46842002310",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=1491366011-850848062-1281392838,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=45.151.255.178",
					},
				},
			},

			&channel.Channel{},

			"1587842233.10218",
			ari.ChannelStateUp,
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

			mockChannel.EXPECT().UpdateState(ctx, tt.expectChannelID, tt.expectState).Return(tt.responseChannel, nil)
			mockCall.EXPECT().ARIChannelStateChange(ctx, tt.responseChannel).Return(nil)

			if err := h.EventHandlerChannelStateChange(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelEnteredBridge(t *testing.T) {

	type test struct {
		name           string
		event          *ari.ChannelEnteredBridge
		channel        *channel.Channel
		responseBridge *bridge.Bridge
	}

	tests := []test{
		{
			"channel type is join type",
			&ari.ChannelEnteredBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelEnteredBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:36:04.595",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "a6abbe41-2a83-447b-8175-e52e5dea000f",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"1589020563.4752", "915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			}, &channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
				Type:       channel.TypeJoin,
			},
			&bridge.Bridge{
				ID:         "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID: "42:01:0a:a4:00:05",
			},
		},
		{
			"channel type is call type",
			&ari.ChannelEnteredBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelEnteredBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:36:04.595",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "aedc915a-3920-11ec-b800-8bda16e1ef0c",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"1589020563.4752", "915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
				Type:       channel.TypeCall,
			},
			&bridge.Bridge{
				ID:         "aedc915a-3920-11ec-b800-8bda16e1ef0c",
				AsteriskID: "42:01:0a:a4:00:05",
			},
		},
		{
			"channel type is confbridge type",
			&ari.ChannelEnteredBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelEnteredBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:36:04.595",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "eb7c0136-3920-11ec-b99e-e3e6a65976f5",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"1589020563.4752", "915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
				Type:       channel.TypeConfbridge,
			},
			&bridge.Bridge{
				ID:         "eb7c0136-3920-11ec-b99e-e3e6a65976f5",
				AsteriskID: "42:01:0a:a4:00:05",
			},
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
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := eventHandler{
				db:                mockDB,
				rabbitSock:        mockSock,
				reqHandler:        mockRequest,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
				channelHandler:    mockChannel,
				bridgeHandler:     mockBridge,
			}
			ctx := context.Background()

			mockChannel.EXPECT().UpdateBridgeID(ctx, tt.channel.ID, tt.responseBridge.ID).Return(tt.channel, nil)
			mockBridge.EXPECT().AddChannelID(ctx, tt.responseBridge.ID, tt.channel.ID).Return(tt.responseBridge, nil)

			if tt.channel.Type == channel.TypeConfbridge {
				mockConfbridge.EXPECT().ARIChannelEnteredBridge(gomock.Any(), tt.channel, tt.responseBridge).Return(nil)
			}

			if err := h.EventHandlerChannelEnteredBridge(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelLeftBridge(t *testing.T) {

	tests := []struct {
		name           string
		event          *ari.ChannelLeftBridge
		channel        *channel.Channel
		responseBridge *bridge.Bridge
	}{
		{
			"channel left from the conference bridge",
			&ari.ChannelLeftBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelLeftBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:53:39.181",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "a6abbe41-2a83-447b-8175-e52e5dea000f",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&bridge.Bridge{
				ID:            "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID:    "42:01:0a:a4:00:05",
				ReferenceType: bridge.ReferenceTypeConfbridge,
			},
		},
		{
			"channel left from the conference snoop bridge",
			&ari.ChannelLeftBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelLeftBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:53:39.181",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "a6abbe41-2a83-447b-8175-e52e5dea000f",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&bridge.Bridge{
				ID:            "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID:    "42:01:0a:a4:00:05",
				ReferenceType: bridge.ReferenceTypeConfbridgeSnoop,
			},
		},
		{
			"channel left from the call bridge",
			&ari.ChannelLeftBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelLeftBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:53:39.181",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "a6abbe41-2a83-447b-8175-e52e5dea000f",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&bridge.Bridge{
				ID:            "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID:    "42:01:0a:a4:00:05",
				ReferenceType: bridge.ReferenceTypeCall,
			},
		},
		{
			"channel left from the call snoop bridge",
			&ari.ChannelLeftBridge{
				Event: ari.Event{
					Type:        ari.EventTypeChannelLeftBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-09T10:53:39.181",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1589020563.4752",
					Name:         "PJSIP/in-voipbin-000008cc",
					Language:     "en",
					CreationTime: "2020-05-09T10:36:03.792",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "999999",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Bridge: ari.Bridge{
					ID:           "a6abbe41-2a83-447b-8175-e52e5dea000f",
					Name:         "echo",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					VideoMode:    "talker",
					Channels:     []string{"915befe7-7fff-490e-8432-ffe063d5c46d"},
					CreationTime: "2020-05-09T10:36:04.360+0000",
				},
			},
			&channel.Channel{
				ID:         "1589020563.4752",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			&bridge.Bridge{
				ID:            "a6abbe41-2a83-447b-8175-e52e5dea000f",
				AsteriskID:    "42:01:0a:a4:00:05",
				ReferenceType: bridge.ReferenceTypeCallSnoop,
			},
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
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := eventHandler{
				db:                mockDB,
				rabbitSock:        mockSock,
				reqHandler:        mockRequest,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
				channelHandler:    mockChannel,
				bridgeHandler:     mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().UpdateBridgeID(ctx, tt.event.Channel.ID, "").Return(tt.channel, nil)
			mockBridge.EXPECT().RemoveChannelID(ctx, tt.event.Bridge.ID, tt.event.Channel.ID).Return(tt.responseBridge, nil)

			switch tt.responseBridge.ReferenceType {
			case bridge.ReferenceTypeConfbridge, bridge.ReferenceTypeConfbridgeSnoop:
				mockConfbridge.EXPECT().ARIChannelLeftBridge(ctx, tt.channel, tt.responseBridge).Return(nil)

			case bridge.ReferenceTypeCall, bridge.ReferenceTypeCallSnoop:
				mockCall.EXPECT().ARIChannelLeftBridge(ctx, tt.channel, tt.responseBridge).Return(nil)
			}

			if err := h.EventHandlerChannelLeftBridge(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelDtmfReceived(t *testing.T) {

	tests := []struct {
		name            string
		event           *ari.ChannelDtmfReceived
		responseChannel *channel.Channel
		expectDigit     string
		expectDuration  int
	}{
		{
			"normal",
			&ari.ChannelDtmfReceived{
				Event: ari.Event{
					Type:        ari.EventTypeChannelDtmfReceived,
					Application: "voipbin",
					Timestamp:   "2020-05-20T18:43:32.809",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Channel: ari.Channel{
					ID:           "1590000197.6557",
					Name:         "PJSIP/call-in-0000067a",
					Language:     "en",
					CreationTime: "2020-05-20T18:43:17.822",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "call-in",
						Exten:    "918298437394",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=Oi9M1NhtxK,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
				Digit:    "5",
				Duration: 100,
			},

			&channel.Channel{
				ID:         "1590000197.6557",
				AsteriskID: "42:01:0a:a4:00:03",
			},
			"5",
			100,
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

			mockChannel.EXPECT().Get(ctx, tt.responseChannel.ID).Return(tt.responseChannel, nil)
			mockCall.EXPECT().ARIChannelDtmfReceived(ctx, tt.responseChannel, tt.expectDigit, tt.expectDuration).Return(nil)

			if err := h.EventHandlerChannelDtmfReceived(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelVarsetDirection(t *testing.T) {

	tests := []struct {
		name      string
		event     *ari.ChannelVarset
		channel   *channel.Channel
		direction channel.Direction
	}{
		{
			"None",
			&ari.ChannelVarset{
				Event: ari.Event{
					Type:        ari.EventTypeChannelVarset,
					Application: "voipbin",
					Timestamp:   "2020-08-16T00:52:39.218",
					AsteriskID:  "42:01:0a:a4:0f:ce",
				},
				Channel: ari.Channel{
					ID:           "instance-asterisk-production-europe-west4-a-1-1597539159.10032",
					Name:         "PJSIP/call-in-00004fb4",
					Language:     "en",
					CreationTime: "2020-08-16T00:52:39.214",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Number: "7trunk",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "call-in",
						Exten:    "34967970028",
						Priority: 3,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77",
					},
				},
				Variable: "VB-DIRECTION",
				Value:    "",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.10032",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.DirectionNone,
		},
		{
			"incoming",
			&ari.ChannelVarset{
				Event: ari.Event{
					Type:        ari.EventTypeChannelVarset,
					Application: "voipbin",
					Timestamp:   "2020-08-16T00:52:39.218",
					AsteriskID:  "42:01:0a:a4:0f:ce",
				},
				Channel: ari.Channel{
					ID:           "instance-asterisk-production-europe-west4-a-1-1597539159.10042",
					Name:         "PJSIP/call-in-00004fb4",
					Language:     "en",
					CreationTime: "2020-08-16T00:52:39.214",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Number: "7trunk",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "call-in",
						Exten:    "34967970028",
						Priority: 3,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77",
					},
				},
				Variable: "VB-DIRECTION",
				Value:    "incoming",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.10042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.DirectionIncoming,
		},
		{
			"outgoing",
			&ari.ChannelVarset{
				Event: ari.Event{
					Type:        ari.EventTypeChannelVarset,
					Application: "voipbin",
					Timestamp:   "2020-08-16T00:52:39.218",
					AsteriskID:  "42:01:0a:a4:0f:ce",
				},
				Channel: ari.Channel{
					ID:           "instance-asterisk-production-europe-west4-a-1-1597539159.11042",
					Name:         "PJSIP/call-in-00004fb4",
					Language:     "en",
					CreationTime: "2020-08-16T00:52:39.214",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Number: "7trunk",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "call-in",
						Exten:    "34967970028",
						Priority: 3,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77",
					},
				},
				Variable: "VB-DIRECTION",
				Value:    "outgoing",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.11042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.DirectionOutgoing,
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

			mockChannel.EXPECT().SetDirection(ctx, tt.channel.ID, tt.direction).Return(nil)
			if err := h.EventHandlerChannelVarset(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelVarsetSIPTransport(t *testing.T) {

	tests := []struct {
		name         string
		event        *ari.ChannelVarset
		channel      *channel.Channel
		sipTransport channel.SIPTransport
	}{
		{
			"tcp type",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_TRANSPORT",
				Value:    "tcp",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.SIPTransportTCP,
		},
		{
			"udp type",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_TRANSPORT",
				Value:    "udp",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.SIPTransportUDP,
		},
		{
			"tls type",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_TRANSPORT",
				Value:    "tls",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.SIPTransportTLS,
		},
		{
			"wss type",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_TRANSPORT",
				Value:    "wss",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			channel.SIPTransportWSS,
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

			mockChannel.EXPECT().SetSIPTransport(ctx, tt.channel.ID, tt.sipTransport).Return(nil)
			if err := h.EventHandlerChannelVarset(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelVarsetSIPCallID(t *testing.T) {

	tests := []struct {
		name      string
		event     *ari.ChannelVarset
		channel   *channel.Channel
		sipCallID string
	}{
		{
			"normal",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_CALLID",
				Value:    "d224d2a8-e471-11ea-93f2-e302e86922cc",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			"d224d2a8-e471-11ea-93f2-e302e86922cc",
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

			mockChannel.EXPECT().SetSIPCallID(ctx, tt.channel.ID, tt.sipCallID).Return(nil)
			if err := h.EventHandlerChannelVarset(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelVarsetSIPDataItem(t *testing.T) {

	tests := []struct {
		name    string
		event   *ari.ChannelVarset
		channel *channel.Channel
		key     string
		value   interface{}
	}{
		{
			"test VB-SIP_PAI",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_PAI",
				Value:    "tel:+31616818985",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			"sip_pai",
			"tel:+31616818985",
		},
		{
			"test VB-SIP_PRIVACY",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "VB-SIP_PRIVACY",
				Value:    "id",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			"sip_privacy",
			"id",
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

			mockChannel.EXPECT().SetDataItem(ctx, tt.channel.ID, tt.key, tt.value).Return(nil)
			if err := h.EventHandlerChannelVarset(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerChannelVarsetVBTYPE(t *testing.T) {

	tests := []struct {
		name    string
		event   *ari.ChannelVarset
		channel *channel.Channel
		cType   channel.Type
	}{
		{
			"None",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-e42907e8-e549-11ea-8744-dfde41483063",
				},
				Variable: "VB-TYPE",
				Value:    "",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-e42907e8-e549-11ea-8744-dfde41483063",
				AsteriskID: "42:01:0a:a4:0f:ce",
				Type:       channel.TypeNone,
			},
			channel.TypeNone,
		},
		{
			"call",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-dac9f91e-e549-11ea-9491-e315e9ebdc0a",
				},
				Variable: "VB-TYPE",
				Value:    "call",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-dac9f91e-e549-11ea-9491-e315e9ebdc0a",
				AsteriskID: "42:01:0a:a4:0f:ce",
				Type:       channel.TypeCall,
			},
			channel.TypeCall,
		},
		{
			"conf",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-cf436148-e549-11ea-813a-036e5febe4ac",
				},
				Variable: "VB-TYPE",
				Value:    "confbridge",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-cf436148-e549-11ea-813a-036e5febe4ac",
				AsteriskID: "42:01:0a:a4:0f:ce",
				Type:       channel.TypeConfbridge,
			},
			channel.TypeConfbridge,
		},
		{
			"join",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-c39d269e-e549-11ea-856a-db4440c2c2fe",
				},
				Variable: "VB-TYPE",
				Value:    "join",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-c39d269e-e549-11ea-856a-db4440c2c2fe",
				AsteriskID: "42:01:0a:a4:0f:ce",
				Type:       channel.TypeJoin,
			},
			channel.TypeJoin,
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

			mockChannel.EXPECT().SetType(ctx, tt.channel.ID, tt.cType).Return(nil)
			if err := h.EventHandlerChannelVarset(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
