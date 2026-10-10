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

func Test_Start_startReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		id              uuid.UUID
		referenceType   externalmedia.ReferenceType
		referenceID     uuid.UUID
		externalHost    string
		encapsulation   externalmedia.Encapsulation
		transport       externalmedia.Transport
		transportData   string
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseCall.ChannelID).Return(tt.responseChannel, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDBridgeID)
			mockBridge.EXPECT().Start(ctx, tt.responseChannel.AsteriskID, tt.responseUUIDBridgeID.String(), tt.expectBridgeArgs, []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(tt.responseBridge, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDSnoopID)
			mockChannel.EXPECT().StartSnoop(ctx, tt.responseCall.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirection(tt.directionListen), channel.SnoopDirection(tt.directionSpeak)).Return(&channel.Channel{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseChannel.AsteriskID, gomock.Any(), tt.externalHost, string(tt.encapsulation), string(tt.transport), tt.transportData, tt.connectionType, tt.format, string(tt.directionListen), tt.expectChannelData, gomock.Any()).Return(&channel.Channel{}, nil)

			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia).Return(nil)

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.expectExternalMedia, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.expectExternalMedia, nil)

			res, err := h.Start(
				ctx,
				tt.id,
				tt.referenceType,
				tt.referenceID,
				tt.externalHost,
				tt.encapsulation,
				tt.transport,
				tt.transportData,
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
		transportData   string
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

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.referenceID).Return(tt.responseConfbridge, nil)
			mockBridge.EXPECT().Get(ctx, tt.responseConfbridge.BridgeID).Return(tt.responseBridge, nil)

			// startExternalMedia
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseBridge.AsteriskID, tt.responseUUIDChannelID.String(), tt.expectExternalHost, string(tt.encapsulation), string(tt.transport), tt.transportData, tt.connectionType, defaultFormat, defaultDirection, tt.expectChannelData, gomock.Any()).Return(&channel.Channel{}, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, tt.expectExternalMedia).Return(nil)

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.expectExternalMedia, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.expectExternalMedia, nil)

			res, err := h.Start(
				ctx,
				tt.id,
				tt.referenceType,
				tt.referenceID,
				tt.externalHost,
				tt.encapsulation,
				tt.transport,
				tt.transportData,
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

func Test_isCallBridgeEligible(t *testing.T) {
	eligibleCall := func() *call.Call {
		return &call.Call{
			Status:   call.StatusProgressing,
			BridgeID: "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba",
		}
	}

	allDirections := []externalmedia.Direction{
		externalmedia.DirectionNone,
		externalmedia.DirectionBoth,
		externalmedia.DirectionIn,
		externalmedia.DirectionOut,
	}

	tests := []struct {
		name string

		call         func() *call.Call
		externalHost string
		listen       externalmedia.Direction
		speak        externalmedia.Direction

		expectRes bool
	}{
		{name: "ineligible status dialing", call: func() *call.Call { c := eligibleCall(); c.Status = call.StatusDialing; return c }, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible status ringing", call: func() *call.Call { c := eligibleCall(); c.Status = call.StatusRinging; return c }, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible empty bridge id", call: func() *call.Call { c := eligibleCall(); c.BridgeID = ""; return c }, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible confbridge id", call: func() *call.Call {
			c := eligibleCall()
			c.ConfbridgeID = uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")
			return c
		}, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible master call id", call: func() *call.Call {
			c := eligibleCall()
			c.MasterCallID = uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4072")
			return c
		}, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible chained call ids", call: func() *call.Call {
			c := eligibleCall()
			c.ChainedCallIDs = []uuid.UUID{uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4073")}
			return c
		}, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible groupcall id", call: func() *call.Call {
			c := eligibleCall()
			c.GroupcallID = uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4074")
			return c
		}, externalHost: externalHostIncoming, listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible customer host", call: eligibleCall, externalHost: "example.com", listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
		{name: "ineligible empty host", call: eligibleCall, externalHost: "", listen: externalmedia.DirectionNone, speak: externalmedia.DirectionOut, expectRes: false},
	}

	// all direction combinations on an eligible call: only (none|in, out) is eligible
	for _, listen := range allDirections {
		for _, speak := range allDirections {
			expect := speak == externalmedia.DirectionOut && (listen == externalmedia.DirectionNone || listen == externalmedia.DirectionIn)
			tests = append(tests, struct {
				name         string
				call         func() *call.Call
				externalHost string
				listen       externalmedia.Direction
				speak        externalmedia.Direction
				expectRes    bool
			}{
				name:         "direction listen=" + string(listen) + " speak=" + string(speak),
				call:         eligibleCall,
				externalHost: externalHostIncoming,
				listen:       listen,
				speak:        speak,
				expectRes:    expect,
			})
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, reason := isCallBridgeEligible(tt.call(), tt.externalHost, tt.listen, tt.speak)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v, reason: %s", tt.expectRes, res, reason)
			}
			if res && reason != "" {
				t.Errorf("Wrong match. eligible must have an empty reason, got: %s", reason)
			}
			if !res && reason == "" {
				t.Errorf("Wrong match. ineligible must have a reason")
			}
		})
	}
}

