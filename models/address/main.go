package address

import (
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// Address contains source/destination detail info.
type Address struct {
	Type       Type   `json:"type"`        // type of address
	Target     string `json:"target"`      // address endpoint
	TargetName string `json:"target_name"` // address's name.
	Name       string `json:"name"`        // name
	Detail     string `json:"detail"`      // detail description.
}

// Type type
type Type string

// List of CallAddressType
const (
	TypeAgent    Type = "agent"
	TypeEndpoint Type = "endpoint"
	TypeLine     Type = "line"
	TypeSIP      Type = "sip"
	TypeTel      Type = "tel"
)

// CreateAddressByChannelSource creates and return the Address using channel's source.
func CreateAddressByChannelSource(cn *channel.Channel) *Address {
	r := &Address{
		Type:       TypeTel,
		Target:     cn.SourceNumber,
		TargetName: cn.SourceName,
	}
	return r
}

// CreateAddressByChannelDestination creates and return the Address using channel's destination.
func CreateAddressByChannelDestination(cn *channel.Channel) *Address {
	r := &Address{
		Type:       TypeTel,
		Target:     cn.DestinationNumber,
		TargetName: cn.DestinationName,
	}
	return r
}

// ParseAddressByCallerID parsing the ari's CallerID and returns Address
func ParseAddressByCallerID(e *ari.CallerID) *Address {
	r := &Address{
		Type:       TypeTel,
		Target:     e.Number,
		TargetName: e.Name,
	}

	return r
}

// NewAddressByDialplan parsing the ari's CallerID and returns Address
func NewAddressByDialplan(e *ari.DialplanCEP) *Address {
	r := &Address{
		Type:   TypeTel,
		Target: e.Exten,
	}

	return r
}

// ConvertFromCMAddress convert the *cmaddrees.Address to the *Address
func ConvertFromCMAddress(addr *cmaddress.Address) *Address {
	return &Address{
		Type:       Type(addr.Type),
		Target:     addr.Target,
		TargetName: addr.TargetName,
		Name:       addr.Name,
		Detail:     addr.Detail,
	}
}

// ConvertToCMAddress converts the *Address to the *cmaddress.Address
func ConvertToCMAddress(addr *Address) *cmaddress.Address {
	return &cmaddress.Address{
		Type:       cmaddress.Type(addr.Type),
		Target:     addr.Target,
		TargetName: addr.TargetName,
		Name:       addr.Name,
		Detail:     addr.Detail,
	}
}
