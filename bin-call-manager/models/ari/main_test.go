package ari

import (
	"encoding/json"
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
		{"ChannelVarset", `{"variable":"STASISSTATUS","value":"","type":"ChannelVarset","timestamp":"2020-08-16T00:52:39.218+0000","channel":{"id":"instance-asterisk-production-europe-west4-a-1-1597539159.80042","name":"PJSIP/call-in-00004fb4","state":"Ring","caller":{"name":"","number":"7trunk"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"call-in","exten":"34967970028","priority":3,"app_name":"Stasis","app_data":"voipbin,CONTEXT=call-in,SIP_CALLID=7b9d3e3148cb48aca801f7a015e7aa7b@1634430,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=51.79.98.77"},"creationtime":"2020-08-16T00:52:39.214+0000","language":"en"},"asterisk_id":"42:01:0a:a4:0f:ce","application":"test"}`},
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

func Test_ArgsMapUnmarshalJSON(t *testing.T) {
	m := `["context=test_context", "domain=sip-service.voipbin.net"]`

	res := ArgsMap{}
	if err := json.Unmarshal([]byte(m), &res); err != nil {
		t.Errorf("Wrong match. expact: ok, got: %v", err)
	}

	if res["context"] != "test_context" {
		t.Errorf("Wrong match. expact: text_context, got: %s", res["context"])
	}
	if res["domain"] != "sip-service.voipbin.net" {
		t.Errorf("Wrong match. expact: sip-service.voipbin.net, got: %s", res["domain"])
	}
}

func TestArgsMapUnmarshalJSONError(t *testing.T) {
	type test struct {
		name    string
		message string
	}

	tests := []test{
		{"wrong list", `["context=test_context", "domain=sip-service.voipbin.net"`},
		{"wrong item", `["context=test_context", "domain=sip-service.voipbin.net]`},
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