func Test_Start_startReferenceTypeCall_callBridge(t *testing.T) {
	tests := []struct {
		name string

		id              uuid.UUID
		referenceID     uuid.UUID
		externalHost    string
		directionListen externalmedia.Direction
		directionSpeak  externalmedia.Direction

		responseCall          *call.Call
		responseChannel       *channel.Channel
		responseUUIDChannelID uuid.UUID

		expectChannelData   string
		expectExternalMedia *externalmedia.ExternalMedia
	}{
		{
			name: "eligible call joins the call bridge without a snoop channel",

			id:              uuid.FromStringOrNil("78473c24-b331-11ef-aa9c-e7c52f9d3f7b"),
			referenceID:     uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
			externalHost:    "INCOMING",
			directionListen: externalmedia.DirectionNone,
			directionSpeak:  externalmedia.DirectionOut,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				},
				ChannelID: "8066017c-02fb-11ec-ba6c-c320820accf1",
				BridgeID:  "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba",
				Status:    call.StatusProgressing,
			},
			responseChannel: &channel.Channel{
				AsteriskID: "42:01:0a:a4:00:05",
				ID:         "8066017c-02fb-11ec-ba6c-c320820accf1",
			},
			responseUUIDChannelID: uuid.FromStringOrNil("488feb00-96e3-11ed-8ae7-1fe9bc7a995f"),

			expectChannelData: "context_type=call,context=call-externalmedia,bridge_id=51bf770e-7f1b-11f0-ac50-971ccbe0a7ba,reference_type=call,reference_id=7f6dbc1a-02fb-11ec-897b-ef9b30e25c57,external_media_id=78473c24-b331-11ef-aa9c-e7c52f9d3f7b",
			expectExternalMedia: &externalmedia.ExternalMedia{
				ID:              uuid.FromStringOrNil("78473c24-b331-11ef-aa9c-e7c52f9d3f7b"),
				AsteriskID:      "42:01:0a:a4:00:05",
				ChannelID:       "488feb00-96e3-11ed-8ae7-1fe9bc7a995f",
				BridgeID:        "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba",
				ReferenceType:   externalmedia.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57"),
				ExternalHost:    "INCOMING",
				Encapsulation:   "rtp",
				Transport:       "udp",
				ConnectionType:  "client",
				Format:          "ulaw",
				DirectionListen: externalmedia.DirectionNone,
				DirectionSpeak:  externalmedia.DirectionOut,
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

			// no bridgeHandler.Start, no channelHandler.StartSnoop: gomock fails on any unexpected call.
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChannelID)
			mockChannel.EXPECT().StartExternalMedia(ctx, tt.responseChannel.AsteriskID, gomock.Any(), tt.externalHost, "rtp", "udp", "", "client", "ulaw", gomock.Any(), tt.expectChannelData, gomock.Any()).Return(&channel.Channel{}, nil)

			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.expectExternalMedia, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.id).Return(tt.expectExternalMedia, nil)

			res, err := h.Start(ctx, tt.id, externalmedia.ReferenceTypeCall, tt.referenceID, tt.externalHost, externalmedia.EncapsulationRTP, "udp", "", "client", "ulaw", tt.directionListen, tt.directionSpeak)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectExternalMedia, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectExternalMedia, res)
			}
		})
	}
}

func Test_Start_startReferenceTypeCall_ineligibleUsesSnoop(t *testing.T) {
	tests := []struct {
		name string

		externalHost string
		call         *call.Call
	}{
		{
			name:         "customer supplied host keeps the snoop path",
			externalHost: "example.com",
			call: &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57")},
				ChannelID: "8066017c-02fb-11ec-ba6c-c320820accf1",
				BridgeID:  "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba",
				Status:    call.StatusProgressing,
			},
		},
		{
			name:         "call in a conference keeps the snoop path",
			externalHost: "INCOMING",
			call: &call.Call{
				Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("7f6dbc1a-02fb-11ec-897b-ef9b30e25c57")},
				ChannelID:    "8066017c-02fb-11ec-ba6c-c320820accf1",
				BridgeID:     "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba",
				Status:       call.StatusProgressing,
				ConfbridgeID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
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
			responseChannel := &channel.Channel{AsteriskID: "42:01:0a:a4:00:05", ID: tt.call.ChannelID}
			responseBridge := &bridge.Bridge{ID: "9b6c7a78-96e3-11ed-904b-9baa2c0183fd"}

			mockReq.EXPECT().CallV1CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockChannel.EXPECT().Get(ctx, tt.call.ChannelID).Return(responseChannel, nil)

			// the legacy snoop path: private bridge and snoop channel are created.
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("9b6c7a78-96e3-11ed-904b-9baa2c0183fd"))
			mockBridge.EXPECT().Start(ctx, responseChannel.AsteriskID, gomock.Any(), gomock.Any(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(responseBridge, nil)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("80981342-96e3-11ed-bc85-830940cba8ea"))
			mockChannel.EXPECT().StartSnoop(ctx, tt.call.ChannelID, gomock.Any(), gomock.Any(), channel.SnoopDirection(externalmedia.DirectionNone), channel.SnoopDirection(externalmedia.DirectionOut)).Return(&channel.Channel{}, nil)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("488feb00-96e3-11ed-8ae7-1fe9bc7a995f"))
			mockChannel.EXPECT().StartExternalMedia(ctx, responseChannel.AsteriskID, gomock.Any(), tt.externalHost, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			em := &externalmedia.ExternalMedia{ID: uuid.FromStringOrNil("78473c24-b331-11ef-aa9c-e7c52f9d3f7b")}
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil).Times(2)
			mockDB.EXPECT().ExternalMediaGet(ctx, em.ID).Return(em, nil).Times(2)

			if _, err := h.Start(ctx, em.ID, externalmedia.ReferenceTypeCall, tt.call.ID, tt.externalHost, externalmedia.EncapsulationRTP, "udp", "", "client", "ulaw", externalmedia.DirectionNone, externalmedia.DirectionOut); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
