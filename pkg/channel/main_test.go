package channel

import (
	"reflect"
	"testing"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

func TestGetTech(t *testing.T) {
	type test struct {
		name       string
		testName   string
		expectTech Tech
	}

	tests := []test{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getTech(tt.testName)
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

				State: "Ring",
				Data:  make(map[string]interface{}, 1),

				TMCreate: "2020-04-25T00:08:32.346",
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
