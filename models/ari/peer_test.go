package ari

import (
	"reflect"
	"testing"
)

func TestParsePeerStatusChange(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *PeerStatusChange
	}

	tests := []test{
		{
			"test normal",
			`{"endpoint": { "technology": "PJSIP", "channel_ids": [], "resource": "test11@test.trunk.voipbin.net", "state": "online" }, "timestamp": "2021-02-18T06:23:53.016+0000", "application": "voipbin", "peer": { "peer_status": "Reachable" }, "type": "PeerStatusChange", "asterisk_id": "42:6c:60:50:79:b3"}`,
			&PeerStatusChange{
				Event: Event{
					Type:        EventTypePeerStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-18T06:23:53.016",
					AsteriskID:  "42:6c:60:50:79:b3",
				},
				Peer: Peer{
					PeerStatus: PeerStatusReachable,
				},
				Endpoint: Endpoint{
					Technology: "PJSIP",
					ChannelIDs: []string{},
					Resource:   "test11@test.trunk.voipbin.net",
					State:      EndpointStateOnline,
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

			e := evt.(*PeerStatusChange)
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectEvent, e)
			}
		})
	}

}
