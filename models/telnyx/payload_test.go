package telnyx

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_payloadUnmarshal(t *testing.T) {
	tests := []struct {
		name string

		data      []byte
		expectRes *Payload
	}{
		{
			"normal",

			[]byte(` {
					"cc": [],
					"completed_at": null,
					"cost": null,
					"direction": "inbound",
					"encoding": "GSM-7",
					"errors": [],
					"from": {
						"carrier": "",
						"line_type": "",
						"phone_number": "+75973"
					},
					"id": "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
					"media": [],
					"messaging_profile_id": "40017f8e-49bd-4f16-9e3d-ef103f916228",
					"organization_id": "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
					"parts": 1,
					"received_at": "2022-03-15T16:16:23.466+00:00",
					"record_type": "message",
					"sent_at": null,
					"subject": "",
					"tags": [],
					"text": "pchero21:\nTest message from skype.",
					"to": [
						{
						"carrier": "Telnyx",
						"line_type": "Wireless",
						"phone_number": "+15734531118",
						"status": "webhook_delivered"
						}
					],
					"type": "SMS",
					"valid_until": null,
					"webhook_failover_url": null,
					"webhook_url": "https://en7evajwhmqbt.x.pipedream.net"
				}`),
			&Payload{
				Direction: "inbound",
				Encoding:  "GSM-7",
				From: FromTo{
					PhoneNumber: "+75973",
				},
				ID:                 "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
				MessagingProfileID: "40017f8e-49bd-4f16-9e3d-ef103f916228",
				OrganizationID:     "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
				Parts:              1,
				ReceivedAt:         "2022-03-15T16:16:23.466+00:00",
				RecordType:         "message",
				Subject:            "",
				Text:               "pchero21:\nTest message from skype.",
				To: []FromTo{
					{
						Carrier:     "Telnyx",
						LineType:    "Wireless",
						PhoneNumber: "+15734531118",
						Status:      "webhook_delivered",
					},
				},
				Type:       "SMS",
				WebhookURL: "https://en7evajwhmqbt.x.pipedream.net",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := Payload{}
			if err := json.Unmarshal(tt.data, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, res)
			}
		})
	}
}
