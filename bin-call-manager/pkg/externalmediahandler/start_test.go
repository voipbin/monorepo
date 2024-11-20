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
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Start_reference_type_call_with_insert_media(t *testing.T) {
	tests := []struct {
		name string

		referenceType  externalmedia.ReferenceType
		referenceID    uuid.UUID
		externalHost   string
		encapsulation  externalmedia.Encapsulation
		transport      externalmedia.Transport
		connectionType string
		format         string
		direction      string

		responseCall                *call.Call
		responseChannel             *channel.Channel
		responseUUIDChannelID       uuid.UUID
		responseUUIDExternalMediaID uuid.UUID

		expectBridgeArgs     string
		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string
		expectChannelData    string
		expectExternalMedia  *externalmedia.ExternalMedia
	}{
		{
			"normal",

			externalmedia.ReferenceTypeCall,
			uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
			"example.com",
			externalmedia.EncapsulationRTP,
			externalmedia.TransportUDP,
			"client",
			"ulaw",
			"both",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				},
				ChannelID: "8066017c-02fb-11ec-ba6c-c320820accf1",
				BridgeID:  "500d0b6e-eb39-11ee-a30a-9392749106cc",
			},
			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "8066017c-02fb-11ec-ba6c-c320820accf1",
			},
			uuid.FromStringOrNil("488feb00-96e3-11ed-8ae7-1fe9bc7a995f"),
			uuid.FromStringOrNil("ae01d90e-96e2-11ed-8b03-f31329c0298c"),

			"reference_type=call-snoop,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57",
			"example.com",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",
			"context_type=call,context=call-externalmedia,bridge_id=500d0b6e-eb39-11ee-a30a-9392749106cc,reference_type=call,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57",
			&externalmedia.ExternalMedia{
				ID:             uuid.FromStringOrNil("ae01d90e-96e2-11ed-8b03-f31329c0298c"),
				AsteriskID:     "42:01:0a:a4:00:05",
				ChannelID:      "488feb00-96e3-11ed-8ae7-1fe9bc7a995f",
				ReferenceType:  externalmedia.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				LocalIP:        "",
				LocalPort:      0,
				ExternalHost:   "example.com",
				Encapsulation:  "rtp",
				Transport:      "udp",
				ConnectionType: "client",
				Format:         "ulaw",
				Direction:      "both",
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID).Return(tt.responseChannel, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseChannel.AsteriskID, gomock.Any(), tt.expectExternalHost, tt.expectEncapsulation, tt.expectTransport, tt.expectConnectionType, tt.expectFormat, tt.expectDirection, tt.expectChannelData, gomock.Any()).Return(&channel.Channel{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDExternalMediaID)
			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia).Return(nil)

			res, err := h.Start(ctx, tt.referenceType, tt.referenceID, false, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectExternalMedia, res) {
				t.Errorf("Wrong match.\nexpect: %vgot: %v", tt.expectExternalMedia, res)
			}
		})
	}
}

