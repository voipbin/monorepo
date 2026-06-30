package callhandler

import (
	"monorepo/bin-call-manager/pkg/testhelper"
	"context"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_ARIChannelStateChangeStatusProgressing(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call

		responseCall   *call.Call
		responseBridge *bridge.Bridge
		responsePeer   *channel.Channel
	}{
		{
			"normal answer",
			&channel.Channel{
				ID:       "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data:     map[string]interface{}{},
				State:    ari.ChannelStateUp,
				Type:     channel.TypeCall,
				TMAnswer: testhelper.TimePtr("2020-05-02T20:56:51.498Z"),
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4974832-4d3b-11ec-895b-0b7796863054"),
				},
				Status: call.StatusRinging,
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4974832-4d3b-11ec-895b-0b7796863054"),
				},
				Status: call.StatusProgressing,
			},
			nil,
			nil,
		},
		{
			"update answer from dialing",
			&channel.Channel{
				ID:       "849f1e92-4d40-11ec-b40a-739fbc078d18",
				Data:     map[string]interface{}{},
				State:    ari.ChannelStateUp,
				Type:     channel.TypeCall,
				TMAnswer: testhelper.TimePtr("2020-05-02T20:56:51.498Z"),
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("84e77160-4d40-11ec-aa31-8b1d57a189d0"),
				},
				Status: call.StatusDialing,
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("84e77160-4d40-11ec-aa31-8b1d57a189d0"),
				},
				Status: call.StatusProgressing,
			},
			nil,
			nil,
		},
		{
			// integration: outgoing channel Up with BridgeID set → answerCallBridgePeers fires
			// and answers the non-Up incoming peer in the call bridge
			"answer with bridge peer auto-answer",
			&channel.Channel{
				ID:       "call-out-channel-bridge-test",
				Data:     map[string]interface{}{},
				State:    ari.ChannelStateUp,
				BridgeID: "call-bridge-test-id",
				Type:     channel.TypeCall,
				TMAnswer: testhelper.TimePtr("2020-05-02T20:56:51.498Z"),
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc000000-0000-0000-0000-000000000001"),
				},
				Status: call.StatusDialing,
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cc000000-0000-0000-0000-000000000001"),
				},
				Status: call.StatusProgressing,
			},
			&bridge.Bridge{
				ID:            "call-bridge-test-id",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"call-out-channel-bridge-test", "call-in-channel-bridge-test"},
			},
			&channel.Channel{
				ID:    "call-in-channel-bridge-test",
				State: ari.ChannelStateRing,
			},
		},
	}

	for _, tt := range tests {
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

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatusProgressing(gomock.Any(), tt.call.ID)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseCall.CustomerID, call.EventTypeCallProgressing, tt.responseCall)
			if tt.call.Direction != call.DirectionIncoming {
				// rtp_debug is now read from call metadata; no CustomerV1CustomerGet expected.
				// ActionNext
				// consider the call was hungup already to make this test done quickly.
				mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(&call.Call{Status: call.StatusHangup}, nil)
			}

			// answerCallBridgePeers path: only when BridgeID is set
			if tt.channel.BridgeID != "" && tt.responseBridge != nil {
				mockBridge.EXPECT().Get(ctx, tt.channel.BridgeID).Return(tt.responseBridge, nil)
				if tt.responsePeer != nil {
					mockChannel.EXPECT().Get(ctx, tt.responsePeer.ID).Return(tt.responsePeer, nil)
					mockChannel.EXPECT().Answer(ctx, tt.responsePeer.ID).Return(nil)
				}
			}

			if err := h.ARIChannelStateChange(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_answerCallBridgePeers(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel

		responseBridge *bridge.Bridge
		bridgeGetErr   error
		responsePeer   *channel.Channel
		peerGetErr     error
		answerErr      error
		expectGetPeer  bool
		expectAnswer   bool
	}{
		{
			// normal case: peer channel in call bridge is not yet Up → Answer called
			name: "peer not up - answer called",
			channel: &channel.Channel{
				ID:       "call-out-channel-id",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-aaa",
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-aaa",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"call-out-channel-id", "call-in-channel-id"},
			},
			responsePeer: &channel.Channel{
				ID:    "call-in-channel-id",
				State: ari.ChannelStateRing,
			},
			expectGetPeer: true,
			expectAnswer:  true,
		},
		{
			// guard check: confbridge bridge → no answer called
			name: "confbridge bridge - no answer",
			channel: &channel.Channel{
				ID:       "conf-out-channel-id",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-bbb",
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-bbb",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ChannelIDs:    []string{"conf-out-channel-id", "conf-in-channel-id"},
			},
			expectGetPeer: false,
			expectAnswer:  false,
		},
		{
			// peer already Up → skip, no answer called
			name: "peer already up - skip",
			channel: &channel.Channel{
				ID:       "call-out-channel-id-2",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-ccc",
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-ccc",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"call-out-channel-id-2", "peer-already-up-id"},
			},
			responsePeer: &channel.Channel{
				ID:    "peer-already-up-id",
				State: ari.ChannelStateUp,
			},
			expectGetPeer: true,
			expectAnswer:  false,
		},
		{
			// single-channel bridge (only self) → loop no-ops, no Get/Answer called
			name: "single channel bridge - no-op",
			channel: &channel.Channel{
				ID:       "only-channel-id",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-ddd",
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-ddd",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"only-channel-id"},
			},
			expectGetPeer: false,
			expectAnswer:  false,
		},
		{
			// bridge Get returns error → early return, no peer interaction
			name: "bridge get error - no-op",
			channel: &channel.Channel{
				ID:       "call-out-channel-id-3",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-eee",
			},
			bridgeGetErr:  fmt.Errorf("bridge not found"),
			expectGetPeer: false,
			expectAnswer:  false,
		},
		{
			// peer channel Get returns error → continue loop, no Answer called
			name: "peer channel get error - continue",
			channel: &channel.Channel{
				ID:       "call-out-channel-id-4",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-fff",
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-fff",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"call-out-channel-id-4", "peer-error-id"},
			},
			peerGetErr:    fmt.Errorf("channel not found"),
			expectGetPeer: true,
			expectAnswer:  false,
		},
		{
			// Answer() returns error (e.g., groupcall race teardown) → loop continues, Warnf emitted
			name: "answer error - warn and continue",
			channel: &channel.Channel{
				ID:       "call-out-channel-id-5",
				State:    ari.ChannelStateUp,
				BridgeID: "bridge-ggg",
			},
			responseBridge: &bridge.Bridge{
				ID:            "bridge-ggg",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"call-out-channel-id-5", "peer-answer-fail-id"},
			},
			responsePeer: &channel.Channel{
				ID:    "peer-answer-fail-id",
				State: ari.ChannelStateRing,
			},
			answerErr:     fmt.Errorf("channel already hung up"),
			expectGetPeer: true,
			expectAnswer:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				bridgeHandler:  mockBridge,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			// bridge Get always called
			if tt.bridgeGetErr != nil {
				mockBridge.EXPECT().Get(ctx, tt.channel.BridgeID).Return(nil, tt.bridgeGetErr)
			} else {
				mockBridge.EXPECT().Get(ctx, tt.channel.BridgeID).Return(tt.responseBridge, nil)
			}

			if tt.expectGetPeer && tt.responsePeer != nil {
				if tt.peerGetErr != nil {
					mockChannel.EXPECT().Get(ctx, gomock.Any()).Return(nil, tt.peerGetErr)
				} else {
					mockChannel.EXPECT().Get(ctx, tt.responsePeer.ID).Return(tt.responsePeer, nil)
				}
			} else if tt.expectGetPeer && tt.peerGetErr != nil {
				// peer ID from bridge ChannelIDs
				for _, id := range tt.responseBridge.ChannelIDs {
					if id != tt.channel.ID {
						mockChannel.EXPECT().Get(ctx, id).Return(nil, tt.peerGetErr)
					}
				}
			}

			if tt.expectAnswer {
				mockChannel.EXPECT().Answer(ctx, tt.responsePeer.ID).Return(tt.answerErr)
			}

			h.answerCallBridgePeers(ctx, tt.channel)
		})
	}
}

