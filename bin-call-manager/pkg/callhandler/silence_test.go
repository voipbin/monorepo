package callhandler

import (
	"context"
	"fmt"
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

func Test_SilenceOn(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall       *call.Call
		responseSilenceOnErr error
		expectError        bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("4f52b58a-d13a-11ed-ba73-0b9ff66d000f"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f52b58a-d13a-11ed-ba73-0b9ff66d000f"),
				},
				ChannelID: "9a4086ec-cef3-11ed-b377-ef35b455442f",
			},
			responseSilenceOnErr: nil,
			expectError:          false,
		},
		{
			name: "channel silence on error",

			id: uuid.FromStringOrNil("5f52b58a-d13a-11ed-ba73-0b9ff66d000f"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f52b58a-d13a-11ed-ba73-0b9ff66d000f"),
				},
				ChannelID: "aa4086ec-cef3-11ed-b377-ef35b455442f",
			},
			responseSilenceOnErr: fmt.Errorf("channel silence on error"),
			expectError:          true,
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

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().SilenceOn(ctx, tt.responseCall.ChannelID).Return(tt.responseSilenceOnErr)

			err := h.SilenceOn(ctx, tt.id)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_SilenceOff(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall        *call.Call
		responseSilenceOffErr error
		expectError         bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("4f8b6006-d13a-11ed-9159-ff36007e7014"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f8b6006-d13a-11ed-9159-ff36007e7014"),
				},
				ChannelID: "9a6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseSilenceOffErr: nil,
			expectError:           false,
		},
		{
			name: "channel silence off error",

			id: uuid.FromStringOrNil("5f8b6006-d13a-11ed-9159-ff36007e7014"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f8b6006-d13a-11ed-9159-ff36007e7014"),
				},
				ChannelID: "aa6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseSilenceOffErr: fmt.Errorf("channel silence off error"),
			expectError:           true,
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

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().SilenceOff(ctx, tt.responseCall.ChannelID).Return(tt.responseSilenceOffErr)

			err := h.SilenceOff(ctx, tt.id)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
