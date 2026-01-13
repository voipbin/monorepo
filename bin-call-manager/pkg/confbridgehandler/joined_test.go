package confbridgehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Joined_type_connect(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge

		responseConfbridge *confbridge.Confbridge
		responseCall       *call.Call

		expectCallID       uuid.UUID
		expectConfbridgeID uuid.UUID
		expectEvent        *confbridge.EventConfbridgeJoined
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "4268f036-38d0-11ec-a912-ebca1cd51965",
				StasisData: map[channel.StasisDataType]string{
					"confbridge_id": "eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2",
					"call_id":       "ebb3c432-38cf-11ec-ad96-fb9640d4c6ee",
				},
			},
			bridge: &bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
				},
				BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				Type:     confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"4268f036-38d0-11ec-a912-ebca1cd51965": uuid.FromStringOrNil("ebb3c432-38cf-11ec-ad96-fb9640d4c6ee"),
				},
			},
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ebb3c432-38cf-11ec-ad96-fb9640d4c6ee"),
				},
			},

			expectCallID:       uuid.FromStringOrNil("ebb3c432-38cf-11ec-ad96-fb9640d4c6ee"),
			expectConfbridgeID: uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
			expectEvent: &confbridge.EventConfbridgeJoined{
				Confbridge: confbridge.Confbridge{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("eb2e51b2-38cf-11ec-9b34-5ff390dc1ef2"),
					},
					BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
					Type:     confbridge.TypeConnect,
					ChannelCallIDs: map[string]uuid.UUID{
						"4268f036-38d0-11ec-a912-ebca1cd51965": uuid.FromStringOrNil("ebb3c432-38cf-11ec-ad96-fb9640d4c6ee"),
					},
				},
				JoinedCallID: uuid.FromStringOrNil("ebb3c432-38cf-11ec-ad96-fb9640d4c6ee"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := confbridgeHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeAddChannelCallID(ctx, tt.expectConfbridgeID, tt.channel.ID, tt.expectCallID.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.expectConfbridgeID.Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, tt.expectEvent)

			mockReq.EXPECT().CallV1CallUpdateConfbridgeID(ctx, tt.expectCallID, tt.expectConfbridgeID.Return(tt.responseCall, nil)

			mockChannel.EXPECT().Ring(ctx, tt.channel.ID.Return(nil)

			if err := h.Joined(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Joined_type_conference(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge

		responseConfbridge *confbridge.Confbridge
		responseCall       *call.Call

		expectCallID       uuid.UUID
		expectConfbridgeID uuid.UUID
		expectEvent        *confbridge.EventConfbridgeJoined
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				AsteriskID: "00:11:22:33:44:55",
				ID:         "27faec50-a3bd-11ed-b330-23d982f79917",
				StasisData: map[channel.StasisDataType]string{
					"confbridge_id": "282882b4-a3bd-11ed-9f93-3b874a128ea2",
					"call_id":       "28582dac-a3bd-11ed-9f5e-93bf1afcbcf6",
				},
			},
			bridge: &bridge.Bridge{
				AsteriskID: "00:11:22:33:44:66",
				ID:         "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				TMDelete:   dbhandler.DefaultTimeStamp,
			},

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("282882b4-a3bd-11ed-9f93-3b874a128ea2"),
				},
				BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
				Type:     confbridge.TypeConference,
			},
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("28582dac-a3bd-11ed-9f5e-93bf1afcbcf6"),
				},
			},

			expectCallID:       uuid.FromStringOrNil("28582dac-a3bd-11ed-9f5e-93bf1afcbcf6"),
			expectConfbridgeID: uuid.FromStringOrNil("282882b4-a3bd-11ed-9f93-3b874a128ea2"),
			expectEvent: &confbridge.EventConfbridgeJoined{
				Confbridge: confbridge.Confbridge{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("282882b4-a3bd-11ed-9f93-3b874a128ea2"),
					},
					BridgeID: "eb6d4516-38cf-11ec-9414-eb20d908d9a1",
					Type:     confbridge.TypeConference,
				},
				JoinedCallID: uuid.FromStringOrNil("28582dac-a3bd-11ed-9f5e-93bf1afcbcf6"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := confbridgeHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeAddChannelCallID(ctx, tt.expectConfbridgeID, tt.channel.ID, tt.expectCallID.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.expectConfbridgeID.Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, tt.expectEvent)

			mockReq.EXPECT().CallV1CallUpdateConfbridgeID(ctx, tt.expectCallID, tt.expectConfbridgeID.Return(tt.responseCall, nil)

			mockChannel.EXPECT().Answer(ctx, tt.channel.ID.Return(nil)

			if err := h.Joined(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_joinedTypeConnect(t *testing.T) {

	tests := []struct {
		name string

		channelID  string
		call       *call.Call
		confbridge *confbridge.Confbridge

		responseCalls []*call.Call

		expectFlagRing bool
	}{
		{
			name: "first channel",

			channelID: "afaa2896-a3bd-11ed-8195-eb6e150c260e",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("afd0acf0-a3bd-11ed-921e-8bf69d65d2c1"),
				},
			},
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("aff70e90-a3bd-11ed-8099-7b4446aef7ad"),
				},
				ChannelCallIDs: map[string]uuid.UUID{
					"9a6c08b2-ae91-11ed-b460-a377b7040b7c": uuid.FromStringOrNil("9a99bbae-ae91-11ed-971f-9b7ae6d5063d"),
				},
			},

			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("afd0acf0-a3bd-11ed-921e-8bf69d65d2c1"),
					},
				},
			},
		},
		{
			name: "not a first channel and outoing call is ringing",

			channelID: "070da3b0-a3be-11ed-aec8-27faeea66ece",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("07360832-a3be-11ed-80e9-6f824b36d382"),
				},
				Status: call.StatusRinging,
			},
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("07666a04-a3be-11ed-8367-bf3360333c86"),
				},
				Type: confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"070da3b0-a3be-11ed-aec8-27faeea66ece": uuid.FromStringOrNil("07360832-a3be-11ed-80e9-6f824b36d382"),
					"07980c12-a3be-11ed-b5cd-affd092220f9": uuid.FromStringOrNil("07cd17cc-a3be-11ed-b200-a75abbfb90ca"),
				},
			},

			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("07360832-a3be-11ed-80e9-6f824b36d382"),
					},
					Status:    call.StatusRinging,
					Direction: call.DirectionIncoming,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("07cd17cc-a3be-11ed-b200-a75abbfb90ca"),
					},
					Status:    call.StatusRinging,
					Direction: call.DirectionOutgoing,
				},
			},

			expectFlagRing: true,
		},
		{
			name: "not a first channel and outgoing call is progressing",

			channelID: "166454ca-a3bf-11ed-bb15-77d1ba7adeda",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("168b8482-a3bf-11ed-8739-db2e4a76beb9"),
				},
				Status: call.StatusRinging,
			},
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("07666a04-a3be-11ed-8367-bf3360333c86"),
				},
				Type: confbridge.TypeConnect,
				ChannelCallIDs: map[string]uuid.UUID{
					"16b1f518-a3bf-11ed-a015-ebe29a318ec0": uuid.FromStringOrNil("16dc7a40-a3bf-11ed-b0f3-1b2254e87579"),
					"166454ca-a3bf-11ed-bb15-77d1ba7adeda": uuid.FromStringOrNil("168b8482-a3bf-11ed-8739-db2e4a76beb9"),
				},
			},

			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("16dc7a40-a3bf-11ed-b0f3-1b2254e87579"),
					},
					Status:    call.StatusProgressing,
					Direction: call.DirectionOutgoing,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("168b8482-a3bf-11ed-8739-db2e4a76beb9"),
					},
					Status: call.StatusRinging,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := confbridgeHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			if len(tt.confbridge.ChannelCallIDs) == 1 {
				mockChannel.EXPECT().Ring(ctx, tt.channelID.Return(nil)
			} else {
				i := 0
				for _, callID := range tt.confbridge.ChannelCallIDs {
					mockReq.EXPECT().CallV1CallGet(ctx, callID.Return(tt.responseCalls[i], nil).AnyTimes()
					i++
				}

				if tt.expectFlagRing {
					mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(tt.confbridge, nil)
					for channelID := range tt.confbridge.ChannelCallIDs {
						mockChannel.EXPECT().Ring(ctx, channelID.Return(nil)
					}
				} else {
					mockDB.EXPECT().ConfbridgeGet(ctx, tt.confbridge.ID.Return(tt.confbridge, nil)
					for channelID := range tt.confbridge.ChannelCallIDs {
						mockChannel.EXPECT().Answer(ctx, channelID.Return(nil)
					}
				}
			}

			if errConnect := h.joinedTypeConnect(ctx, tt.channelID, tt.call, tt.confbridge); errConnect != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errConnect)
			}
		})
	}
}

func Test_joinedTypeConference(t *testing.T) {

	tests := []struct {
		name string

		channelID  string
		call       *call.Call
		confbridge *confbridge.Confbridge
	}{
		{
			name: "normal",

			channelID: "7b98f864-a3bf-11ed-8777-df84c037fd2b",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7bc0581e-a3bf-11ed-9781-07084ef6a76b"),
				},
			},
			confbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7bea77e8-a3bf-11ed-b44a-0f9d205baa32"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := confbridgeHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				cache:          mockCache,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().Answer(ctx, tt.channelID.Return(nil)

			if err := h.joinedTypeConference(ctx, tt.channelID, tt.call, tt.confbridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
