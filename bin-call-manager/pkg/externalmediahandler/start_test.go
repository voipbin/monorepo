package externalmediahandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/playback"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Start_startReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		id              uuid.UUID
		referenceType   externalmedia.ReferenceType
		referenceID     uuid.UUID
		externalHost    string
		encapsulation   externalmedia.Encapsulation
		transport       externalmedia.Transport
		connectionType  string
		format          string
		directionListen externalmedia.Direction
		directionSpeak  externalmedia.Direction

		responseCall          *call.Call
		responseChannel       *channel.Channel
		responseUUIDBridgeID  uuid.UUID
		responseBridge        *bridge.Bridge
		responseUUIDSnoopID   uuid.UUID
		responseUUIDChannelID uuid.UUID

		expectBridgeArgs    string
		expectChannelData   string
		expectPlaybackID    string
		expectExternalMedia *externalmedia.ExternalMedia
	}{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("78473c24-b331-11ef-aa9c-e7c52f9d3f7b"),
			referenceType:   externalmedia.ReferenceTypeCall,
			referenceID:     uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
			externalHost:    "example.com",
			encapsulation:   externalmedia.EncapsulationRTP,
			transport:       "udp",
			connectionType:  "client",
			format:          "ulaw",
			directionListen: externalmedia.DirectionBoth,
			directionSpeak:  externalmedia.DirectionBoth,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				},
				// AsteriskID: "42:01:0a:a4:00:05",
				ChannelID: "8066017c-02fb-11ec-ba6c-c320820accf1",
				BridgeID:  "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba",
			},
			responseChannel: &channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "8066017c-02fb-11ec-ba6c-c320820accf1",
			},
			responseUUIDBridgeID: uuid.FromStringOrNil("9b6c7a78-96e3-11ed-904b-9baa2c0183fd"),
			responseBridge: &bridge.Bridge{
				ID: "9b6c7a78-96e3-11ed-904b-9baa2c0183fd",
			},
			responseUUIDSnoopID:   uuid.FromStringOrNil("80981342-96e3-11ed-bc85-830940cba8ea"),
			responseUUIDChannelID: uuid.FromStringOrNil("488feb00-96e3-11ed-8ae7-1fe9bc7a995f"),

			expectBridgeArgs:  "reference_type=call-snoop,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57",
			expectChannelData: "context_type=call,context=call-externalmedia,bridge_id=9b6c7a78-96e3-11ed-904b-9baa2c0183fd,reference_type=call,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57,external_media_id=78473c24-b331-11ef-aa9c-e7c52f9d3f7b",
			expectPlaybackID:  playback.IDPrefixExternalMedia + "78473c24-b331-11ef-aa9c-e7c52f9d3f7b",
			expectExternalMedia: &externalmedia.ExternalMedia{
				ID:              uuid.FromStringOrNil("78473c24-b331-11ef-aa9c-e7c52f9d3f7b"),
				AsteriskID:      "42:01:0a:a4:00:05",
				ChannelID:       "488feb00-96e3-11ed-8ae7-1fe9bc7a995f",
				ReferenceType:   externalmedia.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				LocalIP:         "",
				LocalPort:       0,
				ExternalHost:    "example.com",
				Encapsulation:   "rtp",
				Transport:       "udp",
				ConnectionType:  "client",
				Format:          "ulaw",
				DirectionListen: "both",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &externalMediaHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.responseCall.ID.Return(tt.responseCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID.Return(tt.responseChannel, nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDBridgeID)
			mockBridge.EXPECT().Start(ctx, tt.responseChannel.AsteriskID, tt.responseUUIDBridgeID.String(), tt.expectBridgeArgs, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}.Return(tt.responseBridge, nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDSnoopID)
			mockChannel.EXPECT().StartSnoop(ctx, tt.responseCall.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirection(tt.directionListen), channel.SnoopDirection(tt.directionSpeak).Return(&channel.Channel{}, nil)

			mockBridge.EXPECT().Play(ctx, tt.responseCall.BridgeID, tt.expectPlaybackID, []string{defaultSilencePlaybackMedia}, "", 0, 0.Return(nil)

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseChannel.AsteriskID, gomock.Any(), tt.externalHost, string(tt.encapsulation), string(tt.transport), tt.connectionType, tt.format, string(tt.directionListen), tt.expectChannelData, gomock.Any().Return(&channel.Channel{}, nil)

			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia.Return(nil)

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id.Return(tt.expectExternalMedia, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any().Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id.Return(tt.expectExternalMedia, nil)

			res, err := h.Start(
				ctx,
				tt.id,
				tt.referenceType,
				tt.referenceID,
				tt.externalHost,
				tt.encapsulation,
				tt.transport,
				tt.connectionType,
				tt.format,
				tt.directionListen,
				tt.directionSpeak,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectExternalMedia, res) {
				t.Errorf("Wrong match.\nexpect: %vgot: %v", tt.expectExternalMedia, res)
			}
		})
	}
}

func Test_Start_reference_type_confbridge(t *testing.T) {
	tests := []struct {
		name string

		id              uuid.UUID
		referenceType   externalmedia.ReferenceType
		referenceID     uuid.UUID
		externalHost    string
		encapsulation   externalmedia.Encapsulation
		transport       externalmedia.Transport
		connectionType  string
		format          string
		directionListen externalmedia.Direction
		directionSpeak  externalmedia.Direction

		responseConfbridge    *confbridge.Confbridge
		responseBridge        *bridge.Bridge
		responseUUIDChannelID uuid.UUID

		expectExternalHost  string
		expectChannelData   string
		expectExternalMedia *externalmedia.ExternalMedia
	}{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("79076e90-b331-11ef-bc31-33cb17f32724"),
			referenceType:   externalmedia.ReferenceTypeConfbridge,
			referenceID:     uuid.FromStringOrNil("543f0d00-97ba-11ed-86fe-ef2b82ea3c6f"),
			externalHost:    "example.com",
			encapsulation:   externalmedia.EncapsulationRTP,
			transport:       externalmedia.TransportUDP,
			connectionType:  "client",
			format:          "ulaw",
			directionListen: externalmedia.DirectionBoth,
			directionSpeak:  externalmedia.DirectionBoth,

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("543f0d00-97ba-11ed-86fe-ef2b82ea3c6f"),
				},
				BridgeID: "5466b238-97ba-11ed-9021-0b336edbced2",
			},
			responseBridge: &bridge.Bridge{
				ID:         "5466b238-97ba-11ed-9021-0b336edbced2",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			responseUUIDChannelID: uuid.FromStringOrNil("548cc82e-97ba-11ed-9f0c-43e1928c2d6e"),

			expectExternalHost: "example.com",
			expectChannelData:  "context_type=call,context=call-externalmedia,bridge_id=5466b238-97ba-11ed-9021-0b336edbced2,reference_type=confbridge,reference_id=543f0d00-97ba-11ed-86fe-ef2b82ea3c6f,external_media_id=79076e90-b331-11ef-bc31-33cb17f32724",
			expectExternalMedia: &externalmedia.ExternalMedia{
				ID:              uuid.FromStringOrNil("79076e90-b331-11ef-bc31-33cb17f32724"),
				AsteriskID:      "42:01:0a:a4:00:05",
				ChannelID:       "548cc82e-97ba-11ed-9f0c-43e1928c2d6e",
				ReferenceType:   externalmedia.ReferenceTypeConfbridge,
				ReferenceID:     uuid.FromStringOrNil("543f0d00-97ba-11ed-86fe-ef2b82ea3c6f"),
				LocalIP:         "",
				LocalPort:       0,
				ExternalHost:    "example.com",
				Encapsulation:   defaultEncapsulation,
				Transport:       defaultTransport,
				ConnectionType:  defaultConnectionType,
				Format:          defaultFormat,
				DirectionListen: defaultDirection,
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &externalMediaHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.referenceID.Return(tt.responseConfbridge, nil)
			mockBridge.EXPECT().Get(ctx, tt.responseConfbridge.BridgeID.Return(tt.responseBridge, nil)

			// startExternalMedia
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseBridge.AsteriskID, tt.responseUUIDChannelID.String(), tt.expectExternalHost, string(tt.encapsulation), string(tt.transport), defaultConnectionType, defaultFormat, defaultDirection, tt.expectChannelData, gomock.Any().Return(&channel.Channel{}, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia.Return(nil)

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id.Return(tt.expectExternalMedia, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any().Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id.Return(tt.expectExternalMedia, nil)

			res, err := h.Start(
				ctx,
				tt.id,
				tt.referenceType,
				tt.referenceID,
				tt.externalHost,
				tt.encapsulation,
				tt.transport,
				tt.connectionType,
				tt.format,
				tt.directionListen,
				tt.directionSpeak,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectExternalMedia, res) {
				t.Errorf("Wrong match.\nexpect: %vgot: %v", tt.expectExternalMedia, res)
			}
		})
	}
}
