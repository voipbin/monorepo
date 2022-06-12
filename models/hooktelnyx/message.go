package hooktelnyx

import (
	"strings"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	mmtarget "gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
)

// Message defines
// {
// 	"data": {
// 		"event_type": "message.received",
// 		"id": "19539336-11ba-4792-abd8-26d4f8745c4c",
// 		"occurred_at": "2022-03-15T16:16:24.073+00:00",
// 		"payload": {
// 			"cc": [],
// 			"completed_at": null,
// 			"cost": null,
// 			"direction": "inbound",
// 			"encoding": "GSM-7",
// 			"errors": [],
// 			"from": {
// 				"carrier": "",
// 				"line_type": "",
// 				"phone_number": "+75973"
// 			},
// 			"id": "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
// 			"media": [],
// 			"messaging_profile_id": "40017f8e-49bd-4f16-9e3d-ef103f916228",
// 			"organization_id": "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
// 			"parts": 1,
// 			"received_at": "2022-03-15T16:16:23.466+00:00",
// 			"record_type": "message",
// 			"sent_at": null,
// 			"subject": "",
// 			"tags": [],
// 			"text": "pchero21:\nTest message from skype.",
// 			"to": [
// 				{
// 				"carrier": "Telnyx",
// 				"line_type": "Wireless",
// 				"phone_number": "+15734531118",
// 				"status": "webhook_delivered"
// 				}
// 			],
// 			"type": "SMS",
// 			"valid_until": null,
// 			"webhook_failover_url": null,
// 			"webhook_url": "https://en7evajwhmqbt.x.pipedream.net"
// 		},
// 		"record_type": "event"
// 	},
// 	"meta": {
// 		"attempt": 1,
// 		"delivered_to": "https://en7evajwhmqbt.x.pipedream.net"
// 	}
// }
type Message struct {
	Data Data `json:"data"`
	Meta Meta `json:"meta"`
}

// Meta defines
type Meta struct {
	Attempt     int    `json:"attempt"`
	DeliveredTo string `json:"delivered_to"`
}

// Data defines
type Data struct {
	EventType  string  `json:"event_type"`
	ID         string  `json:"id"`
	OccurredAt string  `json:"occurred_at"`
	Payload    Payload `json:"payload"`
	RecordType string  `json:"record_type"`
}

// FromTo defines
type FromTo struct {
	Carrier     string `json:"carrier"`
	LineType    string `json:"line_type"`
	PhoneNumber string `json:"phone_number"`
	Status      string `json:"status"`
}

// ConvertAddress returns converted commonaddress.Address
func (h *FromTo) ConvertAddress() *commonaddress.Address {
	return &commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: h.PhoneNumber,
	}
}

// ConvertMessage returns converted mmmessage.Message
func (h *Message) ConvertMessage(id uuid.UUID, customerID uuid.UUID) *mmmessage.Message {
	source := h.Data.Payload.From.ConvertAddress()

	// convert the to to the targets
	targets := []mmtarget.Target{}
	for _, to := range h.Data.Payload.To {
		destination := to.ConvertAddress()
		target := mmtarget.Target{
			Destination: *destination,
			Status:      mmtarget.StatusReceived,
			Parts:       h.Data.Payload.Parts,
		}
		targets = append(targets, target)
	}

	return &mmmessage.Message{
		ID:         id,
		CustomerID: customerID,
		Type:       mmmessage.Type(strings.ToLower(h.Data.Payload.Type)),
		Source:     source,
		Targets:    targets,

		ProviderName:        mmmessage.ProviderNameTelnyx,
		ProviderReferenceID: h.Data.Payload.ID,
		Text:                h.Data.Payload.Text,
		Medias:              []string{},
		Direction:           mmmessage.DirectionInbound,

		TMCreate: dbhandler.GetCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}
}
