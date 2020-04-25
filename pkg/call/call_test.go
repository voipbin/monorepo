package call

import (
	"reflect"
	"testing"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

func TestParseAddressByCallerID(t *testing.T) {
	type test struct {
		name          string
		callerID      *ari.CallerID
		expectAddress *Address
	}

	tests := []test{
		{
			"normal",
			&ari.CallerID{
				Name:   "test",
				Number: "123456789",
			},
			&Address{
				Type:   AddressTypeTel,
				Target: "123456789",
				Name:   "test",
			},
		},
		{
			"has empty name",
			&ari.CallerID{
				Name:   "",
				Number: "123456789",
			},
			&Address{
				Type:   AddressTypeTel,
				Target: "123456789",
				Name:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := ParseAddressByCallerID(tt.callerID)

			if !reflect.DeepEqual(address, tt.expectAddress) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAddress, address)
			}
		})
	}
}

func TestParseAddressByDialplan(t *testing.T) {
	type test struct {
		name          string
		dialplan      *ari.DialplanCEP
		expectAddress *Address
	}

	tests := []test{
		{
			"test normal",
			&ari.DialplanCEP{
				Context:  "in-voipbin",
				Exten:    "12345679999",
				Priority: 1,
				AppName:  "Stasis",
				AppData:  "test=gogo",
			},
			&Address{
				Type:   AddressTypeTel,
				Target: "12345679999",
				Name:   "",
			},
		},
		{
			"dialplan has exten only",
			&ari.DialplanCEP{
				Exten: "193884272342",
			},
			&Address{
				Type:   AddressTypeTel,
				Target: "193884272342",
				Name:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address := NewAddressByDialplan(tt.dialplan)
			if !reflect.DeepEqual(address, tt.expectAddress) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectAddress, address)
			}
		})
	}
}
