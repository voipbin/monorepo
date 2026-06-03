package confbridgehandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_StartContextIncoming(t *testing.T) {

	joinErr := fmt.Errorf("bridge not found")

	tests := []struct {
		name string

		channel *channel.Channel

		responseConfbridge  *confbridge.Confbridge
		expectConfbridgeID  uuid.UUID
		responseChannelJoin error

		expectError error
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "d93fdf46-977c-11ec-8403-5b0f71484cde",
				StasisData: map[channel.StasisDataType]string{
					"call_id":       "a6a017f6-a3c7-11ed-9313-b7e5ce254097",
					"confbridge_id": "a6c4e9c8-a3c7-11ed-8961-7390b2c3f45c",
				},
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6c4e9c8-a3c7-11ed-8961-7390b2c3f45c"),
				},
				BridgeID: "a6e2e860-a3c7-11ed-9d27-8b42b68dfd08",
			},
			expectConfbridgeID:  uuid.FromStringOrNil("a6c4e9c8-a3c7-11ed-8961-7390b2c3f45c"),
			responseChannelJoin: nil,

			expectError: nil,
		},
		{
			// Regression test: before the fix, ChannelJoin failures were silently swallowed
			// because errors.Wrap(err, msg) used the outer `err` (nil) instead of errJoin.
			// errors.Wrap(nil, msg) returns nil, so the function incorrectly returned nil.
			name: "ChannelJoin error is propagated",

			channel: &channel.Channel{
				ID:         "asterisk-conference-test-channel-001",
				AsteriskID: "80:fa:5b:5e:da:81",
				StasisData: map[channel.StasisDataType]string{
					"call_id":       "b1000000-0000-0000-0000-000000000001",
					"confbridge_id": "b2000000-0000-0000-0000-000000000002",
				},
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b2000000-0000-0000-0000-000000000002"),
				},
				BridgeID: "b3000000-0000-0000-0000-000000000003",
			},
			expectConfbridgeID:  uuid.FromStringOrNil("b2000000-0000-0000-0000-000000000002"),
			responseChannelJoin: joinErr,

			expectError: joinErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.expectConfbridgeID).Return(tt.responseConfbridge, nil)
			mockBridge.EXPECT().ChannelJoin(ctx, tt.responseConfbridge.BridgeID, tt.channel.ID, "", false, false).Return(tt.responseChannelJoin)

			err := h.StartContextIncoming(ctx, tt.channel)

			if tt.expectError != nil {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
					return
				}
				if !errors.Is(err, tt.expectError) {
					t.Errorf("Wrong match. expect error wrapping %v, got: %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}
		})
	}
}


