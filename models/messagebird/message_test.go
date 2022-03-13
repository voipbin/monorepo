package messagebird

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
)

func Test_ConvertMessage(t *testing.T) {

	tests := []struct {
		name string

		message    *Message
		id         uuid.UUID
		customerID uuid.UUID

		expectRes *message.Message
	}{
		{
			"2 items",

			&Message{
				ID:         "6b79e50e426c4d64ac45345bae84fe55",
				Direction:  "mt",
				Type:       "sms",
				Originator: "+821021656521",
				Body:       "This is a test message10",

				Recipients: RecipientStruct{
					Items: []Recipient{
						{
							Recipient:        31616818985,
							Status:           "sent",
							StatusDatetime:   "2022-03-09T05:21:45+00:00",
							MessagePartCount: 1,
						},
						{
							Recipient:        821021656521,
							Status:           "sent",
							StatusDatetime:   "2022-03-09T05:21:45+00:00",
							MessagePartCount: 1,
						},
					},
				},
			},
			uuid.FromStringOrNil("82a1c5b4-a064-11ec-bad7-ebff9efa798e"),
			uuid.FromStringOrNil("82d16b7a-a064-11ec-9ce8-777e6052475b"),

			&message.Message{
				ID:         uuid.FromStringOrNil("82a1c5b4-a064-11ec-bad7-ebff9efa798e"),
				CustomerID: uuid.FromStringOrNil("82d16b7a-a064-11ec-9ce8-777e6052475b"),
				Type:       message.TypeSMS,
				Source: &cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				Targets: []target.Target{
					{
						Destination: cmaddress.Address{
							Type:   cmaddress.TypeTel,
							Target: "+31616818985",
						},
						Status: target.StatusSent,
						Parts:  1,
					},
					{
						Destination: cmaddress.Address{
							Type:   cmaddress.TypeTel,
							Target: "+821021656521",
						},
						Status: target.StatusSent,
						Parts:  1,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "This is a test message10",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
				TMCreate:            "",
				TMUpdate:            "",
				TMDelete:            "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.message.ConvertMessage(tt.id, tt.customerID)

			for i, target := range res.Targets {
				tt.expectRes.Targets[i].TMUpdate = target.TMUpdate
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_marshal(t *testing.T) {

	tests := []struct {
		name string

		data      []byte
		expectRes *Message
	}{
		{
			"normal",
			[]byte(`{
				"id": "6b79e50e426c4d64ac45345bae84fe55",
				"href": "https://rest.messagebird.com/messages/6b79e50e426c4d64ac45345bae84fe55",
				"direction": "mt",
				"type": "sms",
				"originator": "+821021656521",
				"body": "This is a test message10",
				"reference": null,
				"validity": null,
				"gateway": 10,
				"typeDetails": {},
				"datacoding": "plain",
				"mclass": 1,
				"scheduledDatetime": null,
				"createdDatetime": "2022-03-09T05:21:45+00:00",
				"recipients": {
				  "totalCount": 2,
				  "totalSentCount": 2,
				  "totalDeliveredCount": 0,
				  "totalDeliveryFailedCount": 0,
				  "items": [
					{
					  "recipient": 31616818985,
					  "status": "sent",
					  "statusDatetime": "2022-03-09T05:21:45+00:00",
					  "messagePartCount": 1
					},
					{
					  "recipient": 821021656521,
					  "status": "sent",
					  "statusDatetime": "2022-03-09T05:21:45+00:00",
					  "messagePartCount": 1
					}
				  ]
				}
			  }`),
			&Message{
				ID:              "6b79e50e426c4d64ac45345bae84fe55",
				Href:            "https://rest.messagebird.com/messages/6b79e50e426c4d64ac45345bae84fe55",
				Direction:       "mt",
				Type:            "sms",
				Originator:      "+821021656521",
				Body:            "This is a test message10",
				Gateway:         10,
				DataCoding:      "plain",
				MClass:          1,
				CreatedDatetime: "2022-03-09T05:21:45+00:00",
				Recipients: RecipientStruct{
					TotalCount:               2,
					TotalSentCount:           2,
					TotalDeliveredCount:      0,
					TotalDeliveryFailedCount: 0,
					Items: []Recipient{
						{
							Recipient:        31616818985,
							Status:           "sent",
							StatusDatetime:   "2022-03-09T05:21:45+00:00",
							MessagePartCount: 1,
						},
						{
							Recipient:        821021656521,
							Status:           "sent",
							StatusDatetime:   "2022-03-09T05:21:45+00:00",
							MessagePartCount: 1,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := Message{}
			if err := json.Unmarshal(tt.data, &res); err != nil {
				t.Errorf("Wrong match. expect: ok got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, &res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
