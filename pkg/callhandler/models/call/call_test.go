package call

import (
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
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
		{"last status hangup", StatusHangup, HangupByRemote},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := CalculateHangupBy(tt.lastStatus); ret != tt.expectHangupby {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectHangupby, ret)
			}
		})
	}
}

func TestNewCallByChannel(t *testing.T) {
	type test struct {
		name              string
		ariChannelCreated string
		channelType       Type
		direction         Direction

		expectCall *Call
	}

	tests := []test{
		{
			"normarl",
			`{"type":"ChannelCreated","timestamp":"2020-05-02T20:56:51.498+0000","channel":{"id":"1588453011.231","name":"PJSIP/in-voipbin-00000074","state":"Ring","caller":{"name":"","number":"3001"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"9901146812420898","priority":1,"app_name":"","app_data":""},"creationtime":"2020-05-02T20:56:51.498+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`,
			TypeEcho,
			DirectionIncoming,

			&Call{
				AsteriskID: "42:01:0a:a4:00:03",
				ChannelID:  "1588453011.231",
				FlowID:     uuid.Nil,
				Type:       TypeEcho,

				Status: StatusRinging,

				Direction: DirectionIncoming,

				TMCreate: "2020-05-02T20:56:51.498",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, event, err := ari.Parse([]byte(tt.ariChannelCreated))
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			eventChannel := event.(*ari.ChannelCreated)
			channel := channel.NewChannelByChannelCreated(eventChannel)

			c := NewCallByChannel(channel, tt.channelType, tt.direction)
			if c == nil {
				t.Errorf("Wrong match. expect: not nil, got: nil")
			}

			c.ID = uuid.Nil
			c.Source = nil
			c.Destination = nil
			c.Data = nil

			if reflect.DeepEqual(*c, *tt.expectCall) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, c)
			}

		})
	}
}
