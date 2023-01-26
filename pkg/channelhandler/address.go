package channelhandler

import (
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// AddressGetSource gets the source address from the given channel
func (h *channelHandler) AddressGetSource(cn *channel.Channel, addressType commonaddress.Type) *commonaddress.Address {
	res := &commonaddress.Address{
		Type:       addressType,
		Target:     cn.SourceNumber,
		TargetName: cn.SourceName,
	}

	return res
}

// AddressGetDestination gets the destination address from the given channel
func (h *channelHandler) AddressGetDestination(cn *channel.Channel, addressType commonaddress.Type) *commonaddress.Address {
	res := &commonaddress.Address{
		Type:       addressType,
		Target:     cn.DestinationNumber,
		TargetName: cn.DestinationName,
	}

	return res
}
