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
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_MuteOn(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		direction call.MuteDirection

		responseCall      *call.Call
		responseMuteOnErr error
		responseMuteSetErr error
		expectError       bool
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("17cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
				},
				ChannelID: "9a4086ec-cef3-11ed-b377-ef35b455442f",
			},
			responseMuteOnErr:  nil,
			responseMuteSetErr: nil,
			expectError:        false,
		},
		{
			name: "channel mute on error",

			id:        uuid.FromStringOrNil("27cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
				},
				ChannelID: "aa4086ec-cef3-11ed-b377-ef35b455442f",
			},
			responseMuteOnErr: fmt.Errorf("channel error"),
			expectError:       true,
		},
		{
			name: "update mute direction error",

			id:        uuid.FromStringOrNil("37cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("37cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
				},
				ChannelID: "ba4086ec-cef3-11ed-b377-ef35b455442f",
			},
			responseMuteOnErr:  nil,
			responseMuteSetErr: fmt.Errorf("db error"),
			expectError:        true,
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
			mockChannel.EXPECT().MuteOn(ctx, tt.responseCall.ChannelID, channel.MuteDirection(tt.direction)).Return(tt.responseMuteOnErr)
			if tt.responseMuteOnErr == nil {
				mockDB.EXPECT().CallSetMuteDirection(ctx, tt.id, tt.direction).Return(tt.responseMuteSetErr)
				if tt.responseMuteSetErr == nil {
					mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
					mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)
				}
			}

			err := h.MuteOn(ctx, tt.id, tt.direction)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MuteOff(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		direction call.MuteDirection

		responseCall        *call.Call
		responseMuteOffErr  error
		responseMuteSetErr  error
		expectMuteDirection call.MuteDirection
		expectError         bool
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("183303a2-d13a-11ed-a800-9b0f57c3143f"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("183303a2-d13a-11ed-a800-9b0f57c3143f"),
				},
				ChannelID: "9a6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseMuteOffErr:  nil,
			responseMuteSetErr:  nil,
			expectMuteDirection: call.MuteDirectionNone,
			expectError:         false,
		},
		{
			name: "channel mute off error",

			id:        uuid.FromStringOrNil("283303a2-d13a-11ed-a800-9b0f57c3143f"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("283303a2-d13a-11ed-a800-9b0f57c3143f"),
				},
				ChannelID: "aa6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseMuteOffErr: fmt.Errorf("channel error"),
			expectError:        true,
		},
		{
			name: "update mute direction error",

			id:        uuid.FromStringOrNil("383303a2-d13a-11ed-a800-9b0f57c3143f"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("383303a2-d13a-11ed-a800-9b0f57c3143f"),
				},
				ChannelID: "ba6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseMuteOffErr:  nil,
			responseMuteSetErr:  fmt.Errorf("db error"),
			expectMuteDirection: call.MuteDirectionNone,
			expectError:         true,
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
			mockChannel.EXPECT().MuteOff(ctx, tt.responseCall.ChannelID, channel.MuteDirection(tt.direction)).Return(tt.responseMuteOffErr)
			if tt.responseMuteOffErr == nil {
				mockDB.EXPECT().CallSetMuteDirection(ctx, tt.id, tt.expectMuteDirection).Return(tt.responseMuteSetErr)
				if tt.responseMuteSetErr == nil {
					mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
					mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)
				}
			}

			err := h.MuteOff(ctx, tt.id, tt.direction)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
