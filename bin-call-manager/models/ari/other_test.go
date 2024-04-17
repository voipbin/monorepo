package ari

import (
	"reflect"
	"testing"
)

func Test_ParseStasisStart(t *testing.T) {

	tests := []struct {
		name string

		message string

		expectRes *StasisStart
	}{

		{
			name:    "normal",
			message: `{"type":"StasisStart","timestamp":"2020-04-12T22:34:41.144+0000","args":["context=in-voipbin","domain=34.90.68.237"],"channel":{"id":"1586730880.1791","name":"PJSIP/in-voipbin-00000381","state":"Up","caller":{"name":"","number":"test123"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0046605844066","priority":4,"app_name":"Stasis","app_data":"voipbin,context=in-voipbin,domain=34.90.68.237"},"creationtime":"2020-04-12T22:34:40.641+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`,

			expectRes: &StasisStart{
				Event: Event{
					Type:        EventTypeStasisStart,
					Application: "voipbin",
					Timestamp:   "2020-04-12T22:34:41.144",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Args: map[string]string{
					"context": "in-voipbin",
					"domain":  "34.90.68.237",
				},
				Channel: Channel{
					AccountCode:  "",
					ID:           "1586730880.1791",
					Name:         "PJSIP/in-voipbin-00000381",
					Language:     "en",
					CreationTime: "2020-04-12T22:34:40.641",
					State:        "Up",
					Caller: CallerID{
						Number: "test123",
					},
					Connected: CallerID{},
					Dialplan: DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "0046605844066",
						Priority: 4,
						AppName:  "Stasis",
						AppData:  "voipbin,context=in-voipbin,domain=34.90.68.237",
					},
				},
			},
		},
		{
			name:    "wrong args format",
			message: `{"type":"StasisStart","timestamp":"2024-03-25T07:18:48.380+0000","args":["5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547"],"channel":{"id":"asterisk-call-f4df6d4d7-clwdl-1711351128.83","name":"AudioSocket/10.96.1.162:10000-5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547","state":"Up","protocol_id":"","caller":{"name":"","number":""},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"default","exten":"s","priority":1,"app_name":"Stasis","app_data":"voipbin,5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547"},"creationtime":"2024-03-25T07:18:48.378+0000","language":"en"},"asterisk_id":"be:c6:98:a4:21:17","application":"voipbin"}`,

			expectRes: &StasisStart{
				Event: Event{
					Type:        EventTypeStasisStart,
					Application: "voipbin",
					Timestamp:   "2024-03-25T07:18:48.380",
					AsteriskID:  "be:c6:98:a4:21:17",
				},
				Args: map[string]string{
					"5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547": "",
				},
				Channel: Channel{
					ID:           "asterisk-call-f4df6d4d7-clwdl-1711351128.83",
					Name:         "AudioSocket/10.96.1.162:10000-5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547",
					Language:     "en",
					CreationTime: "2024-03-25T07:18:48.378",
					State:        "Up",
					Dialplan: DialplanCEP{
						Context:  "default",
						Exten:    "s",
						Priority: 1,
						AppName:  "Stasis",
						AppData:  "voipbin,5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, evt, err := Parse([]byte(tt.message))
			t.Logf("Parse: %s", evt)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res := evt.(*StasisStart)
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}

	// m := `{"type":"StasisStart","timestamp":"2020-04-12T22:34:41.144+0000","args":["context=in-voipbin","domain=34.90.68.237"],"channel":{"id":"1586730880.1791","name":"PJSIP/in-voipbin-00000381","state":"Up","caller":{"name":"","number":"test123"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0046605844066","priority":4,"app_name":"Stasis","app_data":"voipbin,context=in-voipbin,domain=34.90.68.237"},"creationtime":"2020-04-12T22:34:40.641+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`

	// _, evt, err := Parse([]byte(m))
	// t.Logf("Parse: %s", evt)
	// if err != nil {
	// 	t.Errorf("Wrong match. expect: ok, got: %v", err)
	// }

	// e := evt.(*StasisStart)
	// if reflect.TypeOf(e.Channel) != reflect.TypeOf(Channel{}) {
	// 	t.Errorf("Wrong")
	// }

	// if e.Args["context"] != "in-voipbin" {
	// 	t.Errorf("Wrong match. expect: in-voipbin, got: %s", e.Args["CONTEXT"])
	// }
	// if e.Args["domain"] != "34.90.68.237" {
	// 	t.Errorf("Wrong match. expect: 34.90.68.237, got: %s", e.Args["DOMAIN"])
	// }
}

// {"type":"ChannelCreated","timestamp":"2024-03-25T07:18:48.378+0000","channel":{"id":"asterisk-call-f4df6d4d7-clwdl-1711351128.83","name":"AudioSocket/10.96.1.162:10000-5e0c90ce-c282-4c6f-ab62-e3dc1f9f2547","state":"Down","protocol_id":"","caller":{"name":"","number":""},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"default","exten":"s","priority":1,"app_name":"","app_data":""},"creationtime":"2024-03-25T07:18:48.378+0000","language":"en"},"asterisk_id":"be:c6:98:a4:21:17","application":"voipbin"}
