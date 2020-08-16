package bridge

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
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
				AsteriskID:     "42:01:0a:a4:00:03",
				ID:             "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:           "test",
				Type:           TypeMixing,
				Tech:           TechSimple,
				Class:          "stasis",
				Creator:        "Stasis",
				VideoMode:      "none",
				ChannelIDs:     []string{},
				ConferenceID:   uuid.Nil,
				ConferenceType: "",
				TMCreate:       "2020-05-03T21:35:02.809",
			},
		},
		{
			"have conference ID",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"conference_id=d15ab81a-9313-11ea-9e29-5b9ebfaeb39d","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID:     "42:01:0a:a4:00:03",
				ID:             "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:           "conference_id=d15ab81a-9313-11ea-9e29-5b9ebfaeb39d",
				Type:           TypeMixing,
				Tech:           TechSimple,
				Class:          "stasis",
				Creator:        "Stasis",
				VideoMode:      "none",
				ChannelIDs:     []string{},
				ConferenceID:   uuid.FromStringOrNil("d15ab81a-9313-11ea-9e29-5b9ebfaeb39d"),
				ConferenceType: "",
				TMCreate:       "2020-05-03T21:35:02.809",
			},
		},
		{
			"have conference id/type",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"conference_type=echo,conference_id=d15ab81a-9313-11ea-9e29-5b9ebfaeb39d","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID:     "42:01:0a:a4:00:03",
				ID:             "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:           "conference_type=echo,conference_id=d15ab81a-9313-11ea-9e29-5b9ebfaeb39d",
				Type:           TypeMixing,
				Tech:           TechSimple,
				Class:          "stasis",
				Creator:        "Stasis",
				VideoMode:      "none",
				ChannelIDs:     []string{},
				ConferenceID:   uuid.FromStringOrNil("d15ab81a-9313-11ea-9e29-5b9ebfaeb39d"),
				ConferenceType: conference.TypeEcho,
				TMCreate:       "2020-05-03T21:35:02.809",
			},
		},
		{
			"empty name",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&Bridge{
				AsteriskID:     "42:01:0a:a4:00:03",
				ID:             "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
				Name:           "",
				Type:           TypeMixing,
				Tech:           TechSimple,
				Class:          "stasis",
				Creator:        "Stasis",
				VideoMode:      "none",
				ChannelIDs:     []string{},
				ConferenceID:   uuid.Nil,
				ConferenceType: "",
				TMCreate:       "2020-05-03T21:35:02.809",
			},
		},
		{
			"joining type",
			`{"type":"BridgeCreated","timestamp":"2020-05-29T16:06:41.715+0000","bridge":{"id":"00a1dc4b-ad3c-4ceb-93c2-378452e4032c","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"conference_type=conference,conference_id=ef556f3f-ea0b-416f-bae1-91c38865aa3b,join=true","channels":[],"creationtime":"2020-05-29T16:06:41.715+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"test"}`,
			&Bridge{
				AsteriskID:     "42:01:0a:a4:00:03",
				ID:             "00a1dc4b-ad3c-4ceb-93c2-378452e4032c",
				Name:           "conference_type=conference,conference_id=ef556f3f-ea0b-416f-bae1-91c38865aa3b,join=true",
				Type:           TypeMixing,
				Tech:           TechSimple,
				Class:          "stasis",
				Creator:        "Stasis",
				VideoMode:      "none",
				ChannelIDs:     []string{},
				ConferenceID:   uuid.FromStringOrNil("ef556f3f-ea0b-416f-bae1-91c38865aa3b"),
				ConferenceType: conference.TypeConference,
				ConferenceJoin: true,
				TMCreate:       "2020-05-29T16:06:41.715",
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
			"normal",
			"type=echo,conference_id=eae05bf2-9311-11ea-bdbf-d393f883e80f",
			map[string]string{
				"type":          "echo",
				"conference_id": "eae05bf2-9311-11ea-bdbf-d393f883e80f",
			},
		},
		{
			"joining true",
			"type=conference,conference_id=9d21f58a-a237-11ea-b1f9-e74d37d1177c,joining=true",
			map[string]string{
				"type":          "conference",
				"conference_id": "9d21f58a-a237-11ea-b1f9-e74d37d1177c",
				"joining":       "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := parseBridgeName(tt.bridgeName)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expact: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
