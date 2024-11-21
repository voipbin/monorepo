package transferhandler

import (
	"context"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

func Test_TransfereeHangup_type_blind(t *testing.T) {

	tests := []struct {
		name string

		tr *transfer.Transfer
		gc *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			tr: &transfer.Transfer{
				Type:         transfer.TypeBlind,
				ConfbridgeID: uuid.FromStringOrNil("f2492f48-dbae-11ed-8d78-db8ab88a4c38"),
			},
			gc: &cmgroupcall.Groupcall{},
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

			mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.tr.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)

			if err := h.TransfereeHangup(ctx, tt.tr, tt.gc); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_transfereeHangupTypeAttended(t *testing.T) {

	tests := []struct {
		name string

		tr *transfer.Transfer
		gc *cmgroupcall.Groupcall

		responseTransfererCall *cmcall.Call
		responseConfbridge     *cmconfbridge.Confbridge
	}{
		{
			name: "normal",

			tr: &transfer.Transfer{
				Type:             transfer.TypeBlind,
				ConfbridgeID:     uuid.FromStringOrNil("12ea4df8-dd13-11ed-a2d0-d78513698076"),
				TransfererCallID: uuid.FromStringOrNil("131e8226-dd13-11ed-b587-fbf6f7f2d5fa"),
			},
			gc: &cmgroupcall.Groupcall{},

			responseTransfererCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("131e8226-dd13-11ed-b587-fbf6f7f2d5fa"),
				},
				ConfbridgeID: uuid.FromStringOrNil("12ea4df8-dd13-11ed-a2d0-d78513698076"),
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("12ea4df8-dd13-11ed-a2d0-d78513698076"),
				ChannelCallIDs: map[string]uuid.UUID{
					"1349b00e-dd13-11ed-a3c0-ef2fe7037b66": uuid.FromStringOrNil("131e8226-dd13-11ed-b587-fbf6f7f2d5fa"),
					"13702536-dd13-11ed-bfc0-3bef698ff6ae": uuid.FromStringOrNil("139ce378-dd13-11ed-aacf-3b12ccca6a22"),
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.tr.TransfererCallID).Return(tt.responseTransfererCall, nil)
			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.responseTransfererCall.ConfbridgeID).Return(tt.responseConfbridge, nil)
			for _, callID := range tt.responseConfbridge.ChannelCallIDs {
				if callID == tt.responseTransfererCall.ID {
					continue
				}
				mockReq.EXPECT().CallV1CallMusicOnHoldOff(ctx, callID).Return(nil)
				mockReq.EXPECT().CallV1CallMuteOff(ctx, callID, cmcall.MuteDirectionIn).Return(nil)
			}

			if err := h.transfereeHangupTypeAttended(ctx, tt.tr, tt.gc); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_TransfereeAnswer_TypeBlind(t *testing.T) {

	tests := []struct {
		name string

		tr *transfer.Transfer
		gc *cmgroupcall.Groupcall

		expectTransfer *transfer.Transfer
	}{
		{
			name: "normal",

			tr: &transfer.Transfer{
				ID:           uuid.FromStringOrNil("b22de150-dbaf-11ed-84f8-9f0eb78f20bf"),
				Type:         transfer.TypeBlind,
				ConfbridgeID: uuid.FromStringOrNil("4e2b6cc2-dbaf-11ed-b0ac-8ff64c6f7a2d"),
			},
			gc: &cmgroupcall.Groupcall{
				AnswerCallID: uuid.FromStringOrNil("4e5527ec-dbaf-11ed-a26e-5b3c8272a6e5"),
			},

			expectTransfer: &transfer.Transfer{
				ID:               uuid.FromStringOrNil("b22de150-dbaf-11ed-84f8-9f0eb78f20bf"),
				Type:             transfer.TypeBlind,
				ConfbridgeID:     uuid.FromStringOrNil("4e2b6cc2-dbaf-11ed-b0ac-8ff64c6f7a2d"),
				TransfereeCallID: uuid.FromStringOrNil("4e5527ec-dbaf-11ed-a26e-5b3c8272a6e5"),
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

			mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, tt.tr.ConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(&cmconfbridge.Confbridge{}, nil)
			mockDB.EXPECT().TransferGet(ctx, tt.tr.ID).Return(tt.tr, nil)
			mockDB.EXPECT().TransferUpdate(ctx, tt.expectTransfer).Return(nil)
			mockDB.EXPECT().TransferGet(ctx, tt.tr.ID).Return(tt.tr, nil)
			mockReq.EXPECT().CallV1ConfbridgeAnswer(ctx, tt.tr.ConfbridgeID).Return(nil)

			if err := h.TransfereeAnswer(ctx, tt.tr, tt.gc); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
