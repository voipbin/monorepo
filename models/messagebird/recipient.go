package messagebird

import (
	"strconv"

	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
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
		Destination: cmaddress.Address{
			Type:   cmaddress.TypeTel,
			Target: "+" + strconv.Itoa(h.Recipient),
		},
		Status:   target.Status(h.Status),
		Parts:    h.MessagePartCount,
		TMUpdate: dbhandler.GetCurTime(),
	}
}
