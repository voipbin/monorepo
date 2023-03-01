package channelhandler

import (
	"fmt"
	"strings"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
)

// AddressGetSource gets the source address from the given channel
func (h *channelHandler) AddressGetSource(cn *channel.Channel, addressType commonaddress.Type) *commonaddress.Address {

	target := ""
	if addressType == commonaddress.TypeEndpoint {
		domainName := strings.TrimSuffix(cn.StasisData["domain"], common.DomainSIPSuffix)
		target = fmt.Sprintf("%s@%s", cn.SourceNumber, domainName)
	} else {
		target = cn.SourceNumber
	}

	res := &commonaddress.Address{
		Type:       addressType,
		Target:     target,
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

// AddressGetDestinationWithoutSpecificType gets the destination address type and target from the given channel
// this is not designed for general use. it's valid only for SIP incoming type.
func (h *channelHandler) AddressGetDestinationWithoutSpecificType(cn *channel.Channel) *commonaddress.Address {

	var addressType commonaddress.Type
	var target string
	if strings.HasPrefix(cn.DestinationNumber, "+") {
		addressType = commonaddress.TypeTel
		target = cn.DestinationNumber
	} else if strings.HasPrefix(cn.DestinationNumber, string(commonaddress.TypeAgent)+"-") {
		addressType = commonaddress.TypeAgent
		target = strings.TrimPrefix(cn.DestinationNumber, string(commonaddress.TypeAgent)+"-")
	} else if strings.HasPrefix(cn.DestinationNumber, string(commonaddress.TypeConference)+"-") {
		addressType = commonaddress.TypeConference
		target = strings.TrimPrefix(cn.DestinationNumber, string(commonaddress.TypeConference)+"-")
	} else if strings.HasPrefix(cn.DestinationNumber, string(commonaddress.TypeLine)+"-") {
		addressType = commonaddress.TypeLine
		target = strings.TrimPrefix(cn.DestinationNumber, string(commonaddress.TypeLine)+"-")
	} else {
		addressType = commonaddress.TypeEndpoint
		target = cn.DestinationNumber + "@" + strings.TrimSuffix(cn.StasisData["domain"], common.DomainSIPSuffix)
	}

	res := &commonaddress.Address{
		Type:       addressType,
		Target:     target,
		TargetName: cn.DestinationName,
	}

	return res
}
