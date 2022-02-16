package address

import (
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// Address contains source/destination detail info.
type Address struct {
	Type       Type   `json:"type"`        // type of address
	Target     string `json:"target"`      // address endpoint
	TargetName string `json:"target_name"` // address's name.
	Name       string `json:"name"`        // parsed name
	Detail     string `json:"detail"`
}

// Type type
type Type string

// List of CallAddressType
const (
	TypeAgent    Type = "agent"
	TypeEndpoint Type = "endpoint"
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
