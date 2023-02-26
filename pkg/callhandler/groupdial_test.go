package callhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	rmastcontact "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_createGroupDial(t *testing.T) {

	tests := []struct {
		name string

		customerID                uuid.UUID
		flowID                    uuid.UUID
		masterCallID              uuid.UUID
		source                    *commonaddress.Address
		destination               *commonaddress.Address
		earlyExecution            bool
		executeNextMasterOnHangup bool

		responseExntension *extension.Extension
		responseContacts   []*rmastcontact.AstContact
		responseUUID       uuid.UUID
		responseCalls      []*call.Call

		expectGroupDial *groupdial.GroupDial
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			flowID:       uuid.FromStringOrNil("e9ebb18c-b5c4-11ed-9775-cf1b5f3ac127"),
			masterCallID: uuid.Nil,
			source: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destination: &commonaddress.Address{
				Type:   commonaddress.TypeEndpoint,
				Target: "test-exten@test-domain",
			},
			earlyExecution:            false,
			executeNextMasterOnHangup: false,

			responseExntension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("ea46f164-b5c4-11ed-823e-47dfafd09c7b"),
				CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			},
			responseContacts: []*rmastcontact.AstContact{
				{
					URI: "sip:test-exten1@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
				{
					URI: "sip:test-exten2@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
			},
			responseUUID: uuid.FromStringOrNil("08701bca-b5e8-11ed-9257-4bee6cbc72bf"),
			responseCalls: []*call.Call{
				{
					ID: uuid.FromStringOrNil("a62ac2ae-b5eb-11ed-9607-fff199830675"),
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
						Target: "sip:test-exten1@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
					},
				},
				{
					ID: uuid.FromStringOrNil("a65415be-b5eb-11ed-86f5-83763d30dbf1"),
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeSIP,
						Target: "sip:test-exten2@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
					},
				},
			},

			expectGroupDial: &groupdial.GroupDial{
				ID:         uuid.FromStringOrNil("08701bca-b5e8-11ed-9257-4bee6cbc72bf"),
				CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test-exten@test-domain",
				},
				RingMethod:   groupdial.RingMethodRingAll,
				AnswerMethod: groupdial.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("a62ac2ae-b5eb-11ed-9607-fff199830675"),
					uuid.FromStringOrNil("a65415be-b5eb-11ed-86f5-83763d30dbf1"),
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

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			// getDestinationsAddressTypeEndpoint
			mockReq.EXPECT().RegistrarV1ExtensionGetByEndpoint(ctx, tt.destination.Target).Return(tt.responseExntension, nil)
			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.destination.Target).Return(tt.responseContacts, nil)

			for i := range tt.responseContacts {
				// CreateCallOutgoing
				mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
				mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, gomock.Any(), tt.flowID, fmactiveflow.ReferenceTypeCall, gomock.Any()).Return(&fmactiveflow.Activeflow{}, nil)

				mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
				mockDB.EXPECT().CallCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(tt.responseCalls[i], nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallCreated, gomock.Any())
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)

				mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, tt.responseCalls[i].ChannelID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)
			}
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().GroupDialCreate(ctx, tt.expectGroupDial).Return(nil)

			res, err := h.createGroupDial(ctx, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination, tt.earlyExecution, tt.executeNextMasterOnHangup)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectGroupDial) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectGroupDial, res)
			}
		})
	}
}

func Test_getDestinationsAddressTypeEndpoint(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination *commonaddress.Address

		responseExntension *rmextension.Extension
		responseContacts   []*rmastcontact.AstContact

		expectRes []*commonaddress.Address
	}{
		{
			name: "1 contact",

			customerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			destination: &commonaddress.Address{
				Type:       commonaddress.TypeEndpoint,
				Target:     "test-exten@test-domain",
				TargetName: "test extension",
			},

			responseExntension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("382bee00-b5ef-11ed-a3d9-f7ee8324e815"),
				CustomerID: uuid.FromStringOrNil("38007676-b5ef-11ed-a920-dfb6f25329d5"),
			},
			responseContacts: []*rmastcontact.AstContact{
				{
					URI: "sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
				},
			},

			expectRes: []*commonaddress.Address{
				{
					Type:       commonaddress.TypeSIP,
					TargetName: "test extension",
					Target:     "sip:test11@211.178.226.108:35551;transport=UDP;rinstance=8a1f981a77f30a22",
				},
			},
		},
		{
			name: "2 contacts",

			customerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			destination: &commonaddress.Address{
				Type:       commonaddress.TypeEndpoint,
				Target:     "test-exten@test-domain",
				TargetName: "test extension",
			},

			responseExntension: &rmextension.Extension{
				ID:         uuid.FromStringOrNil("ea46f164-b5c4-11ed-823e-47dfafd09c7b"),
				CustomerID: uuid.FromStringOrNil("e9a6c252-b5c4-11ed-8431-0f528880d39a"),
			},
			responseContacts: []*rmastcontact.AstContact{
				{
					URI: "sip:test-exten1@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
				{
					URI: "sip:test-exten2@211.200.20.28:53941^3Btransport=udp^3Balias=211.200.20.28~53941~1",
				},
			},

			expectRes: []*commonaddress.Address{
				{
					Type:       commonaddress.TypeSIP,
					TargetName: "test extension",
					Target:     "sip:test-exten1@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
				},
				{
					Type:       commonaddress.TypeSIP,
					TargetName: "test extension",
					Target:     "sip:test-exten2@211.200.20.28:53941;transport=udp;alias=211.200.20.28~53941~1",
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGetByEndpoint(ctx, tt.destination.Target).Return(tt.responseExntension, nil)
			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.destination.Target).Return(tt.responseContacts, nil)

			res, err := h.getDestinationsAddressTypeEndpoint(ctx, tt.customerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
