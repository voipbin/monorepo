package telnyx

import (
	"encoding/json"
	"reflect"
	"testing"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
)

func Test_messageUnmarshal(t *testing.T) {
	tests := []struct {
		name string

		data      []byte
		expectRes Message
	}{
		{
			"normal",

			[]byte(`{
				"data": {
					"event_type": "message.received",
					"id": "19539336-11ba-4792-abd8-26d4f8745c4c",
					"occurred_at": "2022-03-15T16:16:24.073+00:00",
					"payload": {
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
					},
					"record_type": "event"
				},
				"meta": {
					"attempt": 1,
					"delivered_to": "https://en7evajwhmqbt.x.pipedream.net"
				}
			}`),
			Message{
				Data: Data{
					EventType:  "message.received",
					ID:         "19539336-11ba-4792-abd8-26d4f8745c4c",
					OccurredAt: "2022-03-15T16:16:24.073+00:00",
					Payload: Payload{
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
					RecordType: "event",
				},
				Meta: Meta{
					Attempt:     1,
					DeliveredTo: "https://en7evajwhmqbt.x.pipedream.net",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := Message{}
			if err := json.Unmarshal(tt.data, &res); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

// func Test_ConvertMessage(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		message    Message
// 		id         uuid.UUID
// 		customerID uuid.UUID

// 		expectRes *message.Message
// 	}{
// 		{
// 			"normal",

// 			Message{
// 				Data: Data{
// 					EventType:  "message.received",
// 					ID:         "19539336-11ba-4792-abd8-26d4f8745c4c",
// 					OccurredAt: "2022-03-15T16:16:24.073+00:00",
// 					Payload: Payload{
// 						Direction: "inbound",
// 						Encoding:  "GSM-7",
// 						From: FromTo{
// 							PhoneNumber: "+75973",
// 						},
// 						ID:                 "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
// 						MessagingProfileID: "40017f8e-49bd-4f16-9e3d-ef103f916228",
// 						OrganizationID:     "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
// 						Parts:              1,
// 						ReceivedAt:         "2022-03-15T16:16:23.466+00:00",
// 						RecordType:         "message",
// 						Subject:            "",
// 						Text:               "pchero21:\nTest message from skype.",
// 						To: []FromTo{
// 							{
// 								Carrier:     "Telnyx",
// 								LineType:    "Wireless",
// 								PhoneNumber: "+15734531118",
// 								Status:      "webhook_delivered",
// 							},
// 						},
// 						Type:       "SMS",
// 						WebhookURL: "https://en7evajwhmqbt.x.pipedream.net",
// 					},
// 					RecordType: "event",
// 				},
// 				Meta: Meta{
// 					Attempt:     1,
// 					DeliveredTo: "https://en7evajwhmqbt.x.pipedream.net",
// 				},
// 			},
// 			uuid.FromStringOrNil("56feb856-a4db-11ec-8e80-3be737041d0d"),
// 			uuid.FromStringOrNil("a463ecb8-a4d3-11ec-9c5a-5febe3922ab7"),

// 			&message.Message{
// 				ID:         uuid.FromStringOrNil("56feb856-a4db-11ec-8e80-3be737041d0d"),
// 				CustomerID: uuid.FromStringOrNil("a463ecb8-a4d3-11ec-9c5a-5febe3922ab7"),
// 				Type:       message.TypeSMS,
// 				Source: &commonaddress.Address{
// 					Type:   commonaddress.TypeTel,
// 					Target: "+75973",
// 				},
// 				Targets: []target.Target{
// 					{
// 						Destination: commonaddress.Address{
// 							Type:   commonaddress.TypeTel,
// 							Target: "+15734531118",
// 						},
// 						Status:   target.StatusReceived,
// 						Parts:    1,
// 						TMUpdate: "",
// 					},
// 				},
// 				ProviderName:        message.ProviderNameTelnyx,
// 				ProviderReferenceID: "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
// 				Text:                "pchero21:\nTest message from skype.",
// 				Medias:              []string{},
// 				Direction:           message.DirectionInbound,
// 				TMCreate:            "",
// 				TMUpdate:            "",
// 				TMDelete:            "",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {

// 			res := tt.message.ConvertMessage(tt.id, tt.customerID)

// 			tt.expectRes.TMCreate = res.TMCreate
// 			tt.expectRes.TMUpdate = res.TMUpdate
// 			tt.expectRes.TMDelete = res.TMDelete
// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

func Test_GetSource(t *testing.T) {
	tests := []struct {
		name string

		message Message

		expectRes *commonaddress.Address
	}{
		{
			"normal",

			Message{
				Data: Data{
					EventType:  "message.received",
					ID:         "19539336-11ba-4792-abd8-26d4f8745c4c",
					OccurredAt: "2022-03-15T16:16:24.073+00:00",
					Payload: Payload{
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
					RecordType: "event",
				},
				Meta: Meta{
					Attempt:     1,
					DeliveredTo: "https://en7evajwhmqbt.x.pipedream.net",
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+75973",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.message.GetSource()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GetTargets(t *testing.T) {
	tests := []struct {
		name string

		message Message

		expectRes []target.Target
	}{
		{
			"normal",

			Message{
				Data: Data{
					EventType:  "message.received",
					ID:         "19539336-11ba-4792-abd8-26d4f8745c4c",
					OccurredAt: "2022-03-15T16:16:24.073+00:00",
					Payload: Payload{
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
					RecordType: "event",
				},
				Meta: Meta{
					Attempt:     1,
					DeliveredTo: "https://en7evajwhmqbt.x.pipedream.net",
				},
			},

			[]target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+15734531118",
					},
					Status:   target.StatusReceived,
					Parts:    1,
					TMUpdate: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.message.GetTargets()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GetText(t *testing.T) {
	tests := []struct {
		name string

		message   Message
		expectRes string
	}{
		{
			name: "normal",

			message: Message{
				Data: Data{
					EventType:  "message.received",
					ID:         "19539336-11ba-4792-abd8-26d4f8745c4c",
					OccurredAt: "2022-03-15T16:16:24.073+00:00",
					Payload: Payload{
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
					RecordType: "event",
				},
				Meta: Meta{
					Attempt:     1,
					DeliveredTo: "https://en7evajwhmqbt.x.pipedream.net",
				},
			},
			expectRes: "pchero21:\nTest message from skype.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.message.GetText()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
