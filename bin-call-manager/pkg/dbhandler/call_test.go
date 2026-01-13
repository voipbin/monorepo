package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	rmroute "monorepo/bin-route-manager/models/route"

	uuid "github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_CallCreate(t *testing.T) {

	type test struct {
		name            string
		call            *call.Call
		responseCurTime string

		expectRes *call.Call
	}

	tests := []test{
		{
			"have all",
			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f2e8b62a-2824-11eb-ba7a-b7fd7464daa3"),
					CustomerID: uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("7277e05c-2bf8-11ef-a54d-dfe525b51ec5"),
				},

				ChannelID:    "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				BridgeID:     "fe27852c-90c3-4f60-b357-69c44b605e6e",
				FlowID:       uuid.FromStringOrNil("069ba9f2-2825-11eb-be24-9f2570e3033c"),
				ActiveflowID: uuid.FromStringOrNil("b41cfc24-5380-4c41-88ca-d22e95624445"),
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
				GroupcallID: uuid.FromStringOrNil("41736b26-b790-11ed-8bdd-bb87802b1ae8"),

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
				Data:   map[call.DataType]string{},
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
			},
			"2020-04-18T03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f2e8b62a-2824-11eb-ba7a-b7fd7464daa3"),
					CustomerID: uuid.FromStringOrNil("876fb2c6-796d-4925-aaf0-570b0a4323bb"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("7277e05c-2bf8-11ef-a54d-dfe525b51ec5"),
				},

				ChannelID:    "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				BridgeID:     "fe27852c-90c3-4f60-b357-69c44b605e6e",
				FlowID:       uuid.FromStringOrNil("069ba9f2-2825-11eb-be24-9f2570e3033c"),
				ActiveflowID: uuid.FromStringOrNil("b41cfc24-5380-4c41-88ca-d22e95624445"),
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
				GroupcallID: uuid.FromStringOrNil("41736b26-b790-11ed-8bdd-bb87802b1ae8"),

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
				Data:   map[call.DataType]string{},
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

				TMProgressing: DefaultTimeStamp,
				TMRinging:     DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"empty",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("64e31a36-b6fc-4df5-9a66-48f68ad60a70"),
				},
			},
			"2020-04-18T03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("64e31a36-b6fc-4df5-9a66-48f68ad60a70"),
				},
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMProgressing: DefaultTimeStamp,
				TMRinging:     DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
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
		name  string
		calls []*call.Call

		filters map[call.Field]any

		responseCurTime string

		expectRes []*call.Call
	}

	tests := []test{
		{
			name: "normal",
			calls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9e8a2df2-c8ea-4fea-b982-48103dd04a9e"),
						CustomerID: uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("73b3938c-b79c-4712-8feb-d465bab28441"),
						CustomerID: uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
					},
				},
			},

			filters: map[call.Field]any{
				call.FieldCustomerID: uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
				call.FieldDeleted:    false,
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectRes: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9e8a2df2-c8ea-4fea-b982-48103dd04a9e"),
						CustomerID: uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
					},

					ChainedCallIDs: []uuid.UUID{},
					RecordingIDs:   []uuid.UUID{},
					Data:           map[call.DataType]string{},
					Dialroutes:     []rmroute.Route{},

					TMProgressing: DefaultTimeStamp,
					TMRinging:     DefaultTimeStamp,
					TMHangup:      DefaultTimeStamp,

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("73b3938c-b79c-4712-8feb-d465bab28441"),
						CustomerID: uuid.FromStringOrNil("739625ca-7f43-11ec-8d25-4f519d029295"),
					},

					ChainedCallIDs: []uuid.UUID{},
					RecordingIDs:   []uuid.UUID{},
					Data:           map[call.DataType]string{},
					Dialroutes:     []rmroute.Route{},

					TMProgressing: DefaultTimeStamp,
					TMRinging:     DefaultTimeStamp,
					TMHangup:      DefaultTimeStamp,

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
		{
			name:  "empty",
			calls: []*call.Call{},

			filters: map[call.Field]any{
				call.FieldCustomerID: uuid.FromStringOrNil("cd1bc551-c4e8-45c8-a457-d41d65e1f18c"),
				call.FieldDeleted:    false,
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes:       []*call.Call{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.calls {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().CallSet(ctx, gomock.Any())
				_ = h.CallCreate(ctx, c)
			}

			res, err := h.CallGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallGets_delete(t *testing.T) {

	type test struct {
		name  string
		calls []*call.Call

		customerID uuid.UUID

		deleteCallIDs []uuid.UUID

		responseCurTime string

		expectRes []*call.Call
	}

	tests := []test{
		{
			"Create 2 calls but 1 call deleted",
			[]*call.Call{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("e9decd77-1c77-4ef0-bb92-547fd40cd911"),
						CustomerID: uuid.FromStringOrNil("c6cc16b0-03d5-4332-b1d5-0c0b68e29847"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b4f1f04c-98bc-458c-8dad-726be07da49a"),
						CustomerID: uuid.FromStringOrNil("c6cc16b0-03d5-4332-b1d5-0c0b68e29847"),
					},
				},
			},
			uuid.FromStringOrNil("c6cc16b0-03d5-4332-b1d5-0c0b68e29847"),

			[]uuid.UUID{
				uuid.FromStringOrNil("e9decd77-1c77-4ef0-bb92-547fd40cd911"),
			},

			"2020-04-18 03:22:17.995000",

			[]*call.Call{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b4f1f04c-98bc-458c-8dad-726be07da49a"),
						CustomerID: uuid.FromStringOrNil("c6cc16b0-03d5-4332-b1d5-0c0b68e29847"),
					},

					ChainedCallIDs: []uuid.UUID{},
					RecordingIDs:   []uuid.UUID{},
					Data:           map[call.DataType]string{},
					Dialroutes:     []rmroute.Route{},

					TMProgressing: DefaultTimeStamp,
					TMRinging:     DefaultTimeStamp,
					TMHangup:      DefaultTimeStamp,

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.calls {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().CallSet(ctx, gomock.Any())
				_ = h.CallCreate(ctx, c)
			}

			// delete
			for _, id := range tt.deleteCallIDs {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
				mockCache.EXPECT().CallSet(ctx, gomock.Any())
				_ = h.CallDelete(ctx, id)
			}

			filters := map[call.Field]any{
				call.FieldCustomerID: tt.customerID,
				call.FieldDeleted:    false,
			}

			res, err := h.CallGets(ctx, 10, utilhandler.TimeGetCurTime(), filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallSetBridgeID(t *testing.T) {
	type test struct {
		name string

		call *call.Call

		id              uuid.UUID
		bridgeID        string
		responseCurTime string

		expectRes *call.Call
	}

	tests := []test{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("853a2b18-5f8b-11ed-ba64-d35836e18de8"),
				},
			},

			uuid.FromStringOrNil("853a2b18-5f8b-11ed-ba64-d35836e18de8"),
			"c6d46f04-5f89-11ed-98b2-57f1fabc3cf4",
			"2020-04-18T03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("853a2b18-5f8b-11ed-ba64-d35836e18de8"),
				},

				BridgeID: "c6d46f04-5f89-11ed-98b2-57f1fabc3cf4",

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Data: map[call.DataType]string{},

				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallGet(ctx, tt.id.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetBridgeID(ctx, tt.id, tt.bridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallSetStatus(t *testing.T) {
	type test struct {
		name string

		call *call.Call

		id     uuid.UUID
		status call.Status

		responseCurtime string

		expectRes *call.Call
	}

	tests := []test{
		{
			"status terminating",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("93d7aea3-4a93-4c58-8bce-d956a0f73ad6"),
				},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},

			uuid.FromStringOrNil("93d7aea3-4a93-4c58-8bce-d956a0f73ad6"),
			call.StatusTerminating,

			"2020-04-18 03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("93d7aea3-4a93-4c58-8bce-d956a0f73ad6"),
				},

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusTerminating,
				Direction: call.DirectionIncoming,
				Data:      map[call.DataType]string{},

				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,

				TMProgressing: DefaultTimeStamp,
				TMRinging:     DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurtime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurtime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetStatus(ctx, tt.id, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallGetByChannelID(t *testing.T) {

	type test struct {
		name string
		call call.Call

		channelID string

		responseCurtime string
		expectCall      call.Call
	}

	tests := []test{
		{
			"test normal",
			call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5462dfbe-6bc8-11ed-b128-57813cbc586f"),
				},
				ChannelID: "54963ab2-6bc8-11ed-9acd-8325f591cc80",
				Type:      call.TypeFlow,

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},

			"54963ab2-6bc8-11ed-9acd-8325f591cc80",

			"2020-04-18T03:22:17.995000",
			call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5462dfbe-6bc8-11ed-b128-57813cbc586f"),
				},
				ChannelID: "54963ab2-6bc8-11ed-9acd-8325f591cc80",
				Type:      call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[call.DataType]string{},
				Dialroutes: []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"test normal has source address type sip",
			call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("79dc4e7e-6bc8-11ed-a082-2f57dba9cd50"),
				},
				ChannelID: "79b8a8b6-6bc8-11ed-8ace-af5dbf486a09",
				Type:      call.TypeFlow,

				Source: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},

			"79b8a8b6-6bc8-11ed-8ace-af5dbf486a09",

			"2020-04-18T03:22:17.995000",
			call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("79dc4e7e-6bc8-11ed-a082-2f57dba9cd50"),
				},
				ChannelID: "79b8a8b6-6bc8-11ed-8ace-af5dbf486a09",
				Type:      call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source: commonaddress.Address{
					Type: commonaddress.TypeSIP,
				},

				Status:     call.StatusRinging,
				Direction:  call.DirectionIncoming,
				Data:       map[call.DataType]string{},
				Dialroutes: []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurtime)
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(ctx, &tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CallGetByChannelID(ctx, tt.call.ChannelID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetHangup(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		id       uuid.UUID
		reason   call.HangupReason
		hangupBy call.HangupBy

		responseCurTime string

		expectCall call.Call
	}

	tests := []test{
		{
			"test normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0ffea974-6bc9-11ed-96aa-b35964df6757"),
				},
				ChannelID: "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:      call.TypeFlow,

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,

				TMCreate: "2020-04-18T03:22:17.995000",
			},

			uuid.FromStringOrNil("0ffea974-6bc9-11ed-96aa-b35964df6757"),
			call.HangupReasonNormal,
			call.HangupByLocal,

			"2020-04-18T03:22:18.995000",

			call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0ffea974-6bc9-11ed-96aa-b35964df6757"),
				},
				ChannelID: "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:      call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusHangup,
				Direction: call.DirectionIncoming,

				HangupReason: call.HangupReasonNormal,
				HangupBy:     call.HangupByLocal,
				Data:         map[call.DataType]string{},
				Dialroutes:   []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      "2020-04-18T03:22:18.995000",

				TMCreate: "2020-04-18T03:22:18.995000",
				TMUpdate: "2020-04-18T03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetHangup(context.Background(), tt.id, tt.reason, tt.hangupBy); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetFlowID(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		flowID uuid.UUID

		responseCurTime string

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
				},
			},

			uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
				},

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				FlowID: uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),

				Data:       map[call.DataType]string{},
				Dialroutes: []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetFlowID(ctx, tt.call.ID, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectCall, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetConfbridgeID(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		confbridgeID uuid.UUID

		responseCurTime string

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
				},
			},

			uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),
			"2020-04-18T03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
				},

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				ConfbridgeID: uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),

				Data:       map[call.DataType]string{},
				Dialroutes: []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallCreate(context.Background(), tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			if err := h.CallSetConfbridgeID(context.Background(), tt.call.ID, tt.confbridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
			res, err := h.CallGet(context.Background(), tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectCall, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetActionAndActionNextHold(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		action *fmaction.Action
		hold   bool

		responseCurTime string

		expectCall *call.Call
	}

	tests := []test{
		{
			"echo option duration",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
				},
				ChannelID: "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:      call.TypeFlow,
				FlowID:    uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},
			&fmaction.Action{
				ID:   uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
				Type: fmaction.TypeEcho,
				Option: map[string]any{
					"duration": 180,
				},
			},
			false,

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
				},
				ChannelID: "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
				Type:      call.TypeFlow,
				FlowID:    uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
					Type: fmaction.TypeEcho,
					Option: map[string]any{
						"duration": float64(180),
					},
				},
				ActionNextHold: false,
				Status:         call.StatusRinging,
				Direction:      call.DirectionIncoming,
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"echo option empty",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
				},
				ChannelID: "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
				Type:      call.TypeFlow,
				FlowID:    uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},
			&fmaction.Action{
				ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
				Type: fmaction.TypeEcho,
			},
			false,

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
				},
				ChannelID: "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
				Type:      call.TypeFlow,
				FlowID:    uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Action: fmaction.Action{
					ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
					Type: fmaction.TypeEcho,
				},
				ActionNextHold: false,
				Status:         call.StatusRinging,
				Direction:      call.DirectionIncoming,
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetActionAndActionNextHold(ctx, tt.call.ID, tt.action, tt.hold); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

		responseCurTime string

		expectCall *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14649d2c-24fc-11eb-bb0b-9bd6970f725f"),
				},
				ChannelID: "14daba5c-24fc-11eb-8f58-8b798baaf553",
				Type:      call.TypeFlow,
			},
			uuid.FromStringOrNil("4a6ce0aa-24fc-11eb-aec0-4b97b9a2422a"),

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14649d2c-24fc-11eb-bb0b-9bd6970f725f"),
				},
				ChannelID:      "14daba5c-24fc-11eb-8f58-8b798baaf553",
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},
				MasterCallID:   uuid.FromStringOrNil("4a6ce0aa-24fc-11eb-aec0-4b97b9a2422a"),

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"set nil",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("665db8f2-2501-11eb-86ce-f3a50eef6f26"),
				},
				Type: call.TypeFlow,
			},
			uuid.Nil,

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("665db8f2-2501-11eb-86ce-f3a50eef6f26"),
				},
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetMasterCallID(ctx, tt.call.ID, tt.masterCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetRecordingID(t *testing.T) {

	type test struct {
		name     string
		call     *call.Call
		reocrdID uuid.UUID

		responseCurTime string
		expectCall      *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("46ab9ad8-282b-11eb-82c3-6782faf5e030"),
				},
				ChannelID: "4e2fe520-282b-11eb-ad66-b777dce59261",
				Type:      call.TypeFlow,
				TMCreate:  "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("4e847572-282b-11eb-9c58-97622e4406e2"),

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("46ab9ad8-282b-11eb-82c3-6782faf5e030"),
				},
				ChannelID:      "4e2fe520-282b-11eb-ad66-b777dce59261",
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				RecordingID: uuid.FromStringOrNil("4e847572-282b-11eb-9c58-97622e4406e2"),

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"set empty",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7b3e197e-282b-11eb-956d-4feb054947db"),
				},
				Type:     call.TypeFlow,
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.Nil,

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7b3e197e-282b-11eb-956d-4feb054947db"),
				},
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetRecordingID(ctx, tt.call.ID, tt.reocrdID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetExternalMediaID(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		externalMediaID uuid.UUID

		responseCurTime string
		expectCall      *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("93eef900-96f3-11ed-9ce1-3f97da39b9d8"),
				},
			},

			uuid.FromStringOrNil("94202c28-96f3-11ed-9327-176b7622fb32"),

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("93eef900-96f3-11ed-9ce1-3f97da39b9d8"),
				},
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				ExternalMediaID: uuid.FromStringOrNil("94202c28-96f3-11ed-9327-176b7622fb32"),

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetExternalMediaID(ctx, tt.call.ID, tt.externalMediaID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallSetForRouteFailover(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		id          uuid.UUID
		channelID   string
		dialrouteID uuid.UUID

		responseCurTime string

		expectRes *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eff7e968-6035-11ed-b494-17ddee07f371"),
				},
				ChannelID: "54144112-6036-11ed-9492-5b86fbbf6672",
				Type:      call.TypeFlow,
			},

			uuid.FromStringOrNil("eff7e968-6035-11ed-b494-17ddee07f371"),
			"06372bc6-6036-11ed-bd92-7793e1da99bd",
			uuid.FromStringOrNil("11441a56-6036-11ed-9ac4-3b51fc15b1a1"),

			"2020-04-18T03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eff7e968-6035-11ed-b494-17ddee07f371"),
				},
				ChannelID:      "06372bc6-6036-11ed-bd92-7793e1da99bd",
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				DialrouteID:    uuid.FromStringOrNil("11441a56-6036-11ed-9ac4-3b51fc15b1a1"),
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetForRouteFailover(ctx, tt.id, tt.channelID, tt.dialrouteID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallSetActionNextHold(t *testing.T) {

	type test struct {
		name string

		call *call.Call
		hold bool

		responseCurTime string

		expectCall *call.Call
	}

	tests := []test{
		{
			"set true",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b347a946-6bab-11ed-845d-3fd878a04427"),
				},
			},
			true,

			"2020-04-18 03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b347a946-6bab-11ed-845d-3fd878a04427"),
				},
				ActionNextHold: true,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"set false",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10ecffc8-6bad-11ed-89dd-7bd54d2b1b6b"),
				},
			},
			false,

			"2020-04-18 03:22:17.995000",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("10ecffc8-6bad-11ed-89dd-7bd54d2b1b6b"),
				},
				ActionNextHold: false,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Data:           map[call.DataType]string{},
				Dialroutes:     []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetActionNextHold(ctx, tt.call.ID, tt.hold); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectCall, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CallDelete(t *testing.T) {

	type test struct {
		name string
		call *call.Call

		id uuid.UUID

		responseCurTime string

		expectRes call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407a8f3a-0fed-45b6-9587-a963e39c91ec"),
				},
			},

			uuid.FromStringOrNil("407a8f3a-0fed-45b6-9587-a963e39c91ec"),
			"2020-04-18T03:22:18.995000",

			call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407a8f3a-0fed-45b6-9587-a963e39c91ec"),
				},

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Source:      commonaddress.Address{},
				Destination: commonaddress.Address{},

				Data:       map[call.DataType]string{},
				Dialroutes: []rmroute.Route{},

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:18.995000",
				TMUpdate: "2020-04-18T03:22:18.995000",
				TMDelete: "2020-04-18T03:22:18.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallGet(ctx, tt.call.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallSetData(t *testing.T) {
	type test struct {
		name string
		call *call.Call

		id              uuid.UUID
		data            map[call.DataType]string
		responseCurTime string

		expectRes *call.Call
	}

	tests := []test{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d80157e-98bf-49e8-9827-06c744bfa81a"),
				},
			},

			uuid.FromStringOrNil("8d80157e-98bf-49e8-9827-06c744bfa81a"),
			map[call.DataType]string{
				call.DataTypeExecuteNextMasterOnHangup: "false",
			},
			"2020-04-18T03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d80157e-98bf-49e8-9827-06c744bfa81a"),
				},

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Data: map[call.DataType]string{
					call.DataTypeExecuteNextMasterOnHangup: "false",
				},

				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallGet(ctx, tt.id.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetData(ctx, tt.id, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallSetMuteDirection(t *testing.T) {
	type test struct {
		name string
		call *call.Call

		id              uuid.UUID
		mute            call.MuteDirection
		responseCurTime string

		expectRes *call.Call
	}

	tests := []test{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("21771598-d243-11ed-bbd2-b39e5d43e568"),
				},
			},

			uuid.FromStringOrNil("21771598-d243-11ed-bbd2-b39e5d43e568"),
			call.MuteDirectionBoth,
			"2020-04-18T03:22:17.995000",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("21771598-d243-11ed-bbd2-b39e5d43e568"),
				},
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Data:          map[call.DataType]string{},
				MuteDirection: call.MuteDirectionBoth,

				Dialroutes: []rmroute.Route{},

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: "2020-04-18T03:22:17.995000",
				TMDelete: DefaultTimeStamp,

				TMRinging:     DefaultTimeStamp,
				TMProgressing: DefaultTimeStamp,
				TMHangup:      DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallCreate(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().CallGet(ctx, tt.id.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			if err := h.CallSetMuteDirection(ctx, tt.id, tt.mute); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CallSet(ctx, gomock.Any())
			res, err := h.CallGet(ctx, tt.call.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
