package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
)

func Test_CallCreate(t *testing.T) {

	type test struct {
		name      string
		call      *call.Call
		expectRes *call.Call
	}

	tests := []test{
		{
			"have all",
			&call.Call{
				ID:           uuid.FromStringOrNil("f2e8b62a-2824-11eb-ba7a-b7fd7464daa3"),
				CustomerID:   uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				AsteriskID:   "3e:50:6b:43:bb:30",
				ChannelID:    "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				BridgeID:     "fe27852c-90c3-4f60-b357-69c44b605e6e",
				FlowID:       uuid.FromStringOrNil("069ba9f2-2825-11eb-be24-9f2570e3033c"),
				ActiveFlowID: uuid.FromStringOrNil("b41cfc24-5380-4c41-88ca-d22e95624445"),
				ConfbridgeID: uuid.FromStringOrNil("33115c61-1e97-486f-a337-f212f15a7284"),
				Type:         call.TypeFlow,

				MasterCallID: uuid.FromStringOrNil("d6fc5d4d-ad03-47e1-8453-a6e6a975304f"),
				ChainedCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("d6fc5d4d-ad03-47e1-8453-a6e6a975304f"),
				},
				RecordingID: uuid.FromStringOrNil("b73f769d-7b59-49a3-b8f2-23aad17ba476"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("b73f769d-7b59-49a3-b8f2-23aad17ba476"),
				},

				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "source",
					Name:       "test source name",
					Detail:     "test source detail",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000002",
					TargetName: "destination",
					Name:       "test destination name",
					Detail:     "test destination detail",
				},

				Status: call.StatusHangup,
				Data: map[string]string{
					"context": "call-in",
					"domain":  "pstn.voipbin.net",
				},
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				Direction:    call.DirectionIncoming,
				HangupBy:     call.HangupByLocal,
				HangupReason: call.HangupReasonNormal,

				DialrouteID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
				Dialroutes: []rmroute.Route{
					{
						ID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
					},
				},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",

				TMProgressing: "2020-04-18T03:22:17.995000",
				TMRinging:     "2020-04-18T03:22:17.995000",
				TMHangup:      DefaultTimeStamp,
			},
			&call.Call{
				ID:           uuid.FromStringOrNil("f2e8b62a-2824-11eb-ba7a-b7fd7464daa3"),
				CustomerID:   uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				AsteriskID:   "3e:50:6b:43:bb:30",
				ChannelID:    "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				BridgeID:     "fe27852c-90c3-4f60-b357-69c44b605e6e",
				FlowID:       uuid.FromStringOrNil("069ba9f2-2825-11eb-be24-9f2570e3033c"),
				ActiveFlowID: uuid.FromStringOrNil("b41cfc24-5380-4c41-88ca-d22e95624445"),
				ConfbridgeID: uuid.FromStringOrNil("33115c61-1e97-486f-a337-f212f15a7284"),
				Type:         call.TypeFlow,

				MasterCallID: uuid.FromStringOrNil("d6fc5d4d-ad03-47e1-8453-a6e6a975304f"),
				ChainedCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("d6fc5d4d-ad03-47e1-8453-a6e6a975304f"),
				},
				RecordingID: uuid.FromStringOrNil("b73f769d-7b59-49a3-b8f2-23aad17ba476"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("b73f769d-7b59-49a3-b8f2-23aad17ba476"),
				},

				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "source",
					Name:       "test source name",
					Detail:     "test source detail",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000002",
					TargetName: "destination",
					Name:       "test destination name",
					Detail:     "test destination detail",
				},

				Status: call.StatusHangup,
				Data: map[string]string{
					"context": "call-in",
					"domain":  "pstn.voipbin.net",
				},
				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				Direction:    call.DirectionIncoming,
				HangupBy:     call.HangupByLocal,
				HangupReason: call.HangupReasonNormal,

				DialrouteID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
				Dialroutes: []rmroute.Route{
					{
						ID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
					},
				},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",

				TMProgressing: "2020-04-18T03:22:17.995000",
				TMRinging:     "2020-04-18T03:22:17.995000",
				TMHangup:      DefaultTimeStamp,
			},
		},
		{
			"empty",

			&call.Call{
				ID: uuid.FromStringOrNil("64e31a36-b6fc-4df5-9a66-48f68ad60a70"),
			},
			&call.Call{
				ID:             uuid.FromStringOrNil("64e31a36-b6fc-4df5-9a66-48f68ad60a70"),
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[string]string{},
				Dialroutes:     []rmroute.Route{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created call. call: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallGets(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		minNum     int
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
			1,
		},
		{
			"empty",
			uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
			0,
		},
	}

	// creates calls for test
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewHandler(dbTest, mockCache)
	mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
	_ = h.CallCreate(context.Background(), &call.Call{ID: uuid.FromStringOrNil("1c6f0b6e-620b-11eb-bab1-e388ba38401b"), CustomerID: uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295")})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := h.CallGets(context.Background(), tt.customerID, 10, GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) < tt.minNum {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.minNum, len(res))
			}
		})
	}
}

