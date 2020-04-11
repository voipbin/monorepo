package ari

import (
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
	m := `
	{
		"type": "ChannelCreated",
		"timestamp": "2020-04-10T01:09:10.574+0000",
		"channel": {
			"id": "1586480950.6217",
			"name": "PJSIP/in-voipbin-00000c26",
			"state": "Ring",
			"caller": {
				"name": "",
				"number": "3400001"
			},
			"connected": {
				"name": "",
				"number": ""
			},
			"accountcode": "",
			"dialplan": {
				"context": "in-voipbin",
				"exten": "9011441332323027",
				"priority": 1,
				"app_name": "",
				"app_data": ""
			},
			"creationtime": "2020-04-10T01:09:10.574+0000",
			"language": "en"
		},
		"asterisk_id": "42:01:0a:a4:00:05",
		"application": "voipbin"
	}
	`

	_, evt, err := Parse([]byte(m))
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	e := evt.(*ChannelCreated)
	if reflect.TypeOf(e.Channel) != reflect.TypeOf(Channel{}) {
		t.Errorf("Wrong")
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
