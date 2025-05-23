package confbridgehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Ring(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge *confbridge.Confbridge
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("be57968c-a3c0-11ed-bf64-631499c3b3cc"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be57968c-a3c0-11ed-bf64-631499c3b3cc"),
				},
				Type: confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"be8867f8-a3c0-11ed-b2ba-cf8f720b21c6": uuid.FromStringOrNil("beb15a78-a3c0-11ed-9be6-2f00f66bc267"),
					"bedc8522-a3c0-11ed-88bd-37a1d79fca0d": uuid.FromStringOrNil("bf0a64d8-a3c0-11ed-bb12-d3f4f0595174"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			for channelID := range tt.responseConfbridge.ChannelCallIDs {
				mockChannel.EXPECT().Ring(ctx, channelID).Return(nil)
			}

			if err := h.Ring(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Answer(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge *confbridge.Confbridge
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("1c26b522-a3c1-11ed-95bd-d37a121dcbc6"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1c26b522-a3c1-11ed-95bd-d37a121dcbc6"),
				},
				Type: confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"1c50f3a0-a3c1-11ed-9c58-4374f80d8554": uuid.FromStringOrNil("1c764bbe-a3c1-11ed-ab22-e316a3f2bcc1"),
					"1c95aa68-a3c1-11ed-81b8-4b91b947e3a0": uuid.FromStringOrNil("1cbdd22c-a3c1-11ed-8db9-fb3b7bf3c7ec"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(tt.responseConfbridge, nil)
			for channelID := range tt.responseConfbridge.ChannelCallIDs {
				mockChannel.EXPECT().Answer(ctx, channelID).Return(nil)
			}

			if err := h.Answer(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
