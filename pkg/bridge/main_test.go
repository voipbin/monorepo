package bridge

import (
	"reflect"
	"testing"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

func TestNewBridgeByBridgeCreated(t *testing.T) {
	type test struct {
		name         string
		message      string
		expectBridge *Bridge
	}

	tests := []test{
		{
			"normal",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID: "42:01:0a:a4:00:03",
				ID:         "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:       "test",
				Type:       TypeMixing,
				Tech:       TechSimple,
				Class:      "stasis",
				Creator:    "Stasis",
				VideoMode:  "none",
				Channels:   []string{},
				TMCreate:   "2020-05-03T21:35:02.809",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := ari.Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			e := evt.(*ari.BridgeCreated)

			bridge := NewBridgeByBridgeCreated(e)
			if !reflect.DeepEqual(tt.expectBridge, bridge) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBridge, bridge)
			}
		})
	}
}
