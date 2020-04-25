package dbhandler

import (
	"context"
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"

	_ "github.com/mattn/go-sqlite3"
)

func TestCallCreate(t *testing.T) {
	type test struct {
		name   string
		id     uuid.UUID
		flowID uuid.UUID

		call       call.Call
		expectCall call.Call
	}

	tests := []test{
		{
			"test normal",
			uuid.NewV4(),
			uuid.NewV4(),
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"test normal has source address type sip",
			uuid.NewV4(),
			uuid.NewV4(),
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "bd610e10-84ed-11ea-b6e1-ef9d10ec3de6",
				Type:       call.TypeFlow,

				Source: &call.Address{
					Type: call.AddressTypeSIP,
				},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "bd610e10-84ed-11ea-b6e1-ef9d10ec3de6",
				Type:       call.TypeFlow,

				Source: &call.Address{
					Type: call.AddressTypeSIP,
				},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			tt.call.ID = tt.id
			tt.call.FlowID = tt.flowID
			tt.expectCall.ID = tt.id
			tt.expectCall.FlowID = tt.flowID

			if err := h.CallCreate(context.Background(), &tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created call. call: %v", res)

			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestCallSetStatus(t *testing.T) {
	type test struct {
		name     string
		id       uuid.UUID
		flowID   uuid.UUID
		status   call.Status
		tmUpdate string

		call       *call.Call
		expectCall call.Call
	}

	tests := []test{
		{
			"test normal",
			uuid.NewV4(),
			uuid.NewV4(),
			call.StatusProgressing,
			"2020-04-18T03:22:18.995000",
			&call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			tt.call.ID = tt.id
			tt.call.FlowID = tt.flowID
			tt.expectCall.ID = tt.id
			tt.expectCall.FlowID = tt.flowID
			tt.expectCall.TMUpdate = tt.tmUpdate

			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CallSetStatus(context.Background(), tt.id, tt.status, tt.tmUpdate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestCallGetByChannelID(t *testing.T) {
	type test struct {
		name   string
		id     uuid.UUID
		flowID uuid.UUID

		call       call.Call
		expectCall call.Call
	}

	tests := []test{
		{
			"test normal",
			uuid.NewV4(),
			uuid.NewV4(),
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "2505d858-8687-11ea-8723-d35628256201",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "2505d858-8687-11ea-8723-d35628256201",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"test normal has source address type sip",
			uuid.NewV4(),
			uuid.NewV4(),
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "2aa510da-8687-11ea-b1b4-3f62cf9e4def",
				Type:       call.TypeFlow,

				Source: &call.Address{
					Type: call.AddressTypeSIP,
				},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "2aa510da-8687-11ea-b1b4-3f62cf9e4def",
				Type:       call.TypeFlow,

				Source: &call.Address{
					Type: call.AddressTypeSIP,
				},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			tt.call.ID = tt.id
			tt.call.FlowID = tt.flowID
			tt.expectCall.ID = tt.id
			tt.expectCall.FlowID = tt.flowID

			if err := h.CallCreate(context.Background(), &tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CallGetByChannelID(context.Background(), tt.call.ChannelID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created call. call: %v", res)

			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestCallCallSetHangup(t *testing.T) {
	type test struct {
		name     string
		id       uuid.UUID
		reason   call.HangupReason
		hangupBy call.HangupBy
		tmUpdate string

		call       *call.Call
		expectCall call.Call
	}

	tests := []test{
		{
			"test normal",
			uuid.NewV4(),
			call.HangupReasonNormal,
			call.HangupByLocal,
			"2020-04-18T03:22:18.995000",
			&call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusHangup,
				Direction: call.DirectionIncoming,

				HangupReason: call.HangupReasonNormal,
				HangupBy:     call.HangupByLocal,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:18.995000",
				TMHangup: "2020-04-18T03:22:18.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			tt.call.ID = tt.id
			tt.expectCall.ID = tt.id
			tt.expectCall.TMUpdate = tt.tmUpdate

			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CallSetHangup(context.Background(), tt.id, tt.reason, tt.hangupBy, tt.tmUpdate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}
