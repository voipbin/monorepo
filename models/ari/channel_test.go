package ari

import (
	"reflect"
	"testing"
)

func TestParseChannel(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectParse *Channel
	}

	tests := []test{
		{
			"test normal",
			`{"id":"1589706755.0","name":"PJSIP/call-in-00000000","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"8872616","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=NfD0bqG~ys,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-17T09:12:35.988+0000","language":"en"}`,
			&Channel{
				ID:           "1589706755.0",
				Name:         "PJSIP/call-in-00000000",
				Language:     "en",
				CreationTime: "2020-05-17T09:12:35.988",
				State:        ChannelStateUp,
				Caller: CallerID{
					Name:   "tttt",
					Number: "pchero",
				},
				Connected: CallerID{},
				Dialplan: DialplanCEP{
					Context:  "call-in",
					Exten:    "8872616",
					Priority: 2,
					AppName:  "Stasis",
					AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=NfD0bqG~ys,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
				},
			},
		},
		{
			"unicast rtp",
			`{"id": "asterisk-call-5765d977d8-c4k5q-1629605410.6626","name": "UnicastRTP/127.0.0.1:5090-0x7f6d54035300","state": "Down","caller": {"name": "","number": ""},"connected": {"name": "","number": ""},"accountcode": "","dialplan": {"context": "default","exten": "s","priority": 1,"app_name": "AppDial2","app_data": "(Outgoing Line)"},"creationtime": "2021-08-22T04:10:10.331+0000","language": "en","channelvars": {"UNICASTRTP_LOCAL_PORT": "10492","UNICASTRTP_LOCAL_ADDRESS": "127.0.0.1"}}`,
			&Channel{
				ID:           "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				Name:         "UnicastRTP/127.0.0.1:5090-0x7f6d54035300",
				Language:     "en",
				CreationTime: "2021-08-22T04:10:10.331",
				State:        ChannelStateDown,
				Caller:       CallerID{},
				Connected:    CallerID{},
				Dialplan: DialplanCEP{
					Context:  "default",
					Exten:    "s",
					Priority: 1,
					AppName:  "AppDial2",
					AppData:  "(Outgoing Line)",
				},
				ChannelVars: map[string]string{
					"UNICASTRTP_LOCAL_PORT":    "10492",
					"UNICASTRTP_LOCAL_ADDRESS": "127.0.0.1",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := ParseChannel([]byte(tt.message))
			if err != nil {
				t.Errorf("Wront match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectParse, channel) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectParse, channel)
			}
		})
	}
}

func TestParseChannelCreated(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *Event
		expectParse *ChannelCreated
	}

	tests := []test{
		{
			"test normal",
			`{"type":"ChannelCreated","timestamp":"2020-04-24T06:07:48.202+0000","channel":{"id":"1587708468.7504","name":"PJSIP/in-voipbin-00001d4b","state":"Ring","caller":{"name":"","number":"5"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"701146842002329","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-24T06:07:48.202+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			&Event{
				Type:        EventTypeChannelCreated,
				Application: "voipbin",
				Timestamp:   "2020-04-24T06:07:48.202",
				AsteriskID:  "42:01:0a:a4:00:05",
			},
			&ChannelCreated{
				Event: Event{
					Type:        EventTypeChannelCreated,
					Application: "voipbin",
					Timestamp:   "2020-04-24T06:07:48.202",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: Channel{
					ID:           "1587708468.7504",
					Name:         "PJSIP/in-voipbin-00001d4b",
					State:        "Ring",
					Language:     "en",
					CreationTime: "2020-04-24T06:07:48.202",
					Caller: CallerID{
						Name:   "",
						Number: "5",
					},
					Connected: CallerID{
						Name:   "",
						Number: "",
					},
					AccountCode: "",
					Dialplan: DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "701146842002329",
						Priority: 1,
						AppName:  "",
						AppData:  "",
					},
				},
			},
		},
		{
			name:    "has single string args",
			message: `{"type":"ChannelCreated","timestamp":"2024-03-25T07:18:48.378+0000","channel":{"id":"asterisk-call-f4df6d4d7-clwdl-1711351128.83","name":"AudioSocket/10.96.1.162:10000-5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547","state":"Down","protocol_id":"","caller":{"name":"","number":""},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"default","exten":"s","priority":1,"app_name":"","app_data":""},"creationtime":"2024-03-25T07:18:48.378+0000","language":"en"},"asterisk_id":"be:c6:98:a4:21:17","application":"voipbin"}`,
			expectEvent: &Event{
				Type:        EventTypeChannelCreated,
				Application: "voipbin",
				Timestamp:   "2024-03-25T07:18:48.378",
				AsteriskID:  "be:c6:98:a4:21:17",
			},
			expectParse: &ChannelCreated{
				Event: Event{
					Type:        EventTypeChannelCreated,
					Application: "voipbin",
					Timestamp:   "2024-03-25T07:18:48.378",
					AsteriskID:  "be:c6:98:a4:21:17",
				},
				Channel: Channel{
					ID:           "asterisk-call-f4df6d4d7-clwdl-1711351128.83",
					Name:         "AudioSocket/10.96.1.162:10000-5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547",
					State:        "Down",
					Language:     "en",
					CreationTime: "2024-03-25T07:18:48.378",
					AccountCode:  "",
					Dialplan: DialplanCEP{
						Context:  "default",
						Exten:    "s",
						Priority: 1,
						AppName:  "",
						AppData:  "",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(event, tt.expectEvent) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, event)
			}

			e := evt.(*ChannelCreated)
			if !reflect.DeepEqual(e, tt.expectParse) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectParse, e)
			}
		})
	}
}

func TestParseChannelDestroyed(t *testing.T) {
	m := `{"type":"ChannelDestroyed","timestamp":"2020-04-10T01:09:46.681+0000","cause":18,"cause_txt":"No user responding","channel":{"id":"1586480922.6170","name":"PJSIP/in-voipbin-00000c0e","state":"Up","caller":{"name":"","number":"3400001"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":4,"app_name":"","app_data":""},"creationtime":"2020-04-10T01:08:42.678+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`

	_, evt, err := Parse([]byte(m))
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	e := evt.(*ChannelDestroyed)
	if reflect.TypeOf(e.Channel) != reflect.TypeOf(Channel{}) {
		t.Errorf("Wrong")
	}

	if e.Cause != 18 {
		t.Errorf("Wrong match. expect:18, got: %d", e.Cause)
	}
	if e.CauseTxt != "No user responding" {
		t.Errorf("Wrong match. expect: No user responding, got: %s", e.CauseTxt)
	}
}

func TestParseChannelHangupRequest(t *testing.T) {
	m := `{"cause":18,"type":"ChannelHangupRequest","timestamp":"2020-04-10T01:09:47.437+0000","channel":{"id":"1586480923.6171","name":"PJSIP/in-voipbin-00000c0f","state":"Up","caller":{"name":"","number":"3400001"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"011441332323027","priority":4,"app_name":"Echo","app_data":""},"creationtime":"2020-04-10T01:08:43.435+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`

	_, evt, err := Parse([]byte(m))
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	e := evt.(*ChannelHangupRequest)
	if reflect.TypeOf(e.Channel) != reflect.TypeOf(Channel{}) {
		t.Errorf("Wrong")
	}

	if e.Cause != 18 {
		t.Errorf("Wrong match. expect:18, got: %d", e.Cause)
	}
	if e.Soft != false {
		t.Errorf("Wrong match. expect: false, got: %t", e.Soft)
	}
}

func TestParseChannelStateChange(t *testing.T) {
	type test struct {
		name    string
		message string
	}

	tests := []test{
		{
			"normal test",
			`{"type":"ChannelStateChange","timestamp":"2020-04-25T14:14:18.872+0000","channel":{"id":"1587824058.4108","name":"PJSIP/in-voipbin-00000fc8","state":"Up","caller":{"name":"","number":"1001"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"+46842002310","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=1839956265-1792916014-1999965658,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=45.151.255.178"},"creationtime":"2020-04-25T14:14:18.670+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if event.Type != EventTypeChannelStateChange {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelStateChange, event.Type)
			}

			e := evt.(*ChannelStateChange)
			if e.Type != EventTypeChannelStateChange {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelStateChange, e.Type)
			}
		})
	}
}

func TestParseChannelEnteredBridge(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *ChannelEnteredBridge
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelEnteredBridge","timestamp":"2020-05-03T23:46:27.547+0000","bridge":{"id":"06df8336-8ec7-11ea-80ac-ffd373732383","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":["1588549537.157"],"creationtime":"2020-05-03T23:37:49.233+0000","video_mode":"talker"},"channel":{"id":"1588549537.157","name":"PJSIP/in-voipbin-00000050","state":"Ring","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"3312","priority":2,"app_name":"Stasis","app_data":"test"},"creationtime":"2020-05-03T23:45:37.083+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&ChannelEnteredBridge{
				Event{
					Type:        EventTypeChannelEnteredBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-03T23:46:27.547",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Channel{
					ID:           "1588549537.157",
					Name:         "PJSIP/in-voipbin-00000050",
					Language:     "en",
					CreationTime: "2020-05-03T23:45:37.083",
					State:        "Ring",
					Caller: CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "3312",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "test",
					},
				},
				Bridge{
					ID:          "06df8336-8ec7-11ea-80ac-ffd373732383",
					Name:        "test",
					BridgeType:  "mixing",
					Technology:  "simple_bridge",
					BridgeClass: "stasis",
					Creator:     "Stasis",
					VideoMode:   "talker",

					Channels:     []string{"1588549537.157"},
					CreationTime: "2020-05-03T23:37:49.233+0000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if event.Type != EventTypeChannelEnteredBridge {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelEnteredBridge, event.Type)
			}

			e := evt.(*ChannelEnteredBridge)
			if e.Type != EventTypeChannelEnteredBridge {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelEnteredBridge, e.Type)
			}

			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}
		})
	}
}

func TestParseChannelLeftBridge(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *ChannelLeftBridge
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelLeftBridge","timestamp":"2020-05-03T23:43:45.791+0000","bridge":{"id":"b537fb10-8ec8-11ea-800e-ebc541120c4e","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":[],"creationtime":"2020-05-03T23:37:49.233+0000","video_mode":"talker"},"channel":{"id":"1588549395.152","name":"Snoop/1588549377.150-00000002","state":"Up","caller":{"name":"","number":""},"connected":{"name":"tttt","number":"pchero"},"accountcode":"","dialplan":{"context":"default","exten":"s","priority":1,"app_name":"Stasis","app_data":"test"},"creationtime":"2020-05-03T23:43:15.451+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&ChannelLeftBridge{
				Event{
					Type:        EventTypeChannelLeftBridge,
					Application: "voipbin",
					Timestamp:   "2020-05-03T23:43:45.791",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Channel{
					ID:           "1588549395.152",
					Name:         "Snoop/1588549377.150-00000002",
					Language:     "en",
					CreationTime: "2020-05-03T23:43:15.451",
					State:        "Up",
					Connected: CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: DialplanCEP{
						Context:  "default",
						Exten:    "s",
						Priority: 1,
						AppName:  "Stasis",
						AppData:  "test",
					},
				},
				Bridge{
					ID:          "b537fb10-8ec8-11ea-800e-ebc541120c4e",
					Name:        "test",
					BridgeType:  "mixing",
					Technology:  "simple_bridge",
					BridgeClass: "stasis",
					Creator:     "Stasis",
					VideoMode:   "talker",

					Channels:     []string{},
					CreationTime: "2020-05-03T23:37:49.233+0000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if event.Type != EventTypeChannelLeftBridge {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelLeftBridge, event.Type)
			}

			e := evt.(*ChannelLeftBridge)
			if e.Type != EventTypeChannelLeftBridge {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelLeftBridge, e.Type)
			}

			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}
		})
	}
}

func TestParseChannelDtmfReceived(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *ChannelDtmfReceived
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelDtmfReceived","timestamp":"2020-05-20T06:40:45.663+0000","digit":"9","duration_ms":100,"channel":{"id":"1589956827.6285","name":"PJSIP/call-in-00000633","state":"Up","caller":{"name":"tttt","number":"pchero"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"9912321321","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=n1kN7Utaj-,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161"},"creationtime":"2020-05-20T06:40:27.599+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			&ChannelDtmfReceived{
				Event{
					Type:        EventTypeChannelDtmfReceived,
					Application: "voipbin",
					Timestamp:   "2020-05-20T06:40:45.663",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				"9",
				100,
				Channel{
					ID:           "1589956827.6285",
					Name:         "PJSIP/call-in-00000633",
					Language:     "en",
					CreationTime: "2020-05-20T06:40:27.599",
					State:        "Up",
					Caller: CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: DialplanCEP{
						Context:  "call-in",
						Exten:    "9912321321",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=n1kN7Utaj-,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=213.127.79.161",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if event.Type != EventTypeChannelDtmfReceived {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelDtmfReceived, event.Type)
			}

			e := evt.(*ChannelDtmfReceived)
			if e.Type != EventTypeChannelDtmfReceived {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelDtmfReceived, e.Type)
			}

			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectEvent, e)
			}
		})
	}

}

func TestParseChannelVarset(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *ChannelVarset
	}

	tests := []test{
		{
			"normal",
			`{"variable":"STASISSTATUS","value":"","type":"ChannelVarset","timestamp":"2020-08-16T00:52:39.218+0000","channel":{"id":"instance-asterisk-production-europe-west4-a-1-1597539159.80042","name":"PJSIP/call-in-00004fb4","state":"Ring","caller":{"name":"","number":"7trunk"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"34967970028","priority":3,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77"},"creationtime":"2020-08-16T00:52:39.214+0000","language":"en"},"asterisk_id":"42:01:0a:a4:0f:ce","application":"test"}`,
			&ChannelVarset{
				Event{
					Type:        EventTypeChannelVarset,
					Application: "test",
					Timestamp:   "2020-08-16T00:52:39.218",
					AsteriskID:  "42:01:0a:a4:0f:ce",
				},
				"STASISSTATUS",
				"",
				Channel{
					ID:           "instance-asterisk-production-europe-west4-a-1-1597539159.80042",
					Name:         "PJSIP/call-in-00004fb4",
					Language:     "en",
					CreationTime: "2020-08-16T00:52:39.214",
					State:        "Ring",
					Caller: CallerID{
						Name:   "",
						Number: "7trunk",
					},
					Dialplan: DialplanCEP{
						Context:  "call-in",
						Exten:    "34967970028",
						Priority: 3,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if event.Type != EventTypeChannelVarset {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelVarset, event.Type)
			}

			e := evt.(*ChannelVarset)
			if e.Type != EventTypeChannelVarset {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeChannelVarset, e.Type)
			}

			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectEvent, e)
			}
		})
	}

}
