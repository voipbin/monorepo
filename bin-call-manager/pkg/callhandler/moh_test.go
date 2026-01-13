package callhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_MOHOn(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall *call.Call
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("d011cf18-d139-11ed-92f8-d75f1d7e1914"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d011cf18-d139-11ed-92f8-d75f1d7e1914"),
				},
				ChannelID: "9a4086ec-cef3-11ed-b377-ef35b455442f",
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
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id.Return(tt.responseCall, nil)
			mockChannel.EXPECT().MOHOn(ctx, tt.responseCall.ChannelID.Return(nil)

			if err := h.MOHOn(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MOHOff(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall *call.Call
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("d0609bfc-d139-11ed-b643-ff2b2ad0a3eb"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0609bfc-d139-11ed-b643-ff2b2ad0a3eb"),
				},
				ChannelID: "9a6e4122-cef3-11ed-b195-5b72e7449d60",
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
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id.Return(tt.responseCall, nil)
			mockChannel.EXPECT().MOHOff(ctx, tt.responseCall.ChannelID.Return(nil)

			if err := h.MOHOff(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