func Test_Start_reference_type_call_without_insert_media(t *testing.T) {
	tests := []struct {
		name string

		referenceType  externalmedia.ReferenceType
		referenceID    uuid.UUID
		externalHost   string
		encapsulation  externalmedia.Encapsulation
		transport      externalmedia.Transport
		connectionType string
		format         string
		direction      channel.SnoopDirection

		responseCall                *call.Call
		responseChannel             *channel.Channel
		responseUUIDBridgeID        uuid.UUID
		responseBridge              *bridge.Bridge
		responseUUIDSnoopID         uuid.UUID
		responseUUIDChannelID       uuid.UUID
		responseUUIDExternalMediaID uuid.UUID

		expectBridgeArgs    string
		expectChannelData   string
		expectExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",

			externalmedia.ReferenceTypeCall,
			uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
			"example.com",
			externalmedia.EncapsulationRTP,
			"udp",
			"client",
			"ulaw",
			channel.SnoopDirectionBoth,

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				},
				// AsteriskID: "42:01:0a:a4:00:05",
				ChannelID: "8066017c-02fb-11ec-ba6c-c320820accf1",
			},
			&channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "8066017c-02fb-11ec-ba6c-c320820accf1",
			},
			uuid.FromStringOrNil("9b6c7a78-96e3-11ed-904b-9baa2c0183fd"),
			&bridge.Bridge{
				ID: "9b6c7a78-96e3-11ed-904b-9baa2c0183fd",
			},
			uuid.FromStringOrNil("80981342-96e3-11ed-bc85-830940cba8ea"),
			uuid.FromStringOrNil("488feb00-96e3-11ed-8ae7-1fe9bc7a995f"),
			uuid.FromStringOrNil("ae01d90e-96e2-11ed-8b03-f31329c0298c"),

			"reference_type=call-snoop,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57",
			"context_type=call,context=call-externalmedia,bridge_id=9b6c7a78-96e3-11ed-904b-9baa2c0183fd,reference_type=call,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57",
			&externalmedia.ExternalMedia{
				ID:             uuid.FromStringOrNil("ae01d90e-96e2-11ed-8b03-f31329c0298c"),
				AsteriskID:     "42:01:0a:a4:00:05",
				ChannelID:      "488feb00-96e3-11ed-8ae7-1fe9bc7a995f",
				ReferenceType:  externalmedia.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				LocalIP:        "",
				LocalPort:      0,
				ExternalHost:   "example.com",
				Encapsulation:  "rtp",
				Transport:      "udp",
				ConnectionType: "client",
				Format:         "ulaw",
				Direction:      "both",
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID).Return(tt.responseChannel, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDBridgeID)
			mockBridge.EXPECT().Start(ctx, tt.responseChannel.AsteriskID, tt.responseUUIDBridgeID.String(), tt.expectBridgeArgs, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(tt.responseBridge, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDSnoopID)
			mockChannel.EXPECT().StartSnoop(ctx, tt.responseCall.ChannelID, gomock.Any(), gomock.Any(), tt.direction, channel.SnoopDirectionBoth).Return(&channel.Channel{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseChannel.AsteriskID, gomock.Any(), tt.externalHost, string(tt.encapsulation), string(tt.transport), tt.connectionType, tt.format, string(tt.direction), tt.expectChannelData, gomock.Any()).Return(&channel.Channel{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDExternalMediaID)
			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia).Return(nil)

			res, err := h.Start(ctx, tt.referenceType, tt.referenceID, true, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, string(tt.direction))
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

		referenceType  externalmedia.ReferenceType
		referenceID    uuid.UUID
		externalHost   string
		encapsulation  externalmedia.Encapsulation
		transport      externalmedia.Transport
		connectionType string
		format         string
		direction      string

		responseConfbridge          *confbridge.Confbridge
		responseBridge              *bridge.Bridge
		responseUUIDChannelID       uuid.UUID
		responseUUIDExternalMediaID uuid.UUID

		expectExternalHost  string
		expectChannelData   string
		expectExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",

			externalmedia.ReferenceTypeConfbridge,
			uuid.FromStringOrNil("543f0d00-97ba-11ed-86fe-ef2b82ea3c6f"),
			"example.com",
			externalmedia.EncapsulationRTP,
			externalmedia.TransportUDP,
			"client",
			"ulaw",
			"both",

			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("543f0d00-97ba-11ed-86fe-ef2b82ea3c6f"),
				BridgeID: "5466b238-97ba-11ed-9021-0b336edbced2",
			},
			&bridge.Bridge{
				ID:         "5466b238-97ba-11ed-9021-0b336edbced2",
				AsteriskID: "42:01:0a:a4:00:05",
			},
			uuid.FromStringOrNil("548cc82e-97ba-11ed-9f0c-43e1928c2d6e"),
			uuid.FromStringOrNil("54b24914-97ba-11ed-952b-7ff363f5a0a0"),

			"example.com",
			"context_type=call,context=call-externalmedia,bridge_id=5466b238-97ba-11ed-9021-0b336edbced2,reference_type=confbridge,reference_id=543f0d00-97ba-11ed-86fe-ef2b82ea3c6f",
			&externalmedia.ExternalMedia{
				ID:             uuid.FromStringOrNil("54b24914-97ba-11ed-952b-7ff363f5a0a0"),
				AsteriskID:     "42:01:0a:a4:00:05",
				ChannelID:      "548cc82e-97ba-11ed-9f0c-43e1928c2d6e",
				ReferenceType:  externalmedia.ReferenceTypeConfbridge,
				ReferenceID:    uuid.FromStringOrNil("543f0d00-97ba-11ed-86fe-ef2b82ea3c6f"),
				LocalIP:        "",
				LocalPort:      0,
				ExternalHost:   "example.com",
				Encapsulation:  defaultEncapsulation,
				Transport:      defaultTransport,
				ConnectionType: defaultConnectionType,
				Format:         defaultFormat,
				Direction:      defaultDirection,
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

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.referenceID).Return(tt.responseConfbridge, nil)
			mockBridge.EXPECT().Get(ctx, tt.responseConfbridge.BridgeID).Return(tt.responseBridge, nil)

			// startExternalMedia
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseBridge.AsteriskID, tt.responseUUIDChannelID.String(), tt.expectExternalHost, string(tt.encapsulation), string(tt.transport), defaultConnectionType, defaultFormat, defaultDirection, tt.expectChannelData, gomock.Any()).Return(&channel.Channel{}, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDExternalMediaID)
			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia).Return(nil)

			res, err := h.Start(ctx, tt.referenceType, tt.referenceID, false, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, "both")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectExternalMedia, res) {
				t.Errorf("Wrong match.\nexpect: %vgot: %v", tt.expectExternalMedia, res)
			}
		})
	}
}
