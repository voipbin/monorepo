package callhandler

import (
	"context"
	"fmt"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

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
			responseExtension: &rmextension.Extension{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					CustomerID: uuid.FromStringOrNil("138ca9fa-5e5f-11ed-a85f-9f66d5e00566"),
				},
				Extension:  "1001",
				DirectHash: "a3f8b2c1d4e5",
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
			mockReq.EXPECT().RegistrarV1ExtensionGetByDirectHash(ctx, "a3f8b2c1d4e5").Return(tt.responseExtension, nil)

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

func Test_startIncomingDomainTypeSIP_directExtension_hashNotFound(t *testing.T) {
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
			mockReq.EXPECT().RegistrarV1ExtensionGetByDirectHash(ctx, "invalidhash12").Return(nil, fmt.Errorf("not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
