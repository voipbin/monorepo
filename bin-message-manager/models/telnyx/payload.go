package telnyx

// Payload defines
// {
// 	"cc": [],
// 	"completed_at": null,
// 	"cost": null,
// 	"direction": "inbound",
// 	"encoding": "GSM-7",
// 	"errors": [],
// 	"from": {
// 		"carrier": "",
// 		"line_type": "",
// 		"phone_number": "+75973"
// 	},
// 	"id": "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
// 	"media": [],
// 	"messaging_profile_id": "40017f8e-49bd-4f16-9e3d-ef103f916228",
// 	"organization_id": "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
// 	"parts": 1,
// 	"received_at": "2022-03-15T16:16:23.466+00:00",
// 	"record_type": "message",
// 	"sent_at": null,
// 	"subject": "",
// 	"tags": [],
// 	"text": "pchero21:\nTest message from skype.",
// 	"to": [
// 		{
// 		"carrier": "Telnyx",
// 		"line_type": "Wireless",
// 		"phone_number": "+15734531118",
// 		"status": "webhook_delivered"
// 		}
// 	],
// 	"type": "SMS",
// 	"valid_until": null,
// 	"webhook_failover_url": null,
// 	"webhook_url": "https://en7evajwhmqbt.x.pipedream.net"
// }
type Payload struct {
	// CC []string // empty array
	// CompletedAt	// null
	// Cost	// null
	Direction string `json:"direction"`
	Encoding  string `json:"encoding"`
	// Errors // empty array
	From FromTo `json:"from"`
	ID   string `json:"id"`
	// Media // empty array
	MessagingProfileID string `json:"messaging_profile_id"`
	OrganizationID     string `json:"organization_id"`
	Parts              int    `json:"parts"`
	ReceivedAt         string `json:"received_at"`
	RecordType         string `json:"record_type"`
	// SentAt // null
	Subject string `json:"subject"`
	// Tags // empty array
	Text string   `json:"text"`
	To   []FromTo `json:"to"`
	Type string   `json:"type"`
	// ValidUntil // null
	// WebhookFailoverURL // null
	WebhookURL string `json:"webhook_url"`
}
