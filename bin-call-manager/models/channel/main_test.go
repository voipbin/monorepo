package channel

import (
	"monorepo/bin-call-manager/pkg/testhelper"
	"reflect"
	"testing"

	"monorepo/bin-call-manager/models/ari"
)

func TestGetTech(t *testing.T) {
	type test struct {
		name       string
		testName   string
		expectTech Tech
	}

	tests := []test{
		{
			"audiosocket",
			"AudioSocket/10.96.1.162:10000-5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547",
			TechAudioSocket,
		},
		{
			"pjsip",
			"PJSIP/in-voipbin-000002f6",
			TechPJSIP,
		},
		{
			"sip",
			"SIP/in-voipbin-00000006",
			TechSIP,
		},
		{
			"snoop",
			"Snoop/1588549018.132-00000000",
			TechSnoop,
		},
		{
			"local",
			"Local/5q-1629605410.6626",
			TechLocal,
		},
		{
			"unicastrtp",
			"UnicastRTP/127.0.0.1:5090-0x7f6d54035300",
			TechUnicatRTP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := GetTech(tt.testName)
			if res != tt.expectTech {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectTech, res)
			}
		})
	}
}

func TestNewChannelByChannelCreated(t *testing.T) {
	type test struct {
		name          string
		message       string
		expectChannel *Channel
	}

	tests := []test{
		{
			"normal",
			`{"type":"ChannelCreated","timestamp":"2020-04-25T00:08:32.346+0000","channel":{"id":"1587773312.8117","name":"PJSIP/in-voipbin-00001fa9","state":"Ring","caller":{"name":"","number":"1017"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0208442037691478","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-25T00:08:32.346+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			&Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "1587773312.8117",
				Name:       "PJSIP/in-voipbin-00001fa9",
				Tech:       "pjsip",

				SourceName:        "",
				SourceNumber:      "1017",
				DestinationNumber: "0208442037691478",

				State:      "Ring",
				Data:       map[string]interface{}{},
				StasisData: map[StasisDataType]string{},

				TMCreate: testhelper.TimePtr("2020-04-25T00:08:32.346"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := ari.Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			e := evt.(*ari.ChannelCreated)

			channel := NewChannelByChannelCreated(e)
			if !reflect.DeepEqual(tt.expectChannel, channel) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, channel)
			}
		})
	}
}

func Test_NewChannelByStasisStart(t *testing.T) {
	type test struct {
		name          string
		message       string
		expectChannel *Channel
	}

	tests := []test{
		{
			"normal",
			`{"type":"StasisStart","timestamp":"2020-05-10T07:11:05.479+0000","args":["CONTEXT=in-voipbin","SIP_CALLID=1578514523-1170819966-743482919","SIP_PAI=","SIP_PRIVACY=","DOMAIN=sip-service.voipbin.net","SOURCE=45.249.91.194"],"channel":{"id":"1589094665.1053","name":"PJSIP/in-voipbin-0000015e","state":"Ring","caller":{"name":"","number":"2000000"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"9103011442037694942","priority":2,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,SIP_CALLID=1578514523-1170819966-743482919,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=45.249.91.194"},"creationtime":"2020-05-10T07:11:05.477+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,
			&Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "1589094665.1053",
				Name:       "PJSIP/in-voipbin-0000015e",
				Tech:       "pjsip",

				SourceName:        "",
				SourceNumber:      "2000000",
				DestinationNumber: "9103011442037694942",

				State:      "Ring",
				Data:       make(map[string]interface{}, 1),
				StasisName: "voipbin",
				StasisData: map[StasisDataType]string{
					"CONTEXT":     "in-voipbin",
					"SIP_CALLID":  "1578514523-1170819966-743482919",
					"SIP_PAI":     "",
					"SIP_PRIVACY": "",
					"DOMAIN":      "sip-service.voipbin.net",
					"SOURCE":      "45.249.91.194",
				},

				TMCreate: testhelper.TimePtr("2020-05-10T07:11:05.479"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := ari.Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			e := evt.(*ari.StasisStart)

			channel := NewChannelByStasisStart(e)
			if !reflect.DeepEqual(tt.expectChannel, channel) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectChannel, channel)
			}
		})
	}
}

func TestNewChannelByARIChannel(t *testing.T) {
	type test struct {
		name          string
		ariChannel    *ari.Channel
		expectChannel *Channel
	}

	tests := []test{
		{
			"normal",
			&ari.Channel{
				ID:           "asterisk-call-5765d977d8-0a16-1629605410.6626",
				Name:         "UnicastRTP/127.0.0.1:5090-0x7f6d54035300",
				Language:     "en",
				CreationTime: "2021-08-22T04:10:10.331",
				State:        ari.ChannelStateDown,
				Caller:       ari.CallerID{},
				Connected:    ari.CallerID{},
				Dialplan: ari.DialplanCEP{
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
			&Channel{
				ID:   "asterisk-call-5765d977d8-0a16-1629605410.6626",
				Name: "UnicastRTP/127.0.0.1:5090-0x7f6d54035300",
				Tech: TechUnicatRTP,

				DestinationNumber: "s",

				State: ari.ChannelStateDown,
				Data: map[string]interface{}{
					"UNICASTRTP_LOCAL_ADDRESS": "127.0.0.1",
					"UNICASTRTP_LOCAL_PORT":    "10492",
				},
				StasisData: map[StasisDataType]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewChannelByARIChannel(tt.ariChannel)
			if !reflect.DeepEqual(tt.expectChannel, channel) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectChannel, channel)
			}
		})
	}
}
