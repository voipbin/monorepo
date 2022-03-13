package messagebird

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
)

// Message defines messagebird's message
type Message struct {
	ID        string `json:"id"`
	Href      string `json:"href"`
	Direction string `json:"direction"`
	Type      string `json:"type"`

	Originator string `json:"originator"`
	Body       string `json:"body"`

	// Reference string `json:"reference"`	// shows null. Could not check the value type.
	// Validity string `json:"validity"`// shows null. Could not check the value type.
	Gateway int `json:"gateway"`
	// TypeDetails	// shows empty object. Could not check the value type.
	DataCoding string `json:"datacoding"`
	MClass     int    `json:"mclass"`
	// ScheduledDatetime string `json:"scheduledDatetime"` // shows null. Could not check the value type.
	CreatedDatetime string `json:"createdDatetime"`

	Recipients RecipientStruct `json:"recipients"`
}

// RecipientStruct defines
type RecipientStruct struct {
	TotalCount               int         `json:"totalCount"`
	TotalSentCount           int         `json:"totalSentCount"`
	TotalDeliveredCount      int         `json:"totalDeliveredCount"`
	TotalDeliveryFailedCount int         `json:"totalDeliveryFailedCount"`
	Items                    []Recipient `json:"items"`
}

// ConvertMessage converts to the message.Message
func (h *Message) ConvertMessage(id uuid.UUID, customerID uuid.UUID) *message.Message {
	res := &message.Message{
		ID:         id,
		CustomerID: customerID,
		Type:       message.Type(h.Type),
		Source: &cmaddress.Address{
			Type:   cmaddress.TypeTel,
			Target: h.Originator,
		},
		Targets:             []target.Target{},
		ProviderName:        message.ProviderNameMessagebird,
		ProviderReferenceID: h.ID,
		Text:                h.Body,
		Medias:              []string{},
	}

	res.Direction = message.DirectionInbound
	if h.Direction == "mt" {
		res.Direction = message.DirectionOutbound
	}

	// recipient
	for _, recipient := range h.Recipients.Items {
		t := recipient.ConvertTartget()
		res.Targets = append(res.Targets, *t)
	}

	return res
}
