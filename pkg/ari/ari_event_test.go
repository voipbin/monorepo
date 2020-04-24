package ari

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseNormal(t *testing.T) {
	type test struct {
		name    string
		message string
	}

	tests := []test{
		{"ChannelCreate", `{"type":"ChannelCreated","timestamp":"2020-04-10T01:09:10.574+0000","channel":{"id":"1586480950.6217","name":"PJSIP/in-voipbin-00000c26","state":"Ring","caller":{"name":"","number":"3400001"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"9011441332323027","priority":1,"app_name":"","app_data":""},"creationtime":"2020-04-10T01:09:10.574+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`},
		{"ChannelDestroyed", `{"type":"ChannelDestroyed","timestamp":"2020-04-10T01:09:12.924+0000","cause":18,"cause_txt":"No user responding","channel":{"id":"1586480888.6166","name":"PJSIP/in-voipbin-00000c0c","state":"Up","caller":{"name":"","number":"3400000"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"7011441332323027","priority":4,"app_name":"","app_data":""},"creationtime":"2020-04-10T01:08:08.922+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`},
		{"NotInListed", `{"type":"NotInListed","timestamp":"2020-04-10T01:09:12.924+0000","asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Type: %s", event.Type)
			t.Logf("Parsed. event: %v, %s", evt, evt)
		})
	}
}

func TestParseError(t *testing.T) {
	type test struct {
		name    string
		message string
	}

	tests := []test{
		{"wrong json format", `{`},
		{"wrong type in the message", `{"type":"ChannelDestroyed","timestamp":"2020-04-10T01:09:12.924+0000","cause":"wrong type","asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Parse([]byte(tt.message))
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
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

func TestArgsMapUnmarshalJSON(t *testing.T) {
	m := `["CONTEXT=test_context", "DOMAIN=echo.voipbin.net"]`

	res := ArgsMap{}
	if err := json.Unmarshal([]byte(m), &res); err != nil {
		t.Errorf("Wrong match. expact: ok, got: %v", err)
	}

	if res["CONTEXT"] != "test_context" {
		t.Errorf("Wrong match. expact: text_context, got: %s", res["CONTEXT"])
	}
	if res["DOMAIN"] != "echo.voipbin.net" {
		t.Errorf("Wrong match. expact: echo.voipbin.net, got: %s", res["DOMAIN"])
	}
}

func TestArgsMapUnmarshalJSONError(t *testing.T) {
	type test struct {
		name    string
		message string
	}

	tests := []test{
		{"wrong list", `["CONTEXT=test_context", "DOMAIN=echo.voipbin.net"`},
		{"wrong item", `["CONTEXT=test_context", "DOMAIN=echo.voipbin.net]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ArgsMap{}
			if err := json.Unmarshal([]byte(tt.message), &res); err == nil {
				t.Errorf("Wrong match. expact: err, got: ok")
			}
		})
	}
}

func TestParseStasisStart(t *testing.T) {
	m := `{"type":"StasisStart","timestamp":"2020-04-12T22:34:41.144+0000","args":["CONTEXT=in-voipbin","DOMAIN=34.90.68.237"],"channel":{"id":"1586730880.1791","name":"PJSIP/in-voipbin-00000381","state":"Up","caller":{"name":"","number":"test123"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0046605844066","priority":4,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,DOMAIN=34.90.68.237"},"creationtime":"2020-04-12T22:34:40.641+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`

	_, evt, err := Parse([]byte(m))
	t.Logf("Parse: %s", evt)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	e := evt.(*StasisStart)
	if reflect.TypeOf(e.Channel) != reflect.TypeOf(Channel{}) {
		t.Errorf("Wrong")
	}

	if e.Args["CONTEXT"] != "in-voipbin" {
		t.Errorf("Wrong match. expect: in-voipbin, got: %s", e.Args["CONTEXT"])
	}
	if e.Args["DOMAIN"] != "34.90.68.237" {
		t.Errorf("Wrong match. expect: 34.90.68.237, got: %s", e.Args["DOMAIN"])
	}
}
