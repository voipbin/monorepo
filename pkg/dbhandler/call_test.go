package dbhandler

import (
	"context"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"

	_ "github.com/mattn/go-sqlite3"
)

func TestCallCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			uuid.Must(uuid.NewV4()),
			uuid.Must(uuid.NewV4()),
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
			uuid.Must(uuid.NewV4()),
			uuid.Must(uuid.NewV4()),
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
			h := NewHandler(dbTest, mockCache)

			tt.call.ID = tt.id
			tt.call.FlowID = tt.flowID
			tt.expectCall.ID = tt.id
			tt.expectCall.FlowID = tt.flowID

			if err := h.CallCreate(context.Background(), &tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			uuid.Must(uuid.NewV4()),
			uuid.Must(uuid.NewV4()),
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

				TMCreate:      "2020-04-18T03:22:17.995000",
				TMProgressing: "2020-04-18T03:22:18.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

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

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectCall.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestCallGetByChannelIDAndAsteriskID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			uuid.Must(uuid.NewV4()),
			uuid.Must(uuid.NewV4()),
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
			uuid.Must(uuid.NewV4()),
			uuid.Must(uuid.NewV4()),
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
			h := NewHandler(dbTest, mockCache)

			tt.call.ID = tt.id
			tt.call.FlowID = tt.flowID
			tt.expectCall.ID = tt.id
			tt.expectCall.FlowID = tt.flowID

			if err := h.CallCreate(context.Background(), &tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

			res, err := h.CallGetByChannelIDAndAsteriskID(context.Background(), tt.call.ChannelID, tt.call.AsteriskID)
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			uuid.Must(uuid.NewV4()),
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
			h := NewHandler(dbTest, mockCache)

			tt.call.ID = tt.id
			tt.expectCall.ID = tt.id
			tt.expectCall.TMUpdate = tt.tmUpdate

			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CallSetHangup(context.Background(), tt.id, tt.reason, tt.hangupBy, tt.tmUpdate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

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

func TestCallSetFlowID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name   string
		flowID uuid.UUID
		call   *call.Call

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),
			&call.Call{
				ID:         uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				FlowID: uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CallSetFlowID(context.Background(), tt.call.ID, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectCall, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestCallSetConferenceID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		conferenceID uuid.UUID
		call         *call.Call

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),
			&call.Call{
				ID:         uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				ConfID: uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CallSetConferenceID(context.Background(), tt.call.ID, tt.conferenceID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectCall, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestCallSetAction(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name   string
		call   *call.Call
		action *action.Action

		expectCall *call.Call
	}

	tests := []test{
		{
			"echo option duration",
			&call.Call{
				ID:         uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,
				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&action.Action{
				ID:     uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
				Type:   action.TypeEcho,
				Next:   uuid.Nil,
				Option: []byte(`{"duration":180}`),
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,
				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				Source:      &call.Address{},
				Destination: &call.Address{},

				Action: action.Action{
					ID:     uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
					Type:   action.TypeEcho,
					Next:   uuid.Nil,
					Option: []byte(`{"duration":180}`),
				},
				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},

		{
			"echo option empty",
			&call.Call{
				ID:         uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
				Type:       call.TypeFlow,
				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				Source:      &call.Address{},
				Destination: &call.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
				Type: action.TypeEcho,
				Next: uuid.Nil,
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
				Type:       call.TypeFlow,
				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				Source:      &call.Address{},
				Destination: &call.Address{},

				Action: action.Action{
					ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
					Type: action.TypeEcho,
					Next: uuid.Nil,
				},
				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CallSetAction(context.Background(), tt.call.ID, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())

			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}
