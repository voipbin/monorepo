package transferhandler

import (
	"context"
	reflect "reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

func Test_attendedInit(t *testing.T) {

	tests := []struct {
		name string

		transfererCall *cmcall.Call

		responseConfbridge *cmconfbridge.Confbridge
	}{
		{
			name: "normal",

			transfererCall: &cmcall.Call{
				ID:           uuid.FromStringOrNil("1ee4b104-dbb8-11ed-aca8-a3870b5b6fec"),
				ConfbridgeID: uuid.FromStringOrNil("1f268336-dbb8-11ed-9618-db090c5e42d6"),
			},

			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("1f268336-dbb8-11ed-9618-db090c5e42d6"),
				ChannelCallIDs: map[string]uuid.UUID{
					"7e2b352a-dbb8-11ed-b0a2-bfac6acb7193": uuid.FromStringOrNil("1ee4b104-dbb8-11ed-aca8-a3870b5b6fec"),
					"7e5607fa-dbb8-11ed-a821-674a49015505": uuid.FromStringOrNil("7e7f5128-dbb8-11ed-82e6-bb32d9c540fe"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.transfererCall.ConfbridgeID).Return(tt.responseConfbridge, nil)

			for _, callID := range tt.responseConfbridge.ChannelCallIDs {
				if callID == tt.transfererCall.ID {
					continue
				}
				mockReq.EXPECT().CallV1CallMusicOnHoldOn(ctx, callID).Return(nil)
				mockReq.EXPECT().CallV1CallMuteOn(ctx, callID, cmcall.MuteDirectionIn).Return(nil)
			}

			if err := h.attendedBlock(ctx, tt.transfererCall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_attendedCancel(t *testing.T) {

	tests := []struct {
		name string

		transfererCall *cmcall.Call

		responseConfbridge *cmconfbridge.Confbridge
	}{
		{
			name: "normal",

			transfererCall: &cmcall.Call{
				ID:           uuid.FromStringOrNil("d96aaab0-dc44-11ed-8b23-2bc7cc1b025a"),
				ConfbridgeID: uuid.FromStringOrNil("d9c6377c-dc44-11ed-a6ac-7f9a18bc4e86"),
			},

			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("d9c6377c-dc44-11ed-a6ac-7f9a18bc4e86"),
				ChannelCallIDs: map[string]uuid.UUID{
					"d9c6377c-dc44-11ed-a6ac-7f9a18bc4e86": uuid.FromStringOrNil("6669d858-dc47-11ed-9246-335a3dd63ec8"),
					"66878c7c-dc47-11ed-aa6e-d71b35ef69c3": uuid.FromStringOrNil("66ad2fb8-dc47-11ed-974e-dbfd996258ad"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := transferHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.transfererCall.ConfbridgeID).Return(tt.responseConfbridge, nil)

			for _, callID := range tt.responseConfbridge.ChannelCallIDs {
				if callID == tt.transfererCall.ID {
					continue
				}
				mockReq.EXPECT().CallV1CallMusicOnHoldOff(ctx, callID).Return(nil)
				mockReq.EXPECT().CallV1CallMuteOff(ctx, callID, cmcall.MuteDirectionIn).Return(nil)
			}

			if err := h.attendedUnblock(ctx, tt.transfererCall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_attendedExecute(t *testing.T) {

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
				ID:           uuid.FromStringOrNil("64243052-dc6f-11ed-ac22-e7d20e32e435"),
				CustomerID:   uuid.FromStringOrNil("64631858-dc6f-11ed-97c9-c74c6afbd067"),
				ConfbridgeID: uuid.FromStringOrNil("64912a40-dc6f-11ed-bc6c-f3d9bb503ba5"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				MasterCallID: uuid.FromStringOrNil("64bc53aa-dc6f-11ed-9704-03ecd7a766fd"),
			},
			flow: &fmflow.Flow{
				ID:      uuid.FromStringOrNil("64e33d80-dc6f-11ed-a108-0b5c52dbe645"),
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
				ID: uuid.FromStringOrNil("a0878814-dc6f-11ed-8711-333f46f48590"),
			},
			responseTransfer: &transfer.Transfer{
				ID: uuid.FromStringOrNil("a0b45cb8-dc6f-11ed-98ba-177bdc1ba9c2"),
			},
			responseUUIDTransfer: uuid.FromStringOrNil("a0b45cb8-dc6f-11ed-98ba-177bdc1ba9c2"),

			expectTransfer: &transfer.Transfer{
				ID:               uuid.FromStringOrNil("a0b45cb8-dc6f-11ed-98ba-177bdc1ba9c2"),
				CustomerID:       uuid.FromStringOrNil("64631858-dc6f-11ed-97c9-c74c6afbd067"),
				Type:             transfer.TypeAttended,
				TransfererCallID: uuid.FromStringOrNil("64243052-dc6f-11ed-ac22-e7d20e32e435"),
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
				GroupcallID:  uuid.FromStringOrNil("a0878814-dc6f-11ed-8711-333f46f48590"),
				ConfbridgeID: uuid.FromStringOrNil("64912a40-dc6f-11ed-bc6c-f3d9bb503ba5"),
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

			res, err := h.attendedExecute(ctx, tt.transfererCall, tt.flow, tt.transfereeAddresses)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectTransfer) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectTransfer, res)
			}
		})
	}
}
