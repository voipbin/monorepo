package messagebird

import (
	"strconv"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-message-manager/models/target"
)

// Recipient defines
type Recipient struct {
	Recipient        int    `json:"recipient"`
	Status           string `json:"status"`
	StatusDatetime   string `json:"statusDatetime"`
	MessagePartCount int    `json:"messagePartCount"`
}

// ConvertTartget converts to the target.Target
func (h *Recipient) ConvertTartget() *target.Target {
	return &target.Target{
		Destination: commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+" + strconv.Itoa(h.Recipient),
		},
		Status: target.Status(h.Status),
		Parts:  h.MessagePartCount,
	}
}
