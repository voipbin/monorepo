package ari

import (
	"reflect"
	"testing"
)

func TestParseBridgeCreated(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *BridgeCreated
	}

	tests := []test{
		{
			"have no channel",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":[],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&BridgeCreated{
				Event{
					Type:        EventTypeBridgeCreated,
					Application: "voipbin",
					Timestamp:   "2020-05-03T21:35:02.809",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Bridge{
					ID:           "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
					Name:         "test",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					Channels:     []string{},
					VideoMode:    "none",
					CreationTime: "2020-05-03T21:35:02.692+0000",
				},
			},
		},
		{
			"has 1 channels",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":["7bffe100-8ec3-11ea-a8af-cbc12adc0497"],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&BridgeCreated{
				Event{
					Type:        EventTypeBridgeCreated,
					Application: "voipbin",
					Timestamp:   "2020-05-03T21:35:02.809",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Bridge{
					ID:           "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
					Name:         "test",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					Channels:     []string{"7bffe100-8ec3-11ea-a8af-cbc12adc0497"},
					VideoMode:    "none",
					CreationTime: "2020-05-03T21:35:02.692+0000",
				},
			},
		}, {
			"has 2 channels",
			`{"type":"BridgeCreated","timestamp":"2020-05-03T21:35:02.809+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":["5a6ff11a-8ec3-11ea-8d25-ab34368e7c93", "65487e22-8ec3-11ea-9874-07969d28d2f7"],"creationtime":"2020-05-03T21:35:02.692+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&BridgeCreated{
				Event{
					Type:        EventTypeBridgeCreated,
					Application: "voipbin",
					Timestamp:   "2020-05-03T21:35:02.809",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Bridge{
					ID:           "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
					Name:         "test",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					Channels:     []string{"5a6ff11a-8ec3-11ea-8d25-ab34368e7c93", "65487e22-8ec3-11ea-9874-07969d28d2f7"},
					VideoMode:    "none",
					CreationTime: "2020-05-03T21:35:02.692+0000",
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

			if event.Type != EventTypeBridgeCreated {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeBridgeCreated, event.Type)
			}

			e := evt.(*BridgeCreated)
			if e.Type != EventTypeBridgeCreated {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeBridgeCreated, e.Type)
			}

			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}

		})
	}
}

func TestParseBridgeDestroyed(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *BridgeDestroyed
	}

	tests := []test{
		{
			"normal",
			` {"type":"BridgeDestroyed","timestamp":"2020-05-04T00:27:59.747+0000","bridge":{"id":"0e9f0998-8ec2-11ea-970a-df70fd3c4853","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":[],"creationtime":"2020-05-03T23:37:49.233+0000","video_mode":"talker"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			&BridgeDestroyed{
				Event{
					Type:        EventTypeBridgeDestroyed,
					Application: "voipbin",
					Timestamp:   "2020-05-04T00:27:59.747",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Bridge{
					ID:           "0e9f0998-8ec2-11ea-970a-df70fd3c4853",
					Name:         "test",
					BridgeType:   "mixing",
					Technology:   "simple_bridge",
					BridgeClass:  "stasis",
					Creator:      "Stasis",
					Channels:     []string{},
					VideoMode:    "talker",
					CreationTime: "2020-05-03T23:37:49.233+0000",
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

			if event.Type != EventTypeBridgeDestroyed {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeBridgeDestroyed, event.Type)
			}

			e := evt.(*BridgeDestroyed)
			if e.Type != EventTypeBridgeDestroyed {
				t.Errorf("Wrong match. expect: %s, got: %s", EventTypeBridgeDestroyed, e.Type)
			}

			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectEvent, e)
			}
		})
	}
}

func TestParseBridge(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectParse *Bridge
	}

	tests := []test{
		{
			"test normal",
			`{"id":"3e6eec96-fabe-4041-870d-e1daee11aafb","technology":"softmix","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"conference_type=conference,join=false","channels":[],"creationtime":"2020-08-10T22:50:28.085+0000","video_mode":"sfu"}`,
			&Bridge{
				ID:            "3e6eec96-fabe-4041-870d-e1daee11aafb",
				Name:          "conference_type=conference,join=false",
				Technology:    "softmix",
				BridgeType:    "mixing",
				BridgeClass:   "stasis",
				Creator:       "Stasis",
				VideoMode:     "sfu",
				VideoSourceID: "",
				Channels:      []string{},
				CreationTime:  "2020-08-10T22:50:28.085+0000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bridge, err := ParseBridge([]byte(tt.message))
			if err != nil {
				t.Errorf("Wront match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectParse, bridge) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectParse, bridge)
			}
		})
	}
}
