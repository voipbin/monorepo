package address

import (
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// Address contains source/destination detail info.
type Address struct {
	Type       Type   `json:"type"`        // Type of address. must be one of ["sip", "tel"].
	Target     string `json:"target"`      // Target address. If the type is 'tel' type, the terget must follow the E.164 format(https://www.itu.int/rec/T-REC-E.164/en).
	TargetName string `json:"target_name"` // Target's shown name.
	Name       string `json:"name"`        // Name.
	Detail     string `json:"detail"`      // Detail.
}

// Type type
type Type string

// List of type
const (
	TypeSIP Type = "sip"
	TypeTel Type = "tel"
)

// ConvertToAddress define
func ConvertToAddress(h cmaddress.Address) *Address {
	a := &Address{
		Type:       Type(h.Type),
		Target:     h.Target,
		TargetName: h.TargetName,
		Name:       h.Name,
		Detail:     h.Detail,
	}

	return a
}

// ConvertToCMAddress define
func ConvertToCMAddress(h *Address) *cmaddress.Address {
	a := &cmaddress.Address{
		Type:       cmaddress.Type(h.Type),
		Target:     h.Target,
		TargetName: h.TargetName,
		Name:       h.Name,
		Detail:     h.Detail,
	}

	return a

}
