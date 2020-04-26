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

func TestIsUpdatableStatus(t *testing.T) {
	type test struct {
		name      string
		oldStatus Status
		newStatus Status
		expect    bool
	}

	tests := []test{
		{"dialing->dialing", StatusDialing, StatusDialing, false},
		{"dialing->ringing", StatusDialing, StatusRinging, true},
		{"dialing->progressing", StatusDialing, StatusProgressing, true},
		{"dialing->terminating", StatusDialing, StatusTerminating, true},
		{"dialing->canceling", StatusDialing, StatusCanceling, true},
		{"dialing->hangup", StatusDialing, StatusHangup, true},

		{"ringing->dialing", StatusRinging, StatusDialing, false},
		{"ringing->ringing", StatusRinging, StatusRinging, false},
		{"ringing->progressing", StatusRinging, StatusProgressing, true},
		{"ringing->terminating", StatusRinging, StatusTerminating, true},
		{"ringing->canceling", StatusRinging, StatusCanceling, true},
		{"ringing->hangup", StatusRinging, StatusHangup, true},

		{"progressing->dialing", StatusProgressing, StatusDialing, false},
		{"progressing->ringing", StatusProgressing, StatusRinging, false},
		{"progressing->progressing", StatusProgressing, StatusProgressing, false},
		{"progressing->terminating", StatusProgressing, StatusTerminating, true},
		{"progressing->canceling", StatusProgressing, StatusCanceling, false},
		{"progressing->hangup", StatusProgressing, StatusHangup, true},

		{"terminating->dialing", StatusTerminating, StatusDialing, false},
		{"terminating->ringing", StatusTerminating, StatusRinging, false},
		{"terminating->progressing", StatusTerminating, StatusProgressing, false},
		{"terminating->terminating", StatusTerminating, StatusTerminating, false},
		{"terminating->canceling", StatusTerminating, StatusCanceling, false},
		{"terminating->hangup", StatusTerminating, StatusHangup, true},

		{"canceling->dialing", StatusCanceling, StatusDialing, false},
		{"canceling->ringing", StatusCanceling, StatusRinging, false},
		{"canceling->progressing", StatusCanceling, StatusProgressing, false},
		{"canceling->terminating", StatusCanceling, StatusTerminating, false},
		{"canceling->canceling", StatusCanceling, StatusCanceling, false},
		{"canceling->hangup", StatusCanceling, StatusHangup, true},

		{"hangup->dialing", StatusHangup, StatusDialing, false},
		{"hangup->ringing", StatusHangup, StatusRinging, false},
		{"hangup->progressing", StatusHangup, StatusProgressing, false},
		{"hangup->terminating", StatusHangup, StatusTerminating, false},
		{"hangup->canceling", StatusHangup, StatusCanceling, false},
		{"hangup->hangup", StatusHangup, StatusHangup, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := IsUpdatableStatus(tt.oldStatus, tt.newStatus); ret != tt.expect {
				t.Errorf("Wrong match. expect: %t, got: %t", tt.expect, ret)
			}
		})
	}
}

func TestGetStatusByChannelState(t *testing.T) {
	type test struct {
		name         string
		chanState    ari.ChannelState
		expectStatus Status
	}

	tests := []test{
		{"state down", ari.ChannelStateDown, StatusDialing},
		{"state rsrvd", ari.ChannelStateRsrvd, StatusDialing},
		{"state offhook", ari.ChannelStateOffHook, StatusDialing},
		{"state dialing", ari.ChannelStateDialing, StatusDialing},
		{"state busy", ari.ChannelStateBusy, StatusDialing},
		{"state dialing off hook", ari.ChannelStateDialingOffHook, StatusDialing},
		{"state pre ring", ari.ChannelStatePreRing, StatusDialing},
		{"state unknown", ari.ChannelStateUnknown, StatusDialing},

		{"state ringing", ari.ChannelStateRinging, StatusRinging},
		{"state ring", ari.ChannelStateRing, StatusRinging},

		{"state up", ari.ChannelStateUp, StatusProgressing},
		{"state mute", ari.ChannelStateMute, StatusProgressing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := GetStatusByChannelState(tt.chanState); ret != tt.expectStatus {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectStatus, ret)
			}
		})
	}
}

func TestCalculateHangupBy(t *testing.T) {
	type test struct {
		name           string
		lastStatus     Status
		expectHangupby HangupBy
	}

	tests := []test{
		{"last status dialing", StatusDialing, HangupByRemote},
		{"last status ringing", StatusRinging, HangupByRemote},
		{"last status progressing", StatusProgressing, HangupByRemote},
		{"last status terminating", StatusTerminating, HangupByLocal},
		{"last status canceling", StatusCanceling, HangupByLocal},
		{"last status hangup", StatusHangup, HangupByLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := CalculateHangupBy(tt.lastStatus); ret != tt.expectHangupby {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectHangupby, ret)
			}
		})
	}
}
