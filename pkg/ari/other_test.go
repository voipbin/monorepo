package ari

import (
	"reflect"
	"testing"
)

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
