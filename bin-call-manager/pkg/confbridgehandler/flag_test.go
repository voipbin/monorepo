package confbridgehandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_FlagExist(t *testing.T) {

	tests := []struct {
		name string

		flags []confbridge.Flag
		flag  confbridge.Flag

		expectRes bool
	}{
		{
			name: "normal",

			flags: []confbridge.Flag{
				confbridge.FlagNoAutoLeave,
			},
			flag: confbridge.FlagNoAutoLeave,

			expectRes: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				bridgeHandler: mockBridge,
			}
			ctx := context.Background()

			res := h.flagExist(ctx, tt.flags, tt.flag)

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlagAdd(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		flag confbridge.Flag

		responseConfbridge *confbridge.Confbridge

		expectFlags []confbridge.Flag
		expectRes   *confbridge.Confbridge
	}{
		{
			name: "normal",

			id:   uuid.FromStringOrNil("55de6448-d7b2-11ed-a2aa-0f14677b59c0"),
			flag: confbridge.FlagNoAutoLeave,

			responseConfbridge: &confbridge.Confbridge{
				ID:    uuid.FromStringOrNil("55de6448-d7b2-11ed-a2aa-0f14677b59c0"),
				Flags: []confbridge.Flag{},
			},

			expectFlags: []confbridge.Flag{
				confbridge.FlagNoAutoLeave,
			},
			expectRes: &confbridge.Confbridge{
				ID: uuid.FromStringOrNil("55de6448-d7b2-11ed-a2aa-0f14677b59c0"),
				Flags: []confbridge.Flag{
					confbridge.FlagNoAutoLeave,
				},
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				bridgeHandler: mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockDB.EXPECT().ConfbridgeSetFlags(ctx, tt.id, tt.expectFlags).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)

			res, err := h.FlagAdd(ctx, tt.id, tt.flag)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.Flags = tt.expectFlags
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlagRemove(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		flag confbridge.Flag

		responseConfbridge *confbridge.Confbridge

		expectFlags []confbridge.Flag
		expectRes   *confbridge.Confbridge
	}{
		{
			name: "normal",

			id:   uuid.FromStringOrNil("c71bc776-d7b3-11ed-9e34-e336e1512377"),
			flag: confbridge.FlagNoAutoLeave,

			responseConfbridge: &confbridge.Confbridge{
				ID: uuid.FromStringOrNil("c71bc776-d7b3-11ed-9e34-e336e1512377"),
				Flags: []confbridge.Flag{
					confbridge.FlagNoAutoLeave,
				},
			},

			expectFlags: []confbridge.Flag{},
			expectRes: &confbridge.Confbridge{
				ID:    uuid.FromStringOrNil("c71bc776-d7b3-11ed-9e34-e336e1512377"),
				Flags: []confbridge.Flag{},
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				bridgeHandler: mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			mockDB.EXPECT().ConfbridgeSetFlags(ctx, tt.id, tt.expectFlags).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)

			res, err := h.FlagRemove(ctx, tt.id, tt.flag)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.Flags = tt.expectFlags
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
