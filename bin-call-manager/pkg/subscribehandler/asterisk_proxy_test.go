package subscribehandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/pkg/arieventhandler"
)

func Test_processEvent_AsteriskProxy_BridgeCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.BridgeCreated
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"BridgeCreated","timestamp":"2020-05-09T12:41:43.591+0000","bridge":{"id":"4625f6e6-6330-48ea-9d93-5cca714322b3","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":[],"creationtime":"2020-05-09T12:41:43.591+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`),
			},
			&ari.BridgeCreated{
				Event: ari.Event{
					Type:        ari.EventTypeBridgeCreated,
					Application: "voipbin",
					Timestamp:   "2020-05-09T12:41:43.591",
					AsteriskID:  "42:01:0a:a4:00:05",
				},

				Bridge: ari.Bridge{
					ID:   "4625f6e6-6330-48ea-9d93-5cca714322b3",
					Name: "echo",

					BridgeType:  "mixing",
					Technology:  "simple_bridge",
					BridgeClass: "stasis",
					Creator:     "Stasis",

					VideoMode: "none",

					Channels: []string{},

					CreationTime: "2020-05-09T12:41:43.591+0000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerBridgeCreated(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_BridgeDestroyed(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.BridgeDestroyed
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"BridgeDestroyed","timestamp":"2020-05-04T00:27:59.747+0000","bridge":{"id":"17174a5e-91f6-11ea-b637-fb223e63cedf","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":[],"creationtime":"2020-05-03T23:37:49.233+0000","video_mode":"talker"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`),
			},
			&ari.BridgeDestroyed{
				Event: ari.Event{
					Type:        ari.EventTypeBridgeDestroyed,
					Application: "voipbin",
					Timestamp:   "2020-05-04T00:27:59.747",
					AsteriskID:  "42:01:0a:a4:00:03",
				},

				Bridge: ari.Bridge{
					ID:   "17174a5e-91f6-11ea-b637-fb223e63cedf",
					Name: "test",

					BridgeType:  "mixing",
					Technology:  "simple_bridge",
					BridgeClass: "stasis",
					Creator:     "Stasis",

					VideoMode: "talker",

					Channels: []string{},

					CreationTime: "2020-05-03T23:37:49.233+0000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerBridgeDestroyed(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelCreated
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelCreated","timestamp":"2020-04-19T14:38:00.363+0000","channel":{"id":"1587307080.49","name":"PJSIP/in-voipbin-00000030","state":"Ring","caller":{"name":"","number":"68025"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T14:38:00.363+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelCreated(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelDestroyed(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelDestroyed
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelDestroyed","timestamp":"2020-04-19T17:02:58.651+0000","cause":42,"cause_txt":"Switching equipment congestion","channel":{"id":"1587315778.885","name":"PJSIP/in-voipbin-00000370","state":"Ring","caller":{"name":"","number":"804"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"00048323395006","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-19T17:02:58.651+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelDestroyed(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelStateChange(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelStateChange
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelStateChange","timestamp":"2020-04-25T19:17:13.786+0000","channel":{"id":"1587842233.10218","name":"PJSIP/in-voipbin-000026ee","state":"Up","caller":{"name":"","number":"586737682"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"46842002310","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=1491366011-850848062-1281392838,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=45.151.255.178"},"creationtime":"2020-04-25T19:17:13.585+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelStateChange(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelEnteredBridge(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelEnteredBridge
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelEnteredBridge","timestamp":"2020-05-09T10:36:04.595+0000","bridge":{"id":"a6abbe41-2a83-447b-8175-e52e5dea000f","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":["1589020563.4752","915befe7-7fff-490e-8432-ffe063d5c46d"],"creationtime":"2020-05-09T10:36:04.360+0000","video_mode":"talker"},"channel":{"id":"1589020563.4752","name":"PJSIP/in-voipbin-000008cc","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"999999","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-09T10:36:03.792+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`),
			},
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelEnteredBridge(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelLeftBridge(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelLeftBridge
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelLeftBridge","timestamp":"2020-05-09T10:53:39.181+0000","bridge":{"id":"a6abbe41-2a83-447b-8175-e52e5dea000f","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":["915befe7-7fff-490e-8432-ffe063d5c46d"],"creationtime":"2020-05-09T10:36:04.360+0000","video_mode":"talker"},"channel":{"id":"1589020563.4752","name":"PJSIP/in-voipbin-000008cc","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"999999","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=B0SUsFI1eo,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-09T10:36:03.792+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelLeftBridge(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelDtmfReceived(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelDtmfReceived
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"ChannelDtmfReceived","timestamp":"2020-05-20T18:43:32.809+0000","digit":"5","duration_ms":100,"channel":{"id":"1590000197.6557","name":"PJSIP/call-in-0000067a","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"918298437394","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=Oi9M1NhtxK,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-20T18:43:17.822+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelDtmfReceived(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ChannelVarset(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ChannelVarset
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"variable":"VB-DIRECTION","value":"","type":"ChannelVarset","timestamp":"2020-08-16T00:52:39.218+0000","channel":{"id":"instance-asterisk-production-europe-west4-a-1-1597539159.10032","name":"PJSIP/call-in-00004fb4","state":"Ring","caller":{"name":"","number":"7trunk"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"34967970028","priority":3,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77"},"creationtime":"2020-08-16T00:52:39.214+0000","language":"en"},"asterisk_id":"42:01:0a:a4:0f:ce","application":"voipbin"}`),
			},
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
				Variable: common.SIPHeaderDirection,
				Value:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerChannelVarset(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_ContactStatusChange(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.ContactStatusChange
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{ "application": "voipbin", "contact_info": { "uri": "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws", "roundtrip_usec": "0", "aor": "test11@test.trunk.voipbin.net", "contact_status": "NonQualified" }, "type": "ContactStatusChange", "endpoint": { "channel_ids": [], "resource": "test11@test.trunk.voipbin.net", "state": "online", "technology": "PJSIP" }, "timestamp": "2021-02-19T06:32:14.621+0000", "asterisk_id": "8e:86:e2:2c:a7:51"}`),
			},
			&ari.ContactStatusChange{
				Event: ari.Event{
					Type:        ari.EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621",
					AsteriskID:  "8e:86:e2:2c:a7:51",
				},
				Endpoint: ari.Endpoint{
					Resource:   "test11@test.trunk.voipbin.net",
					State:      ari.EndpointStateOnline,
					Technology: "PJSIP",
					ChannelIDs: []string{},
				},
				ContactInfo: ari.ContactInfo{
					AOR:           "test11@test.trunk.voipbin.net",
					URI:           "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws",
					RoundtripUsec: "0",
					ContactStatus: ari.ContactStatusTypeNonQualified,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerContactStatusChange(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_PlaybackFinished(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.PlaybackFinished
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{ "application": "voipbin", "asterisk_id": "42:01:0a:a4:0f:d0", "playback": { "id": "a41baef4-04b9-403d-a9f5-8ea82c8b1749", "language": "en", "media_uri": "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab", "state": "done", "target_uri": "channel:21dccba3-9792-4d57-904d-5260d57cd681" }, "timestamp": "2020-11-15T14:04:49.762+0000", "type": "PlaybackFinished"}`),
			},
			&ari.PlaybackFinished{
				Event: ari.Event{
					Type:        ari.EventTypePlaybackFinished,
					Application: "voipbin",
					Timestamp:   "2020-11-15T14:04:49.762",
					AsteriskID:  "42:01:0a:a4:0f:d0",
				},
				Playback: ari.Playback{
					ID:        "a41baef4-04b9-403d-a9f5-8ea82c8b1749",
					Language:  "en",
					MediaURI:  "sound:/mnt/media/tts/00ad7c95d14643f3f6f61d18acb039e7fedf05ab",
					State:     ari.PlaybackStateDone,
					TargetURI: "channel:21dccba3-9792-4d57-904d-5260d57cd681",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerPlaybackFinished(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_RecordingStarted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.RecordingStarted
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type": "RecordingStarted","timestamp": "2020-02-10T13:08:18.888","recording": {"name": "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z","format": "wav","state": "recording","target_uri": "channel:test_call"},"asterisk_id": "42:01:0a:84:00:12","application": "voipbin"}`),
			},
			&ari.RecordingStarted{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingStarted,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:18.888",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888Z",
					Format:          "wav",
					State:           "recording",
					SilenceDuration: 0,
					Duration:        0,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerRecordingStarted(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_RecordingFinished(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.RecordingFinished
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type": "RecordingFinished","timestamp": "2020-02-10T13:08:40.888","recording": {"name": "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888","format": "wav","state": "done","target_uri": "channel:test_call","duration": 351},"asterisk_id": "42:01:0a:84:00:12","application": "voipbin"}`),
			},
			&ari.RecordingFinished{
				Event: ari.Event{
					Type:        ari.EventTypeRecordingFinished,
					Application: "voipbin",
					Timestamp:   "2020-02-10T13:08:40.888",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				Recording: ari.RecordingLive{
					Name:            "call_3b16cef6-2b99-11eb-87eb-571ab4136611_2020-02-10T13:08:18.888",
					Format:          "wav",
					State:           "done",
					SilenceDuration: 0,
					Duration:        351,
					TalkingDuration: 0,
					TargetURI:       "channel:test_call",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerRecordingFinished(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_processEvent_AsteriskProxy_StasisStart(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.StasisStart
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"StasisStart","timestamp":"2020-04-25T00:27:18.342+0000","args":["context=call-in","domain=sip-service.voipbin.net","source=213.127.79.161"],"channel":{"id":"1587774438.2390","name":"PJSIP/in-voipbin-00000948","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"1234234324","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=8juJJyujlS,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-04-25T00:27:18.341+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerStasisStart(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ProcessEvent_AsteriskProxy_StasisEnd(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectEvent *ari.StasisEnd
	}{
		{
			"normal",

			&sock.Event{
				Publisher: "asterisk-proxy",
				Type:      "ari_event",
				DataType:  "application/json",
				Data:      []byte(`{"type":"StasisEnd","timestamp":"2020-05-06T15:36:26.406+0000","channel":{"id":"1588779386.6019","name":"PJSIP/in-voipbin-00000bcb","state":"Up","caller":{"name":"","number":"287"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0111049442037691478","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=522514233-794783407-1635452815,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=103.145.12.63"},"creationtime":"2020-05-06T15:36:26.003+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`),
			},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
			}

			mockARIEvent.EXPECT().EventHandlerStasisEnd(gomock.Any(), tt.expectEvent)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
