package transferhandler

import (
	"context"
	"errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

func TestServiceStart_BlindTransfer(t *testing.T) {
	tests := []struct {
		name                string
		transferType        transfer.Type
		transfererCallID    uuid.UUID
		transfereeAddresses []commonaddress.Address
		transfererCall      *cmcall.Call
		flow                *fmflow.Flow
		groupcall           *cmgroupcall.Groupcall
		confbridge          *cmconfbridge.Confbridge
		callGetErr          error
		flowCreateErr       error
		confbridgeGetErr    error
		flagAddErr          error
		callHangupErr       error
		groupcallCreateErr  error
		shouldError         bool
	}{
		{
			name:             "blind_transfer_successfully",
			transferType:     transfer.TypeBlind,
			transfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			transfereeAddresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+821100000001"},
			},
			transfererCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				ConfbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000000",
				},
			},
			flow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
				},
			},
			confbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				},
			},
			groupcall: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
				},
			},
			shouldError: false,
		},
		{
			name:             "fails_when_call_get_fails",
			transferType:     transfer.TypeBlind,
			transfererCallID: uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440002"),
			transfereeAddresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+821100000001"},
			},
			callGetErr:  errors.New("call not found"),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.transfererCallID).Return(tt.transfererCall, tt.callGetErr)

			if tt.callGetErr == nil {
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.transfererCall.CustomerID, fmflow.TypeTransfer, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(tt.flow, tt.flowCreateErr)

				if tt.flowCreateErr == nil && tt.transferType == transfer.TypeBlind {
					mockReq.EXPECT().CallV1ConfbridgeFlagAdd(ctx, tt.transfererCall.ConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(tt.confbridge, tt.flagAddErr)

					if tt.flagAddErr == nil {
						mockReq.EXPECT().CallV1CallHangup(ctx, tt.transfererCall.ID).Return(tt.transfererCall, tt.callHangupErr)

						if tt.callHangupErr == nil {
							mockReq.EXPECT().CallV1ConfbridgeRing(ctx, tt.transfererCall.ConfbridgeID).Return(nil)
							mockReq.EXPECT().CallV1GroupcallCreate(ctx, uuid.Nil, tt.transfererCall.CustomerID, tt.flow.ID, tt.transfererCall.Source, tt.transfereeAddresses, gomock.Any(), uuid.Nil, cmgroupcall.RingMethodRingAll, cmgroupcall.AnswerMethodHangupOthers).Return(tt.groupcall, tt.groupcallCreateErr)

							if tt.groupcallCreateErr == nil {
								mockDB.EXPECT().TransferCreate(ctx, gomock.Any()).Return(nil)
							}
						}
					}
				}
			}

			result, err := h.ServiceStart(ctx, tt.transferType, tt.transfererCallID, tt.transfereeAddresses)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
			}
		})
	}
}

func TestServiceStart_AttendedTransfer(t *testing.T) {
	tests := []struct {
		name                string
		transferType        transfer.Type
		transfererCallID    uuid.UUID
		transfereeAddresses []commonaddress.Address
		transfererCall      *cmcall.Call
		flow                *fmflow.Flow
		groupcall           *cmgroupcall.Groupcall
		confbridge          *cmconfbridge.Confbridge
		callGetErr          error
		flowCreateErr       error
		confbridgeGetErr    error
		groupcallCreateErr  error
		shouldError         bool
	}{
		{
			name:             "attended_transfer_successfully",
			transferType:     transfer.TypeAttended,
			transfererCallID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			transfereeAddresses: []commonaddress.Address{
				{Type: commonaddress.TypeTel, Target: "+821100000001"},
			},
			transfererCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				ConfbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000000",
				},
			},
			flow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
				},
			},
			confbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				},
				ChannelCallIDs: map[string]uuid.UUID{
					"chan1": uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
					"chan2": uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440003"),
				},
			},
			groupcall: &cmgroupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockReq.EXPECT().CallV1CallGet(ctx, tt.transfererCallID).Return(tt.transfererCall, tt.callGetErr)

			if tt.callGetErr == nil {
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.transfererCall.CustomerID, fmflow.TypeTransfer, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(tt.flow, tt.flowCreateErr)

				if tt.flowCreateErr == nil && tt.transferType == transfer.TypeAttended {
					mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.transfererCall.ConfbridgeID).Return(tt.confbridge, tt.confbridgeGetErr)

					if tt.confbridgeGetErr == nil {
						// attendedBlock calls
						for _, callID := range tt.confbridge.ChannelCallIDs {
							if callID != tt.transfererCall.ID {
								mockReq.EXPECT().CallV1CallMusicOnHoldOn(ctx, callID).Return(nil)
								mockReq.EXPECT().CallV1CallMuteOn(ctx, callID, cmcall.MuteDirectionIn).Return(nil)
							}
						}

						// attendedExecute calls
						mockReq.EXPECT().CallV1GroupcallCreate(ctx, uuid.Nil, tt.transfererCall.CustomerID, tt.flow.ID, tt.transfererCall.Source, tt.transfereeAddresses, gomock.Any(), uuid.Nil, cmgroupcall.RingMethodRingAll, cmgroupcall.AnswerMethodHangupOthers).Return(tt.groupcall, tt.groupcallCreateErr)

						if tt.groupcallCreateErr == nil {
							mockDB.EXPECT().TransferCreate(ctx, gomock.Any()).Return(nil)
						}
					}
				}
			}

			result, err := h.ServiceStart(ctx, tt.transferType, tt.transfererCallID, tt.transfereeAddresses)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected transfer but got nil")
				}
			}
		})
	}
}

func TestServiceStart_InvalidType(t *testing.T) {
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

	transfererCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		},
		ConfbridgeID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
	}

	flow := &fmflow.Flow{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
		},
	}

	mockReq.EXPECT().CallV1CallGet(ctx, transfererCall.ID).Return(transfererCall, nil)
	mockReq.EXPECT().FlowV1FlowCreate(ctx, transfererCall.CustomerID, fmflow.TypeTransfer, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(flow, nil)

	_, err := h.ServiceStart(ctx, transfer.Type("invalid"), transfererCall.ID, []commonaddress.Address{})

	if err == nil {
		t.Error("Expected error for invalid transfer type but got none")
	}
}

func TestNewTransferHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewTransferHandler(mockReq, mockNotify, mockDB)

	if h == nil {
		t.Error("Expected handler but got nil")
	}
}
