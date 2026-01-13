package bridgehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"
)

func Test_ChannelKick(t *testing.T) {

	type test struct {
		name string

		id        string
		channelID string

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"af4eeca4-8824-4890-9d9c-9a4c27798b33",
			"01e8e225-7776-4d31-a74e-0056b2172fd9",

			&bridge.Bridge{
				ID: "af4eeca4-8824-4890-9d9c-9a4c27798b33",
			},
			&bridge.Bridge{
				ID: "af4eeca4-8824-4890-9d9c-9a4c27798b33",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id.Return(tt.responseBridge, nil)
			mockReq.EXPECT().AstBridgeRemoveChannel(ctx, tt.responseBridge.AsteriskID, tt.responseBridge.ID, tt.channelID.Return(nil)

			if err := h.ChannelKick(ctx, tt.id, tt.channelID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_ChannelJoin(t *testing.T) {

	type test struct {
		name string

		id         string
		channelID  string
		role       string
		absorbDTMF bool
		mute       bool

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"90b030be-b7c0-41e2-aabc-9a4c6f6a1a74",
			"9b80375b-8b5a-492a-977e-bfbe5e2a9d38",
			"",
			false,
			false,

			&bridge.Bridge{
				ID: "90b030be-b7c0-41e2-aabc-9a4c6f6a1a74",
			},
			&bridge.Bridge{
				ID: "90b030be-b7c0-41e2-aabc-9a4c6f6a1a74",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id.Return(tt.responseBridge, nil)
			mockReq.EXPECT().AstBridgeAddChannel(ctx, tt.responseBridge.AsteriskID, tt.responseBridge.ID, tt.channelID, tt.role, tt.absorbDTMF, tt.mute.Return(nil)

			if err := h.ChannelJoin(ctx, tt.id, tt.channelID, tt.role, tt.absorbDTMF, tt.mute); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
