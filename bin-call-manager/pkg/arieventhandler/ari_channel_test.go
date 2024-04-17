package arieventhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
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

			&channel.Channel{
				ID: "1587842233.10218",
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

			mockChannel.EXPECT().ARIChannelStateChange(ctx, tt.event).Return(tt.responseChannel, nil)
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

func Test_EventHandlerChannelVarset(t *testing.T) {

	tests := []struct {
		name    string
		event   *ari.ChannelVarset
		channel *channel.Channel
		key     string
		value   interface{}
	}{
		{
			"normal",
			&ari.ChannelVarset{
				Channel: ari.Channel{
					ID: "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				},
				Variable: "test_key",
				Value:    "test_value",
			},
			&channel.Channel{
				ID:         "instance-asterisk-production-europe-west4-a-1-1597539159.90042",
				AsteriskID: "42:01:0a:a4:0f:ce",
			},
			"test_key",
			"test_value",
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
