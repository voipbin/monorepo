package bridge

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
)

func TestNewBridgeByBridgeCreated(t *testing.T) {
	type test struct {
		name         string
		message      string
		expectBridge *Bridge
	}

	tests := []test{
		{
			"normal reference call type",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"reference_type=call,reference_id=5cef1dbc-13d8-11ec-9199-f3e0e965a469","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID:    "42:01:0a:a4:00:03",
				ID:            "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:          "reference_type=call,reference_id=5cef1dbc-13d8-11ec-9199-f3e0e965a469",
				Type:          TypeMixing,
				Tech:          TechSimple,
				Class:         "stasis",
				Creator:       "Stasis",
				VideoMode:     "none",
				ChannelIDs:    []string{},
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("5cef1dbc-13d8-11ec-9199-f3e0e965a469"),
				TMCreate:      "2020-05-03T21:35:02.809",
			},
		},
		{
			"normal reference conference type",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"reference_type=confbridge,reference_id=8f537474-13d8-11ec-9193-7b377238c934","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID:    "42:01:0a:a4:00:03",
				ID:            "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:          "reference_type=confbridge,reference_id=8f537474-13d8-11ec-9193-7b377238c934",
				Type:          TypeMixing,
				Tech:          TechSimple,
				Class:         "stasis",
				Creator:       "Stasis",
				VideoMode:     "none",
				ChannelIDs:    []string{},
				ReferenceType: ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("8f537474-13d8-11ec-9193-7b377238c934"),
				TMCreate:      "2020-05-03T21:35:02.809",
			},
		},
		{
			"emtpy name",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID:    "42:01:0a:a4:00:03",
				ID:            "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:          "",
				Type:          TypeMixing,
				Tech:          TechSimple,
				Class:         "stasis",
				Creator:       "Stasis",
				VideoMode:     "none",
				ChannelIDs:    []string{},
				ReferenceType: ReferenceTypeUnknown,
				ReferenceID:   uuid.Nil,
				TMCreate:      "2020-05-03T21:35:02.809",
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := ari.Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			e := evt.(*ari.BridgeCreated)

			bridge := NewBridgeByBridgeCreated(e)
			if !reflect.DeepEqual(tt.expectBridge, bridge) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBridge, bridge)
			}
		})
	}
}

func TestParseBridgeName(t *testing.T) {
	type test struct {
		name       string
		bridgeName string
		expectRes  map[string]string
	}

	tests := []test{
		{
			"reference type call",
			"reference_type=call,reference_id=7444bf66-13d9-11ec-9a1a-ab73ca01d2c6",
			map[string]string{
				"reference_type": "call",
				"reference_id":   "7444bf66-13d9-11ec-9a1a-ab73ca01d2c6",
			},
		},
		{
			"reference type conference",
			"reference_type=conference,reference_id=88cba8aa-13d9-11ec-96f2-23bb37899eb4",
			map[string]string{
				"reference_type": "conference",
				"reference_id":   "88cba8aa-13d9-11ec-96f2-23bb37899eb4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ParseBridgeName(tt.bridgeName)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expact: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