func Test_CallSetStatus(t *testing.T) {
	type test struct {
		name string

		id       uuid.UUID
		status   call.Status
		tmUpdate string

		call      *call.Call
		expectRes *call.Call
	}

	tests := []test{
		{
			"test normal",

			uuid.FromStringOrNil("93d7aea3-4a93-4c58-8bce-d956a0f73ad6"),
			call.StatusProgressing,
			"2020-04-18T03:22:18.995000",

			&call.Call{
				ID: uuid.FromStringOrNil("93d7aea3-4a93-4c58-8bce-d956a0f73ad6"),

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&call.Call{
				ID: uuid.FromStringOrNil("93d7aea3-4a93-4c58-8bce-d956a0f73ad6"),

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,
				Data:      map[string]string{},

				Dialroutes: []rmroute.Route{},

				TMCreate:      "2020-04-18T03:22:17.995000",
				TMProgressing: "2020-04-18T03:22:18.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetStatus(ctx, tt.id, tt.status, tt.tmUpdate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallGetByChannelID(t *testing.T) {

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

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "2505d858-8687-11ea-8723-d35628256201",
				Type:       call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[string]string{},
				Dialroutes: []rmroute.Route{},

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

				Source: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "2aa510da-8687-11ea-b1b4-3f62cf9e4def",
				Type:       call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},

				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[string]string{},
				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			tt.call.ID = tt.id
			tt.call.FlowID = tt.flowID
			tt.expectCall.ID = tt.id
			tt.expectCall.FlowID = tt.flowID

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
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

func Test_CallCallSetHangup(t *testing.T) {

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

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			call.Call{
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusHangup,
				Direction: call.DirectionIncoming,

				HangupReason: call.HangupReasonNormal,
				HangupBy:     call.HangupByLocal,
				Data:         map[string]string{},
				Dialroutes:   []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:18.995000",
				TMHangup: "2020-04-18T03:22:18.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			tt.call.ID = tt.id
			tt.expectCall.ID = tt.id
			tt.expectCall.TMUpdate = tt.tmUpdate

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetHangup(context.Background(), tt.id, tt.reason, tt.hangupBy, tt.tmUpdate); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
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

func Test_CallSetFlowID(t *testing.T) {

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

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				FlowID: uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[string]string{},
				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetFlowID(context.Background(), tt.call.ID, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
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

func Test_CallSetConfbridgeID(t *testing.T) {

	type test struct {
		name         string
		confbridgeID uuid.UUID
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

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				ConfbridgeID: uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[string]string{},
				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetConfbridgeID(context.Background(), tt.call.ID, tt.confbridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
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

func Test_CallSetAction(t *testing.T) {

	type test struct {
		name   string
		call   *call.Call
		action *fmaction.Action

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

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&fmaction.Action{
				ID:     uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
				Type:   fmaction.TypeEcho,
				Option: []byte(`{"duration":180}`),
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:       call.TypeFlow,
				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Action: fmaction.Action{
					ID:     uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
					Type:   fmaction.TypeEcho,
					Option: []byte(`{"duration":180}`),
				},
				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[string]string{},
				Dialroutes: []rmroute.Route{},

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

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&fmaction.Action{
				ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
				Type: fmaction.TypeEcho,
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
				Type:       call.TypeFlow,
				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
					Type: fmaction.TypeEcho,
				},
				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[string]string{},
				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetAction(context.Background(), tt.call.ID, tt.action); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
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

func Test_CallSetMasterCallID(t *testing.T) {

	type test struct {
		name         string
		call         *call.Call
		masterCallID uuid.UUID

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("14649d2c-24fc-11eb-bb0b-9bd6970f725f"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "14daba5c-24fc-11eb-8f58-8b798baaf553",
				Type:       call.TypeFlow,
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("4a6ce0aa-24fc-11eb-aec0-4b97b9a2422a"),
			&call.Call{
				ID:             uuid.FromStringOrNil("14649d2c-24fc-11eb-bb0b-9bd6970f725f"),
				AsteriskID:     "3e:50:6b:43:bb:30",
				ChannelID:      "14daba5c-24fc-11eb-8f58-8b798baaf553",
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[string]string{},
				Dialroutes:     []rmroute.Route{},
				MasterCallID:   uuid.FromStringOrNil("4a6ce0aa-24fc-11eb-aec0-4b97b9a2422a"),
				TMCreate:       "2020-04-18T03:22:17.995000",
			},
		},
		{
			"set nil",
			&call.Call{
				ID:       uuid.FromStringOrNil("665db8f2-2501-11eb-86ce-f3a50eef6f26"),
				Type:     call.TypeFlow,
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.Nil,
			&call.Call{
				ID:             uuid.FromStringOrNil("665db8f2-2501-11eb-86ce-f3a50eef6f26"),
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[string]string{},
				Dialroutes:     []rmroute.Route{}, TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetMasterCallID(context.Background(), tt.call.ID, tt.masterCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetRecordID(t *testing.T) {

	type test struct {
		name     string
		call     *call.Call
		reocrdID uuid.UUID

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:         uuid.FromStringOrNil("46ab9ad8-282b-11eb-82c3-6782faf5e030"),
				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "4e2fe520-282b-11eb-ad66-b777dce59261",
				Type:       call.TypeFlow,
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("4e847572-282b-11eb-9c58-97622e4406e2"),
			&call.Call{
				ID:             uuid.FromStringOrNil("46ab9ad8-282b-11eb-82c3-6782faf5e030"),
				AsteriskID:     "3e:50:6b:43:bb:30",
				ChannelID:      "4e2fe520-282b-11eb-ad66-b777dce59261",
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[string]string{},
				Dialroutes:     []rmroute.Route{},

				RecordingID: uuid.FromStringOrNil("4e847572-282b-11eb-9c58-97622e4406e2"),
				TMCreate:    "2020-04-18T03:22:17.995000",
			},
		},
		{
			"set empty",
			&call.Call{
				ID:       uuid.FromStringOrNil("7b3e197e-282b-11eb-956d-4feb054947db"),
				Type:     call.TypeFlow,
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.Nil,
			&call.Call{
				ID:             uuid.FromStringOrNil("7b3e197e-282b-11eb-956d-4feb054947db"),
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[string]string{},
				Dialroutes:     []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetRecordID(context.Background(), tt.call.ID, tt.reocrdID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}
