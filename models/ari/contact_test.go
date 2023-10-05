package ari

import (
	"reflect"
	"testing"
)

func TestParseContactStatusChange(t *testing.T) {
	type test struct {
		name        string
		message     string
		expectEvent *ContactStatusChange
	}

	tests := []test{
		{
			"test normal",
			`{ "application": "voipbin", "contact_info": { "uri": "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws", "roundtrip_usec": "0", "aor": "test11@test.trunk.voipbin.net", "contact_status": "NonQualified" }, "type": "ContactStatusChange", "endpoint": { "channel_ids": [], "resource": "test11@test.trunk.voipbin.net", "state": "online", "technology": "PJSIP" }, "timestamp": "2021-02-19T06:32:14.621+0000", "asterisk_id": "8e:86:e2:2c:a7:51"}`,
			&ContactStatusChange{
				Event: Event{
					Type:        EventTypeContactStatusChange,
					Application: "voipbin",
					Timestamp:   "2021-02-19T06:32:14.621",
					AsteriskID:  "8e:86:e2:2c:a7:51",
				},
				ContactInfo: ContactInfo{
					URI:           "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws",
					RoundtripUsec: "0",
					AOR:           "test11@test.trunk.voipbin.net",
					ContactStatus: ContactStatusTypeNonQualified,
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

			e := evt.(*ContactStatusChange)
			if reflect.DeepEqual(tt.expectEvent, e) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectEvent, e)
			}
		})
	}

}
