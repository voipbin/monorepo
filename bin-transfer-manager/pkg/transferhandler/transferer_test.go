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

func Test_transfererHangupTypeAttended(t *testing.T) {

	tests := []struct {
		name string

		tr             *transfer.Transfer
		transfererCall *cmcall.Call

		responseGroupcall  *cmgroupcall.Groupcall
		responseConfbridge *cmconfbridge.Confbridge

		expectTransfer *transfer.Transfer
	}{
		{
			name: "normal",

			tr: &transfer.Transfer{
				ID:           uuid.FromStringOrNil("7374ed3a-dd14-11ed-9281-cf586a573929"),
				Type:         transfer.TypeAttended,
				ConfbridgeID: uuid.FromStringOrNil("73abc206-dd14-11ed-82f4-c3a0208de1a0"),
			},
			transfererCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73d8d282-dd14-11ed-bcb5-3fbb56ae81cb"),
				},
				ConfbridgeID: uuid.FromStringOrNil("73abc206-dd14-11ed-82f4-c3a0208de1a0"),
			},

			responseGroupcall: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("74032dac-dd14-11ed-9a51-bfa35d33c8c3"),
				},
				AnswerCallID: uuid.FromStringOrNil("742f5fb2-dd14-11ed-a171-3f5bdaf8117d"),
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("73abc206-dd14-11ed-82f4-c3a0208de1a0"),
				ChannelCallIDs: map[string]uuid.UUID{
					"7767f166-dd15-11ed-b14a-6397f82a508f": uuid.FromStringOrNil("73d8d282-dd14-11ed-bcb5-3fbb56ae81cb"),
					"7792f3de-dd15-11ed-b6c7-476bb2c26621": uuid.FromStringOrNil("77bc1cbe-dd15-11ed-b947-8b03dcbc1956"),
				},
			},
			expectTransfer: &transfer.Transfer{
				ID:               uuid.FromStringOrNil("7374ed3a-dd14-11ed-9281-cf586a573929"),
				Type:             transfer.TypeAttended,
				ConfbridgeID:     uuid.FromStringOrNil("73abc206-dd14-11ed-82f4-c3a0208de1a0"),
				TransfereeCallID: uuid.FromStringOrNil("742f5fb2-dd14-11ed-a171-3f5bdaf8117d"),
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

			mockReq.EXPECT().CallV1GroupcallGet(ctx, tt.tr.GroupcallID).Return(tt.responseGroupcall, nil)
			mockDB.EXPECT().TransferGet(ctx, tt.tr.ID).Return(tt.tr, nil)
			mockDB.EXPECT().TransferUpdate(ctx, tt.expectTransfer).Return(nil)
			mockDB.EXPECT().TransferGet(ctx, tt.tr.ID).Return(tt.expectTransfer, nil)

			mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.expectTransfer.ConfbridgeID).Return(tt.responseConfbridge, nil)
			for _, callID := range tt.responseConfbridge.ChannelCallIDs {
				if callID == tt.transfererCall.ID {
					continue
				}
				mockReq.EXPECT().CallV1CallMusicOnHoldOff(ctx, callID).Return(nil)
				mockReq.EXPECT().CallV1CallMuteOff(ctx, callID, cmcall.MuteDirectionIn).Return(nil)
			}

			if err := h.transfererHangupTypeAttended(ctx, tt.tr, tt.transfererCall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
