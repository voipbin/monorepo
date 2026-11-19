package messagehandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	nmnumber "monorepo/bin-number-manager/models/number"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// defaultProviderOrder is the order of the providers of a message whose source number has no owner that can send
// messages: telnyx first, and messagebird for the targets that telnyx definitively refused. It is never changed.
var defaultProviderOrder = []message.ProviderName{
	message.ProviderNameTelnyx,
	message.ProviderNameMessagebird,
}

// providerOrder returns the order in which the providers are tried for one message. The provider that owns the
// source number (owner) comes first when it can send, and the rest of the default order follows in its own order.
// Twilio can send only when its credentials are set, and it is never added to the order of a number that another
// provider owns. An owner that is empty or cannot send gives the default order. The result is a new slice for
// every call: the order belongs to the message and no shared list is changed.
func (h *messageHandler) providerOrder(ctx context.Context, owner nmnumber.ProviderName) []message.ProviderName {
	first := message.ProviderName("")
	switch message.ProviderName(owner) {
	case message.ProviderNameTelnyx:
		first = message.ProviderNameTelnyx
	case message.ProviderNameMessagebird:
		first = message.ProviderNameMessagebird
	case message.ProviderNameTwilio:
		if h.twilioAvailable != nil && h.twilioAvailable(ctx) {
			first = message.ProviderNameTwilio
		}
	}

	res := make([]message.ProviderName, 0, len(defaultProviderOrder)+1)
	if first != "" {
		res = append(res, first)
	}
	for _, name := range defaultProviderOrder {
		if name != first {
			res = append(res, name)
		}
	}

	return res
}

// mmsEligible tells whether the provider may send the mms to the target. Telnyx sends to every target.
// Messagebird and Twilio send mms inside the US and Canada only, so the source and the destination must both be
// +1 numbers.
func mmsEligible(name message.ProviderName, source *commonaddress.Address, t target.Target) bool {
	switch name {
	case message.ProviderNameMessagebird, message.ProviderNameTwilio:
		return mmsFallbackEligible(source, t)
	default:
		return true
	}
}

// smsProvider returns the sms provider of the name, or nil when there is none.
func (h *messageHandler) smsProvider(name message.ProviderName) SMSProvider {
	return h.smsProviders[name]
}

// mmsProvider returns the mms provider of the name, or nil when there is none.
func (h *messageHandler) mmsProvider(name message.ProviderName) MMSProvider {
	return h.mmsProviders[name]
}