func Test_ARIChannelStateChangeStatusRinging(t *testing.T) {

	tests := []struct {
		name          string
		channel       *channel.Channel
		responseCall1 *call.Call
		responseCall2 *call.Call
	}{
		{
			"normal",
			&channel.Channel{
				ID:        "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data:      map[string]interface{}{},
				State:     ari.ChannelStateRing,
				Type:      channel.TypeCall,
				TMRinging: testhelper.TimePtr("2020-05-02T20:56:51.498Z"),
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4974832-4d3b-11ec-895b-0b7796863054"),
				},
				Status: call.StatusDialing,
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4974832-4d3b-11ec-895b-0b7796863054"),
				},
				Status: call.StatusRinging,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(ctx, tt.channel.ID).Return(tt.responseCall1, nil)
			mockDB.EXPECT().CallSetStatusRinging(ctx, tt.responseCall1.ID)
			mockDB.EXPECT().CallGet(ctx, tt.responseCall1.ID).Return(tt.responseCall2, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseCall2.CustomerID, call.EventTypeCallRinging, tt.responseCall2)

			if err := h.ARIChannelStateChange(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeCall(t *testing.T) {

	tests := []struct {
		name string

		channel *channel.Channel

		responseCall *call.Call
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				ID:          "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data:        map[string]interface{}{},
				HangupCause: ari.ChannelCauseNormalClearing,
				Type:        channel.TypeCall,
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("67500948-df45-11ee-b0c6-1383284b63b0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			// call hangup
			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.responseCall, nil)
			mockBridge.EXPECT().Destroy(ctx, tt.responseCall.BridgeID).Return(nil)
			mockDB.EXPECT().CallSetHangup(ctx, tt.responseCall.ID, call.HangupReasonNormal, call.HangupByRemote).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseCall.CustomerID, call.EventTypeCallHangup, tt.responseCall)
			mockReq.EXPECT().FlowV1ActiveflowStop(ctx, tt.responseCall.ActiveflowID).Return(&fmactiveflow.Activeflow{}, nil)

			if err := h.ARIChannelDestroyed(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeConference(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			"conference normal destroy",
			&channel.Channel{
				ID:          "78ff0ed4-dd7b-11ea-9add-dbca62f7e8b9",
				Data:        map[string]interface{}{},
				Type:        channel.TypeConfbridge,
				HangupCause: ari.ChannelCauseNormalClearing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()
			if err := h.ARIChannelDestroyed(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ARIPlaybackFinished(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
		e       *ari.PlaybackFinished

		responseCall *call.Call
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				ID:   "1b8da938-e7dd-11ea-8e4a-1f2bd2b9f5b4",
				Data: map[string]interface{}{},
				Type: channel.TypeConfbridge,
			},
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66795a5a-e7dd-11ea-b2df-0757b438501c"),
				},
				Action: action.Action{
					ID: uuid.FromStringOrNil("77a82874-e7dd-11ea-9647-27054cd71830"),
				},
				FlowID:       uuid.FromStringOrNil("32c36bf4-156f-11ec-af17-87eb4aca917b"),
				ActiveflowID: uuid.FromStringOrNil("244d4566-a7bb-11ec-92eb-fbdbdda3d486"),
			},
			e: &ari.PlaybackFinished{
				Playback: ari.Playback{
					ID: "call:77a82874-e7dd-11ea-9647-27054cd71830",
				},
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("66795a5a-e7dd-11ea-b2df-0757b438501c"),
				},
				Action:       action.Action{},
				FlowID:       uuid.FromStringOrNil("32c36bf4-156f-11ec-af17-87eb4aca917b"),
				ActiveflowID: uuid.FromStringOrNil("244d4566-a7bb-11ec-92eb-fbdbdda3d486"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)

			// action next part.
			mockDB.EXPECT().CallSetActionNextHold(ctx, tt.call.ID, true).Return(fmt.Errorf(""))
			mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(&call.Call{Status: call.StatusHangup}, nil)

			if errFin := h.ARIPlaybackFinished(ctx, tt.channel, tt.e); errFin != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errFin)
			}
		})
	}
}
