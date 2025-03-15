package emailhandler

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"net/mail"
)

func (h *emailHandler) validateEmailAddress(addr commonaddress.Address) bool {

	if addr.Type != commonaddress.TypeEmail {
		return false
	}

	_, err := mail.ParseAddress(addr.Target)
	return err == nil
}
