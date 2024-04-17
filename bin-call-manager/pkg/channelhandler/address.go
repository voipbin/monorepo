package channelhandler

import (
	"fmt"
	"net/url"
	"strings"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// AddressGetSource gets the source address from the given channel
func (h *channelHandler) AddressGetSource(cn *channel.Channel, addressType commonaddress.Type) *commonaddress.Address {

	target := ""
	if addressType == commonaddress.TypeSIP {
		domainName := cn.StasisData[channel.StasisDataTypeDomain]
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
	var tmpTarget string

	// parse tmp target
	if strings.HasPrefix(cn.DestinationNumber, "+") {
		tmpTarget = cn.DestinationNumber
	} else {
		tmp, err := url.QueryUnescape(cn.DestinationNumber)
		if err != nil {
			tmp = cn.DestinationNumber
		}
		tmpTarget = tmp
	}

	// get address type and target
	switch {
	case strings.HasPrefix(tmpTarget, "+"):
		addressType = commonaddress.TypeTel
		target = tmpTarget

	case strings.HasPrefix(tmpTarget, string(commonaddress.TypeAgent)+":"):
		addressType = commonaddress.TypeAgent
		target = strings.TrimPrefix(tmpTarget, string(commonaddress.TypeAgent)+":")

	case strings.HasPrefix(tmpTarget, string(commonaddress.TypeConference)+":"):
		addressType = commonaddress.TypeConference
		target = strings.TrimPrefix(tmpTarget, string(commonaddress.TypeConference)+":")

	case strings.HasPrefix(tmpTarget, string(commonaddress.TypeLine)+":"):
		addressType = commonaddress.TypeLine
		target = strings.TrimPrefix(tmpTarget, string(commonaddress.TypeLine)+":")

	default:
		addressType = commonaddress.TypeExtension
		target = tmpTarget
	}

	res := &commonaddress.Address{
		Type:       addressType,
		Target:     target,
		TargetName: cn.DestinationName,
	}

	return res
}
