package ari

import (
	"reflect"
	"testing"
)

func TestParseRecordingFinished(t *testing.T) {

	type test struct {
		name        string
		message     string
		expectEvent *RecordingFinished
	}

	tests := []test{
		{
			"normal",
			`{"type": "RecordingFinished","timestamp": "2020-02-10T13:08:18.888+0000","recording": {"name": "testrecording-202002101401","format": "wav","state": "done","target_uri": "bridge:e9946f5c-2632-4a92-a608-068994d27cbf","duration": 351},"asterisk_id": "42:01:0a:84:00:68","application": "test"}`,
			&RecordingFinished{
				Event{
					Type:        EventTypeRecordingFinished,
					Application: "test",
					Timestamp:   "2020-02-10T13:08:18.888",
					AsteriskID:  "42:01:0a:84:00:68",
				},
				RecordingLive{
					Name:      "testrecording-202002101401",
					Format:    "wav",
					State:     "done",
					TargetURI: "bridge:e9946f5c-2632-4a92-a608-068994d27cbf",
					Duration:  351,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			e := evt.(*RecordingFinished)
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectEvent, e)
			}
		})
	}
}

func TestParseRecordingStarted(t *testing.T) {

	type test struct {
		name        string
		message     string
		expectEvent *RecordingStarted
	}

	tests := []test{
		{
			"normal",
			`{"type": "RecordingStarted","recording": {"name": "test_call","format": "wav","state": "recording","target_uri": "channel:test_call"},"asterisk_id": "42:01:0a:84:00:12","application": "voipbin"}`,
			&RecordingStarted{
				Event{
					Type:        EventTypeRecordingStarted,
					Application: "voipbin",
					AsteriskID:  "42:01:0a:84:00:12",
				},
				RecordingLive{
					Name:      "test_call",
					Format:    "wav",
					State:     "recording",
					TargetURI: "channel:test_call",
					Duration:  0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, evt, err := Parse([]byte(tt.message))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			e := evt.(*RecordingStarted)
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectEvent, e)
			}
		})
	}
}
