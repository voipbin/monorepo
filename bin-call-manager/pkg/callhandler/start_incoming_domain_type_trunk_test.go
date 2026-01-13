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
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	rmtrunk "monorepo/bin-registrar-manager/models/trunk"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_startIncomingDomainTypeTrunk(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseTrunk       *rmtrunk.Trunk
		responseFlow        *fmflow.Flow

		expectDomainName string
		expectCustomerID uuid.UUID
		expectAgentID    uuid.UUID
		expectActions    []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.171",

				DestinationName:   "",
				DestinationNumber: "+821100000001",
				SourceName:        "",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "test.trunk.voipbin.net",
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
			responseTrunk: &rmtrunk.Trunk{
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

			expectDomainName: "test",
			expectCustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			expectAgentID:    uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
			expectActions: []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: map[string]any{
						"source": map[string]any{
							"type":   "tel",
							"target": "+821100000002",
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

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel.Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestination(tt.channel, commonaddress.TypeTel.Return(tt.responseDestination)
			mockReq.EXPECT().RegistrarV1TrunkGetByDomainName(ctx, tt.expectDomainName.Return(tt.responseTrunk, nil)

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.expectCustomerID,
				fmflow.TypeFlow,
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				uuid.Nil,
				false,
			).DoAndReturn(func(
				_ context.Context,
				_ uuid.UUID,
				_ fmflow.Type,
				_ string,
				_ string,
				actions []fmaction.Action,
				_ uuid.UUID,
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
			mockUtil.EXPECT().UUIDCreate(.Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.expectCustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1.Return(true, nil)
			mockUtil.EXPECT().UUIDCreate(.Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any().Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any().Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeTrunk(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
