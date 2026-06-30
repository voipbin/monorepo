package confbridgehandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_ARIStasisStartTypeConference(t *testing.T) {

	tests := []struct {
		name       string
		channel    *channel.Channel
		confbridge *confbridge.Confbridge
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeContext:      string(channel.ContextConfIncoming),
					channel.StasisDataTypeConfbridgeID: "69e97312-3748-11ec-a94b-2357c957d67e",
					channel.StasisDataTypeCallID:       "22df7716-34f3-11ec-a0d1-1faed65f6fd4",
				},
			},
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("69e97312-3748-11ec-a94b-2357c957d67e"),
				},
				Type:           confbridge.TypeConference,
				BridgeID:       "80c2e1ae-3748-11ec-b52c-c7e704ec1140",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
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

			mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID).Return(tt.confbridge, nil)
			mockBridge.EXPECT().ChannelJoin(ctx, tt.confbridge.BridgeID, tt.channel.ID, "", false, false).Return(nil)

			err := h.ARIStasisStart(ctx, tt.channel)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ARIStasisStartTypeConferenceError(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			"conference outgoing",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeContext: string(channel.ContextConfOutgoing),
				},
			},
		},
		{
			"no context",
			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "4961579e-169c-11ec-ad78-c36f42ca4c10",
				StasisData:        map[channel.StasisDataType]string{},
			},
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

			h := &confbridgeHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				cache:         mockCache,
			}

			ctx := context.Background()

			if err := h.ARIStasisStart(ctx, tt.channel); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_ARIChannelStateChangeTypeJoin(t *testing.T) {
	tests := []struct {
		name         string
		channel      *channel.Channel
		bridgeGetErr error
		responseBridge *bridge.Bridge
		responsePeer   *channel.Channel
		peerGetErr     error
		answerErr      error
		expectGetBridge bool
		expectGetPeer   bool
		expectAnswer    bool
	}{
		{
			// non-TypeJoin → early return
			name: "non-TypeJoin - no-op",
			channel: &channel.Channel{
				ID:    "call-in-channel-id",
				Type:  channel.TypeCall,
				State: ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-aaa",
				},
			},
			expectGetBridge: false,
		},
		{
			// TypeJoin but not Up → early return
			name: "TypeJoin not Up - no-op",
			channel: &channel.Channel{
				ID:    "conf-join-channel-id",
				Type:  channel.TypeJoin,
				State: ari.ChannelStateRing,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-bbb",
				},
			},
			expectGetBridge: false,
		},
		{
			// TypeJoin + Up but empty BridgeID and StasisData → early return
			name: "TypeJoin Up no bridgeID - no-op",
			channel: &channel.Channel{
				ID:         "conf-join-channel-id-2",
				Type:       channel.TypeJoin,
				State:      ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{},
			},
			expectGetBridge: false,
		},
		{
			// TypeJoin + Up + confbridge bridge → ReferenceTypeCall guard blocks
			name: "TypeJoin Up confbridge bridge - no-op",
			channel: &channel.Channel{
				ID:    "conf-join-channel-id-3",
				Type:  channel.TypeJoin,
				State: ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-ccc",
				},
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-ccc",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ChannelIDs:    []string{"conf-join-channel-id-3", "conf-in-id"},
			},
			expectGetBridge: true,
			expectGetPeer:   false,
		},
		{
			// happy path: TypeJoin + Up + call bridge + non-Up peer → Answer called
			name: "TypeJoin Up call bridge peer not up - answer called",
			channel: &channel.Channel{
				ID:    "conf-join-channel-id-4",
				Type:  channel.TypeJoin,
				State: ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-ddd",
				},
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-ddd",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"conf-join-channel-id-4", "call-in-id"},
			},
			responsePeer: &channel.Channel{
				ID:    "call-in-id",
				State: ari.ChannelStateRing,
			},
			expectGetBridge: true,
			expectGetPeer:   true,
			expectAnswer:    true,
		},
		{
			// peer already Up → skip
			name: "TypeJoin Up call bridge peer already up - skip",
			channel: &channel.Channel{
				ID:    "conf-join-channel-id-5",
				Type:  channel.TypeJoin,
				State: ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-eee",
				},
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-eee",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"conf-join-channel-id-5", "already-up-id"},
			},
			responsePeer: &channel.Channel{
				ID:    "already-up-id",
				State: ari.ChannelStateUp,
			},
			expectGetBridge: true,
			expectGetPeer:   true,
			expectAnswer:    false,
		},
		{
			// bridge Get fails → early return
			name: "bridge get error - no-op",
			channel: &channel.Channel{
				ID:    "conf-join-channel-id-6",
				Type:  channel.TypeJoin,
				State: ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-fff",
				},
			},
			bridgeGetErr:    fmt.Errorf("bridge not found"),
			expectGetBridge: true,
			expectGetPeer:   false,
		},
		{
			// peer Get fails → loop continues, no Answer
			name: "peer get error - continue",
			channel: &channel.Channel{
				ID:    "conf-join-channel-id-7",
				Type:  channel.TypeJoin,
				State: ari.ChannelStateUp,
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeBridgeID: "bridge-ggg",
				},
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-ggg",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"conf-join-channel-id-7", "peer-error-id"},
			},
			peerGetErr:      fmt.Errorf("channel not found"),
			expectGetBridge: true,
			expectGetPeer:   true,
			expectAnswer:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &confbridgeHandler{
				bridgeHandler:  mockBridge,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			if tt.expectGetBridge {
				if tt.bridgeGetErr != nil {
					mockBridge.EXPECT().Get(ctx, gomock.Any()).Return(nil, tt.bridgeGetErr)
				} else {
					mockBridge.EXPECT().Get(ctx, gomock.Any()).Return(tt.responseBridge, nil)
				}
			}

			if tt.expectGetPeer {
				if tt.peerGetErr != nil {
					mockChannel.EXPECT().Get(ctx, gomock.Any()).Return(nil, tt.peerGetErr)
				} else {
					mockChannel.EXPECT().Get(ctx, tt.responsePeer.ID).Return(tt.responsePeer, nil)
				}
			}

			if tt.expectAnswer {
				mockChannel.EXPECT().Answer(ctx, tt.responsePeer.ID).Return(tt.answerErr)
			}

			h.ARIChannelStateChange(ctx, tt.channel)
		})
	}
}

