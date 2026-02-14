package transferhandler

import (
	"context"
	"errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

func TestTransfererHangup_BlindType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &transferHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	tr := &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		Type: transfer.TypeBlind,
	}

	transfererCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		},
	}

	err := h.TransfererHangup(ctx, tr, transfererCall)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTransfererHangup_AttendedType_NoAnswer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &transferHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	tr := &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		Type:        transfer.TypeAttended,
		GroupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
	}

	transfererCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		},
		ConfbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
	}

	groupcall := &cmgroupcall.Groupcall{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
		},
		AnswerCallID: uuid.Nil,
	}

	confbridge := &cmconfbridge.Confbridge{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
		},
	}

	mockReq.EXPECT().CallV1GroupcallGet(ctx, tr.GroupcallID).Return(groupcall, nil)
	mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tr.ConfbridgeID).Return(confbridge, nil)

	err := h.TransfererHangup(ctx, tr, transfererCall)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTransfererHangup_AttendedType_WithAnswer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &transferHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	transfereeCallID := uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440003")

	tr := &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		Type:        transfer.TypeAttended,
		GroupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
	}

	transfererCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		},
		ConfbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
	}

	groupcall := &cmgroupcall.Groupcall{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
		},
		AnswerCallID: transfereeCallID,
	}

	confbridge := &cmconfbridge.Confbridge{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
		},
		ChannelCallIDs: map[string]uuid.UUID{
			"chan1": uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			"chan2": uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440003"),
		},
	}

	updatedTransfer := &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		Type:             transfer.TypeAttended,
		GroupcallID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
		TransfereeCallID: transfereeCallID,
	}

	mockReq.EXPECT().CallV1GroupcallGet(ctx, tr.GroupcallID).Return(groupcall, nil)

	// updateTransfereeCallID expectations
	mockDB.EXPECT().TransferGet(ctx, tr.ID).Return(tr, nil)
	mockDB.EXPECT().TransferUpdate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().TransferGet(ctx, tr.ID).Return(updatedTransfer, nil)

	// attendedUnblock expectations
	mockReq.EXPECT().CallV1ConfbridgeGet(ctx, transfererCall.ConfbridgeID).Return(confbridge, nil)
	for _, callID := range confbridge.ChannelCallIDs {
		if callID != transfererCall.ID {
			mockReq.EXPECT().CallV1CallMusicOnHoldOff(ctx, callID).Return(nil)
			mockReq.EXPECT().CallV1CallMuteOff(ctx, callID, cmcall.MuteDirectionIn).Return(nil)
		}
	}

	err := h.TransfererHangup(ctx, tr, transfererCall)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTransfererHangup_InvalidType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &transferHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	tr := &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		Type: transfer.Type("invalid"),
	}

	transfererCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		},
	}

	err := h.TransfererHangup(ctx, tr, transfererCall)
	if err == nil {
		t.Error("Expected error for invalid transfer type but got none")
	}
}

func TestTransfererHangup_AttendedType_GroupcallGetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &transferHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	tr := &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		},
		Type:        transfer.TypeAttended,
		GroupcallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
	}

	transfererCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		},
	}

	mockReq.EXPECT().CallV1GroupcallGet(ctx, tr.GroupcallID).Return(nil, errors.New("groupcall not found"))

	err := h.TransfererHangup(ctx, tr, transfererCall)
	if err == nil {
		t.Error("Expected error but got none")
	}
}
