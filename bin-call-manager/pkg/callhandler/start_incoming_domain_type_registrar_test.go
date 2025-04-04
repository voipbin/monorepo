package callhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cfconference "monorepo/bin-conference-manager/models/conference"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
					CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
				},
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1d82f6c0-e6a6-4718-8f23-720f845a8fbe"),
				},
			},

			expectCustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			expectAgentID:    uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: map[string]any{
						"source": map[string]any{
							"type":   "extension",
							"target": "test-exten",
						},
						"destinations": []map[string]any{
							{
								"type":   "agent",
								"target": "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
							},
						},
					},
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
			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), gomock.Any()).Return([]rmextension.Extension{}, nil)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.expectAgentID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.expectCustomerID,
				fmflow.TypeFlow,
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				false,
			).DoAndReturn(func(
				_ context.Context,
				_ uuid.UUID,
				_ fmflow.Type,
				_ string,
				_ string,
				actions []fmaction.Action,
				_ bool,
			) (*fmflow.Flow, error) {
				tmp, err := json.Marshal(actions)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
					return nil, err
				}

				tmp2, err := json.Marshal(tt.expectActions)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
					return nil, err
				}

				if !reflect.DeepEqual(tmp, tmp2) {
					t.Errorf("unexpected actions:\nexpected: %#v\ngot: %#v", string(tmp2), string(tmp))
				}

				return tt.responseFlow, nil
			})

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
		responseExtensions  []rmextension.Extension
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
			responseExtensions: []rmextension.Extension{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("fd825ef8-3070-11ef-9d4f-7fde01005dda"),
					},
				},
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeConference,
				Target: "99accfb7-c0dd-4a54-997d-dd18af7bc280",
			},
			responseConference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
					CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
				},
				FlowID: uuid.FromStringOrNil("90f05e61-408b-429b-85fb-0ee3d2d77c21"),
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
				},
			},

			expectCustomerID:   uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			expectConferenceID: uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeConferenceJoin,
					Option: map[string]any{
						"conference_id": "99accfb7-c0dd-4a54-997d-dd18af7bc280",
					},
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
			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), gomock.Any()).Return(tt.responseExtensions, nil)
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
		responseAgent       *amagent.Agent
		responseDestination *commonaddress.Address
		responseFlow        *fmflow.Flow

		expectCustomerID   uuid.UUID
		expectAgent        *amagent.Agent
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
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7cb15cd0-2fe8-11ef-b367-db54b9814493"),
				},
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
				},
			},

			expectCustomerID:   uuid.FromStringOrNil("b709f75e-57e2-11ee-9e0e-eb6422fe6fd2"),
			expectConferenceID: uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: map[string]any{
						"source": map[string]any{
							"type":   "extension",
							"target": "test-exten",
						},
						"destinations": []map[string]any{
							{
								"type":   "tel",
								"target": "+821100000001",
							},
						},
						"early_media":  true,
						"relay_reason": true,
					},
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

			// parseAddressTypeExtension
			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), gomock.Any()).Return([]rmextension.Extension{}, nil)

			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.expectCustomerID,
				fmflow.TypeFlow,
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				false,
			).DoAndReturn(func(
				_ context.Context,
				_ uuid.UUID,
				_ fmflow.Type,
				_ string,
				_ string,
				actions []fmaction.Action,
				_ bool,
			) (*fmflow.Flow, error) {
				tmp, err := json.Marshal(actions)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
					return nil, err
				}

				tmp2, err := json.Marshal(tt.expectActions)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
					return nil, err
				}

				if !reflect.DeepEqual(tmp, tmp2) {
					t.Errorf("unexpected actions:\nexpected: %#v\ngot: %#v", string(tmp2), string(tmp))
				}

				return tt.responseFlow, nil
			})

			// startCallTypeFlow
			// we don't go further. just return the error
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.expectCustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(false, nil)
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
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("eb145bae-2814-11ef-b5c9-fb53bd2bff02"),
					},
					Extension: "test-destination",
				},
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
				},
			},

			expectCustomerID: uuid.FromStringOrNil("49c42d3c-57eb-11ee-95a1-2778bda73d76"),
			expectFilters: map[string]string{
				"customer_id": "49c42d3c-57eb-11ee-95a1-2778bda73d76",
				"deleted":     "false",
				"extension":   "test-destination",
			},
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: map[string]any{
						"source": map[string]any{
							"type":   "extension",
							"target": "test-exten",
						},
						"destinations": []map[string]any{
							{
								"type":        "extension",
								"target":      "eb145bae-2814-11ef-b5c9-fb53bd2bff02",
								"target_name": "test-destination",
							},
						},
					},
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

			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), gomock.Any()).Return([]rmextension.Extension{}, nil)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)

			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseExtensions, nil)

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.expectCustomerID,
				fmflow.TypeFlow,
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				false,
			).DoAndReturn(func(
				_ context.Context,
				_ uuid.UUID,
				_ fmflow.Type,
				_ string,
				_ string,
				actions []fmaction.Action,
				_ bool,
			) (*fmflow.Flow, error) {
				tmp, err := json.Marshal(actions)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
					return nil, err
				}

				tmp2, err := json.Marshal(tt.expectActions)
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
					return nil, err
				}

				if !reflect.DeepEqual(tmp, tmp2) {
					t.Errorf("unexpected actions:\nexpected: %#v\ngot: %#v", string(tmp2), string(tmp))
				}

				return tt.responseFlow, nil
			})

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

func Test_parseAddressTypeExtension(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		address    *commonaddress.Address

		responseExtension  *rmextension.Extension
		responseExtensions []rmextension.Extension

		expectExtensionID uuid.UUID
		expectFilters     map[string]string
		expectRes         *commonaddress.Address
	}{
		{
			name: "normal - address has correct uuid target",

			customerID: uuid.FromStringOrNil("9884e39e-3071-11ef-9e2e-bfa99d572134"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "b5352e7c-3071-11ef-8ca8-1f8365f8db34",
			},

			responseExtension: &rmextension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b5352e7c-3071-11ef-8ca8-1f8365f8db34"),
					CustomerID: uuid.FromStringOrNil("9884e39e-3071-11ef-9e2e-bfa99d572134"),
				},
				Extension: "2000",
			},

			expectExtensionID: uuid.FromStringOrNil("b5352e7c-3071-11ef-8ca8-1f8365f8db34"),
			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeExtension,
				Target:     "b5352e7c-3071-11ef-8ca8-1f8365f8db34",
				TargetName: "2000",
			},
		},
		{
			name: "normal - address has invalid uuid target",

			customerID: uuid.FromStringOrNil("b556c3d4-3071-11ef-bb2d-ab2af3aa5a97"),
			address: &commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "3000",
			},

			responseExtensions: []rmextension.Extension{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("b5710de8-3071-11ef-a281-e3ba0cb3824b"),
						CustomerID: uuid.FromStringOrNil("b556c3d4-3071-11ef-bb2d-ab2af3aa5a97"),
					},
					Extension: "3000",
				},
			},

			expectFilters: map[string]string{
				"customer_id": "b556c3d4-3071-11ef-bb2d-ab2af3aa5a97",
				"deleted":     "false",
				"extension":   "3000",
			},
			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeExtension,
				Target:     "b5710de8-3071-11ef-a281-e3ba0cb3824b",
				TargetName: "3000",
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

			if tt.responseExtension != nil {
				mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.expectExtensionID).Return(tt.responseExtension, nil)
			} else {
				mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseExtensions, nil)
			}

			res, err := h.parseAddressTypeExtension(ctx, tt.customerID, tt.address)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
