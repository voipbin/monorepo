package transferhandler

import (
	"context"
	reflect "reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

func Test_blindExecute(t *testing.T) {

	tests := []struct {
		name string

		transfererCall      *cmcall.Call
		flow                *fmflow.Flow
		transfereeAddresses []commonaddress.Address

		responseGroupcall    *cmgroupcall.Groupcall
		responseTransfer     *transfer.Transfer
		responseUUIDTransfer uuid.UUID

		expectTransfer *transfer.Transfer
	}{
		{
			name: "normal",

			transfererCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("884eeae2-dbb3-11ed-bf74-8f39182ac412"),
					CustomerID: uuid.FromStringOrNil("87a36756-dbb5-11ed-8ab5-0b2b7aab415b"),
				},
				ConfbridgeID: uuid.FromStringOrNil("51db01e6-dbb6-11ed-8901-e3139187e083"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				MasterCallID: uuid.FromStringOrNil("89447e1c-dbb3-11ed-be21-6fc0d5041f86"),
			},
			flow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("88990fc8-dbb3-11ed-ba6d-b7d235536fac"),
				},
				Actions: []fmaction.Action{},
			},
			transfereeAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},

			responseGroupcall: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8906ea48-dbb3-11ed-9680-eba11737a7bb"),
				},
			},
			responseTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("89b9c942-dbb3-11ed-8784-67bdc93fd8af"),
				},
			},
			responseUUIDTransfer: uuid.FromStringOrNil("89b9c942-dbb3-11ed-8784-67bdc93fd8af"),

			expectTransfer: &transfer.Transfer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("89b9c942-dbb3-11ed-8784-67bdc93fd8af"),
					CustomerID: uuid.FromStringOrNil("87a36756-dbb5-11ed-8ab5-0b2b7aab415b"),
				},
				Type:             transfer.TypeBlind,
				TransfererCallID: uuid.FromStringOrNil("884eeae2-dbb3-11ed-bf74-8f39182ac412"),
				TransfereeAddresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				GroupcallID:  uuid.FromStringOrNil("8906ea48-dbb3-11ed-9680-eba11737a7bb"),
				ConfbridgeID: uuid.FromStringOrNil("51db01e6-dbb6-11ed-8901-e3139187e083"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallHangup(ctx, tt.transfererCall.ID).Return(&cmcall.Call{}, nil)
			mockReq.EXPECT().CallV1ConfbridgeRing(ctx, tt.transfererCall.ConfbridgeID).Return(nil)
			mockReq.EXPECT().CallV1GroupcallCreate(
				ctx,
				uuid.Nil,
				tt.transfererCall.CustomerID,
				tt.flow.ID,
				tt.transfererCall.Source,
				tt.transfereeAddresses,
				tt.transfererCall.MasterCallID,
				uuid.Nil,
				cmgroupcall.RingMethodRingAll,
				cmgroupcall.AnswerMethodHangupOthers,
			).Return(tt.responseGroupcall, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDTransfer)
			mockDB.EXPECT().TransferCreate(ctx, tt.expectTransfer).Return(nil)

			res, err := h.blindExecute(ctx, tt.transfererCall, tt.flow, tt.transfereeAddresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectTransfer) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectTransfer, res)
			}
		})
	}
}

func Test_blindBlock(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID
	}{
		{
			name: "normal",

			confbridgeID: uuid.FromStringOrNil("83450158-dd11-11ed-9afb-c774196230ee"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeFlagAdd(ctx, tt.confbridgeID, cmconfbridge.FlagNoAutoLeave).Return(&cmconfbridge.Confbridge{}, nil)

			if err := h.blindBlock(ctx, tt.confbridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_blindUnblock(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID
	}{
		{
			name: "normal",

			confbridgeID: uuid.FromStringOrNil("d71b88f6-dd11-11ed-b997-c7acb2c6b00c"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, tt.confbridgeID, cmconfbridge.FlagNoAutoLeave).Return(&cmconfbridge.Confbridge{}, nil)

			if err := h.blindUnblock(ctx, tt.confbridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
