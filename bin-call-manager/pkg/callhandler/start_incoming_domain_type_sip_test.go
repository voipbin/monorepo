package callhandler

import (
	"context"
	"fmt"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"
	nmnumber "monorepo/bin-number-manager/models/number"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_startIncomingDomainTypeSIP(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseNumbers     []nmnumber.Number

		expectFilters map[nmnumber.Field]any
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.200",

				DestinationName:   "",
				DestinationNumber: "+821100000001",
				SourceName:        "",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			responseNumbers: []nmnumber.Number{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bd484f7e-09ef-11eb-9347-377b97e1b9ea"),
						CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
					},
					CallFlowID: uuid.FromStringOrNil("d2e558c2-09ef-11eb-bdec-e3ef3b78ac73"),
				},
			},

			expectFilters: map[nmnumber.Field]any{
				nmnumber.FieldNumber:  "+821100000001",
				nmnumber.FieldDeleted: false,
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeTel).Return(tt.responseDestination)

			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseNumbers, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseNumbers[0].CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseNumbers[0].CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_numberListError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address

		expectFilters map[nmnumber.Field]any
	}{
		{
			name: "number list returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.201",

				DestinationNumber: "+821100000001",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},

			expectFilters: map[nmnumber.Field]any{
				nmnumber.FieldNumber:  "+821100000001",
				nmnumber.FieldDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeTel).Return(tt.responseDestination)

			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), tt.expectFilters).Return(nil, fmt.Errorf("number service error"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_emptyNumbers(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address

		expectFilters map[nmnumber.Field]any
	}{
		{
			name: "no numbers found",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.202",

				DestinationNumber: "+821199999999",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821199999999",
			},

			expectFilters: map[nmnumber.Field]any{
				nmnumber.FieldNumber:  "+821199999999",
				nmnumber.FieldDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeTel).Return(tt.responseDestination)

			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), tt.expectFilters).Return([]nmnumber.Number{}, nil)
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_nilCallFlowID(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseNumbers     []nmnumber.Number

		expectFilters map[nmnumber.Field]any
	}{
		{
			name: "number has no call flow configured",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.203",

				DestinationNumber: "+821100000001",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			responseNumbers: []nmnumber.Number{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bd484f7e-09ef-11eb-9347-377b97e1b9ea"),
						CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
					},
					// CallFlowID is uuid.Nil (not configured)
				},
			},

			expectFilters: map[nmnumber.Field]any{
				nmnumber.FieldNumber:  "+821100000001",
				nmnumber.FieldDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeTel).Return(tt.responseDestination)

			mockReq.EXPECT().NumberV1NumberList(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseNumbers, nil)
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directExtension(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource    *commonaddress.Address
		responseDirect    *dmdirect.Direct
		responseExtension *rmextension.Extension
		responseFlow      *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.300",

				DestinationNumber: "direct.a3f8b2c1d4e5",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d2d3d4-d5d6-d7d8-d9da-dbdcdddedfee"),
					CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				Hash:         "a3f8b2c1d4e5",
			},
			responseExtension: &rmextension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
				},
				Extension: "1001",
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1f2f3f4-f5f6-f7f8-f9fa-fbfcfdfeff00"),
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.a3f8b2c1d4e5").Return(tt.responseDirect, nil)
			mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseExtension, nil)

			expectDestination := commonaddress.Address{
				Type:       commonaddress.TypeExtension,
				Target:     tt.responseExtension.ID.String(),
				TargetName: tt.responseExtension.Extension,
			}
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: fmaction.ConvertOption(fmaction.OptionConnect{
						Source:       *tt.responseSource,
						Destinations: []commonaddress.Address{expectDestination},
						EarlyMedia:   false,
						RelayReason:  false,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseExtension.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				"tmp flow for direct extension dialing",
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseExtension.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseExtension.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directHashNotFound(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "hash not found",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.301",

				DestinationNumber: "direct.invalidhash12",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.invalidhash12").Return(nil, fmt.Errorf("not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAI(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource *commonaddress.Address
		responseDirect *dmdirect.Direct
		responseAI     *amai.AI
		responseFlow   *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.400",

				DestinationNumber: "direct.e90e5ce89f4d",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("27b03598-34df-46af-8b37-8d1ee0d7439d"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
				Hash:         "e90e5ce89f4d",
			},
			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-ai",
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1f2f3f4-f5f6-f7f8-f9fa-fbfcfdfeff01"),
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.e90e5ce89f4d").Return(tt.responseDirect, nil)
			mockReq.EXPECT().AIV1AIGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseAI, nil)

			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type: fmaction.TypeAITalk,
					Option: fmaction.ConvertOption(fmaction.OptionAITalk{
						AssistanceType: amaicall.AssistanceTypeAI,
						AssistanceID:   tt.responseAI.ID,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseAI.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct ai call. ai_id: %s", tt.responseAI.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseAI.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseAI.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAI_aiGetError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel
		hash    string

		responseDirect *dmdirect.Direct
	}{
		{
			name: "ai_get_error",

			channel: &channel.Channel{
				ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.400",
				DestinationNumber: "direct.e90e5ce89f4d",
				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
			hash: "direct.e90e5ce89f4d",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0d0d0d0-d0d0-d0d0-d0d0-d0d0d0d0d0d0"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})

			// AIV1AIGet returns error
			mockReq.EXPECT().AIV1AIGet(ctx, tt.responseDirect.ResourceID).Return(nil, fmt.Errorf("ai not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAI_flowCreateError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel
		hash    string

		responseDirect *dmdirect.Direct
		responseAI     *amai.AI
	}{
		{
			name: "flow_create_error",

			channel: &channel.Channel{
				ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.400",
				DestinationNumber: "direct.e90e5ce89f4d",
				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
			hash: "direct.e90e5ce89f4d",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0d0d0d0-d0d0-d0d0-d0d0-d0d0d0d0d0d0"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
			},

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-ai",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})

			// AIV1AIGet succeeds
			mockReq.EXPECT().AIV1AIGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseAI, nil)

			// FlowV1FlowCreate returns error
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type: fmaction.TypeAITalk,
					Option: fmaction.ConvertOption(fmaction.OptionAITalk{
						AssistanceType: amaicall.AssistanceTypeAI,
						AssistanceID:   tt.responseAI.ID,
					}),
				},
			}
			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseAI.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct ai call. ai_id: %s", tt.responseAI.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(nil, fmt.Errorf("flow create failed"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNetworkOutOfOrder).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAITeam(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource *commonaddress.Address
		responseDirect *dmdirect.Direct
		responseTeam   *amteam.Team
		responseFlow   *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.500",

				DestinationNumber: "direct.f01e6df78a5b",
				SourceNumber:      "+821100000003",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			},

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e1e1e1e1-e1e1-e1e1-e1e1-e1e1e1e1e1e1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: dmdirect.ResourceTypeAITeam,
				ResourceID:   uuid.FromStringOrNil("3b9f842e-a45b-47c9-b5db-f202898728e4"),
			},

			responseTeam: &amteam.Team{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3b9f842e-a45b-47c9-b5db-f202898728e4"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-ai-team",
			},

			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f2f3f4f5-f6f7-f8f9-fafb-fcfdfeff0102"),
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.channel.DestinationNumber).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)

			// AIV1TeamGet
			mockReq.EXPECT().AIV1TeamGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseTeam, nil)

			// FlowV1FlowCreate
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type: fmaction.TypeAITalk,
					Option: fmaction.ConvertOption(fmaction.OptionAITalk{
						AssistanceType: amaicall.AssistanceTypeTeam,
						AssistanceID:   tt.responseTeam.ID,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseTeam.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct ai team call. team_id: %s", tt.responseTeam.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseTeam.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseTeam.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAITeam_teamGetError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel
		hash    string

		responseDirect *dmdirect.Direct
	}{
		{
			name: "team_get_error",

			channel: &channel.Channel{
				ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.500",
				DestinationNumber: "direct.f01e6df78a5b",
				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
			hash: "direct.f01e6df78a5b",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e1e1e1e1-e1e1-e1e1-e1e1-e1e1e1e1e1e1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: dmdirect.ResourceTypeAITeam,
				ResourceID:   uuid.FromStringOrNil("3b9f842e-a45b-47c9-b5db-f202898728e4"),
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			})

			// AIV1TeamGet returns error
			mockReq.EXPECT().AIV1TeamGet(ctx, tt.responseDirect.ResourceID).Return(nil, fmt.Errorf("team not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAITeam_flowCreateError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel
		hash    string

		responseDirect *dmdirect.Direct
		responseTeam   *amteam.Team
	}{
		{
			name: "flow_create_error",

			channel: &channel.Channel{
				ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.500",
				DestinationNumber: "direct.f01e6df78a5b",
				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
			hash: "direct.f01e6df78a5b",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e1e1e1e1-e1e1-e1e1-e1e1-e1e1e1e1e1e1"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: dmdirect.ResourceTypeAITeam,
				ResourceID:   uuid.FromStringOrNil("3b9f842e-a45b-47c9-b5db-f202898728e4"),
			},

			responseTeam: &amteam.Team{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3b9f842e-a45b-47c9-b5db-f202898728e4"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-ai-team",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000003",
			})

			// AIV1TeamGet succeeds
			mockReq.EXPECT().AIV1TeamGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseTeam, nil)

			// FlowV1FlowCreate returns error
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type: fmaction.TypeAITalk,
					Option: fmaction.ConvertOption(fmaction.OptionAITalk{
						AssistanceType: amaicall.AssistanceTypeTeam,
						AssistanceID:   tt.responseTeam.ID,
					}),
				},
			}
			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseTeam.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct ai team call. team_id: %s", tt.responseTeam.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(nil, fmt.Errorf("flow create failed"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNetworkOutOfOrder).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAgent(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource *commonaddress.Address
		responseDirect *dmdirect.Direct
		responseAgent  *amagent.Agent
		responseFlow   *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.600",

				DestinationNumber: "direct.a12b3c4d5e6f",
				SourceNumber:      "+821100000004",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			},

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c2c2c2c2-c2c2-c2c2-c2c2-c2c2c2c2c2c2"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("4c0e953d-b67a-48f2-a1de-e303907628b5"),
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4c0e953d-b67a-48f2-a1de-e303907628b5"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-agent",
			},

			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f3f4f5f6-f7f8-f9fa-fbfc-fdfeff010203"),
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.channel.DestinationNumber).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)

			// AgentV1AgentGet
			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseAgent, nil)

			// FlowV1FlowCreate
			expectDestination := &commonaddress.Address{
				Type:       commonaddress.TypeAgent,
				Target:     tt.responseAgent.ID.String(),
				TargetName: tt.responseAgent.Name,
			}
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: fmaction.ConvertOption(fmaction.OptionConnect{
						Source:       *tt.responseSource,
						Destinations: []commonaddress.Address{*expectDestination},
						EarlyMedia:   false,
						RelayReason:  false,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseAgent.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct agent call. agent_id: %s", tt.responseAgent.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseAgent.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseAgent.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAgent_agentGetError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel
		hash    string

		responseDirect *dmdirect.Direct
	}{
		{
			name: "agent_get_error",

			channel: &channel.Channel{
				ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.600",
				DestinationNumber: "direct.a12b3c4d5e6f",
				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
			hash: "direct.a12b3c4d5e6f",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c2c2c2c2-c2c2-c2c2-c2c2-c2c2c2c2c2c2"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("4c0e953d-b67a-48f2-a1de-e303907628b5"),
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			})

			// AgentV1AgentGet returns error
			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseDirect.ResourceID).Return(nil, fmt.Errorf("agent not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIP_directAgent_flowCreateError(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel
		hash    string

		responseDirect *dmdirect.Direct
		responseAgent  *amagent.Agent
	}{
		{
			name: "flow_create_error",

			channel: &channel.Channel{
				ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.600",
				DestinationNumber: "direct.a12b3c4d5e6f",
				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
			hash: "direct.a12b3c4d5e6f",

			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c2c2c2c2-c2c2-c2c2-c2c2-c2c2c2c2c2c2"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("4c0e953d-b67a-48f2-a1de-e303907628b5"),
			},

			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4c0e953d-b67a-48f2-a1de-e303907628b5"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-agent",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000004",
			})

			// AgentV1AgentGet succeeds
			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseAgent, nil)

			// FlowV1FlowCreate returns error
			expectDestination := &commonaddress.Address{
				Type:       commonaddress.TypeAgent,
				Target:     tt.responseAgent.ID.String(),
				TargetName: tt.responseAgent.Name,
			}
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: fmaction.ConvertOption(fmaction.OptionConnect{
						Source: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000004",
						},
						Destinations: []commonaddress.Address{*expectDestination},
						EarlyMedia:   false,
						RelayReason:  false,
					}),
				},
			}
			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseAgent.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct agent call. agent_id: %s", tt.responseAgent.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(nil, fmt.Errorf("flow create failed"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNetworkOutOfOrder).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