func Test_ARIChannelLeftBridge(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge

		responseConfbridge *confbridge.Confbridge
		expectConfbridgeID uuid.UUID
		expectCallID       uuid.UUID
	}{
		{
			name: "confbridge left",
			channel: &channel.Channel{
				ID:         "e03dc034-9566-11ea-ad83-1f7a1993587b",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{
					"confbridge_id": "e9051ac8-9566-11ea-bde6-331b8236a4c2",
					"call_id":       "ef83edb2-3bf9-11ec-bc7d-1f524326656b",
				},
				Type: channel.TypeConfbridge,
			},
			bridge: &bridge.Bridge{
				ID:            "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
				ReferenceID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				ReferenceType: bridge.ReferenceTypeConfbridge,
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
				},
				Type: confbridge.TypeConference,
			},
			expectConfbridgeID: uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
			expectCallID:       uuid.FromStringOrNil("ef83edb2-3bf9-11ec-bc7d-1f524326656b"),
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

			h := &confbridgeHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
				cache:         mockCache,

				channelHandler: mockChannel,
			}
			ctx := context.Background()

			// Leaved
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(tt.channel, nil)
			mockDB.EXPECT().ConfbridgeRemoveChannelCallID(ctx, tt.expectConfbridgeID, tt.channel.ID)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.expectConfbridgeID).Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeLeaved, gomock.Any())

			mockReq.EXPECT().CallV1CallUpdateConfbridgeID(ctx, tt.expectCallID, uuid.Nil).Return(&call.Call{}, nil)

			err := h.ARIChannelLeftBridge(ctx, tt.channel, tt.bridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}
