package callhandler

import (
	"context"
	"fmt"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_startIncomingDomainTypeRegistrar_DestinationTypeAgent(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseAgent       *amagent.Agent
		responseFlow        *fmflow.Flow

		expectCustomerID uuid.UUID
		expectAgentID    uuid.UUID
		expectActions    []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.171",

				DestinationName:   "",
				DestinationNumber: "agent%3Aeb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
				SourceName:        "",
				SourceNumber:      "test-exten",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "a7be89e0-8170-4f48-ac01-a81a31c6e344.registrar.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "test-exten",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
			},
			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("1d82f6c0-e6a6-4718-8f23-720f845a8fbe"),
			},

			expectCustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			expectAgentID:    uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConnect,
					Option: []byte(`{"source":{"type":"extension","target":"test-exten","target_name":"test-exten","name":"","detail":""},"destinations":[{"type":"agent","target":"eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b","target_name":"","name":"","detail":""}],"early_media":false,"relay_reason":false}`),
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeExtension).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.expectAgentID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.expectCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.expectCustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeRegistrar(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeRegistrar_DestinationTypeConference(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseConference  *cfconference.Conference
		responseFlow        *fmflow.Flow

		expectCustomerID   uuid.UUID
		expectConferenceID uuid.UUID
		expectActions      []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675220154.178",

				DestinationName:   "",
				DestinationNumber: "conference-99accfb7-c0dd-4a54-997d-dd18af7bc280",
				SourceName:        "",
				SourceNumber:      "test-exten",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "a7be89e0-8170-4f48-ac01-a81a31c6e344.registrar.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "test-exten",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeConference,
				Target: "99accfb7-c0dd-4a54-997d-dd18af7bc280",
			},
			responseConference: &cfconference.Conference{
				ID:         uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
				FlowID:     uuid.FromStringOrNil("90f05e61-408b-429b-85fb-0ee3d2d77c21"),
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
			},

			expectCustomerID:   uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			expectConferenceID: uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"99accfb7-c0dd-4a54-997d-dd18af7bc280"}`),
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeExtension).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.expectConferenceID).Return(tt.responseConference, nil)
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.expectCustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.expectCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeRegistrar(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeRegistrar_DestinationTypeTel(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseFlow        *fmflow.Flow

		expectCustomerID   uuid.UUID
		expectConferenceID uuid.UUID
		expectActions      []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675220876.181",

				DestinationName:   "",
				DestinationNumber: "+821100000001",
				SourceName:        "",
				SourceNumber:      "test-exten",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "b709f75e-57e2-11ee-9e0e-eb6422fe6fd2.registrar.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "test-exten",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
			},

			expectCustomerID:   uuid.FromStringOrNil("b709f75e-57e2-11ee-9e0e-eb6422fe6fd2"),
			expectConferenceID: uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConnect,
					Option: []byte(`{"source":{"type":"extension","target":"test-exten","target_name":"test-exten","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""}],"early_media":true,"relay_reason":true}`),
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeExtension).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.expectCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.expectCustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeRegistrar(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeRegistrarDestinationTypeExtension(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseExtensions  []rmextension.Extension
		responseFlow        *fmflow.Flow

		expectCustomerID uuid.UUID
		expectFilters    map[string]string
		expectActions    []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675220876.181",

				DestinationName:   "",
				DestinationNumber: "test-destination",
				SourceName:        "",
				SourceNumber:      "test-exten",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "49c42d3c-57eb-11ee-95a1-2778bda73d76.registrar.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "test-exten",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "test-destination",
			},
			responseExtensions: []rmextension.Extension{
				{
					ID:        uuid.FromStringOrNil("eb145bae-2814-11ef-b5c9-fb53bd2bff02"),
					Extension: "test-destination",
				},
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
			},

			expectCustomerID: uuid.FromStringOrNil("49c42d3c-57eb-11ee-95a1-2778bda73d76"),
			expectFilters: map[string]string{
				"customer_id": "49c42d3c-57eb-11ee-95a1-2778bda73d76",
				"deleted":     "false",
				"extension":   "test-destination",
			},
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConnect,
					Option: []byte(`{"source":{"type":"extension","target":"test-exten","target_name":"test-exten","name":"","detail":""},"destinations":[{"type":"extension","target":"eb145bae-2814-11ef-b5c9-fb53bd2bff02","target_name":"test-destination","name":"","detail":""}],"early_media":false,"relay_reason":false}`),
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeExtension).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseExtensions, nil)

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.expectCustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.expectCustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeRegistrar(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
