package transferhandler

import (
	"context"
	stderrors "errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/dbhandler"
)

var (
	epTransfererCallID = uuid.FromStringOrNil("c0a80001-0001-11ee-9d01-000000000001")
	epOtherCallID      = uuid.FromStringOrNil("c0a80001-0002-11ee-9d01-000000000002")
	epOtherCallID2     = uuid.FromStringOrNil("c0a80001-0003-11ee-9d01-000000000003")
	epConfbridgeID     = uuid.FromStringOrNil("c0a80001-0004-11ee-9d01-000000000004")
	epCustomerID       = uuid.FromStringOrNil("c0a80001-0005-11ee-9d01-000000000005")
	epGroupcallID      = uuid.FromStringOrNil("c0a80001-0006-11ee-9d01-000000000006")
	epTransferID       = uuid.FromStringOrNil("c0a80001-0007-11ee-9d01-000000000007")
	epFlowID           = uuid.FromStringOrNil("c0a80001-0008-11ee-9d01-000000000008")

	errEP = stderrors.New("injected failure")
)

type epEnv struct {
	h       *transferHandler
	mockReq *requesthandler.MockRequestHandler
	mockDB  *dbhandler.MockDBHandler
	mockUtl *utilhandler.MockUtilHandler
}

func newEPEnv(t *testing.T) *epEnv {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	e := &epEnv{
		mockReq: requesthandler.NewMockRequestHandler(mc),
		mockDB:  dbhandler.NewMockDBHandler(mc),
		mockUtl: utilhandler.NewMockUtilHandler(mc),
	}
	e.h = &transferHandler{
		utilHandler: e.mockUtl,
		reqHandler:  e.mockReq,
		db:          e.mockDB,
	}
	return e
}

func epTransfererCall() *cmcall.Call {
	return &cmcall.Call{
		Identity: commonidentity.Identity{
			ID:         epTransfererCallID,
			CustomerID: epCustomerID,
		},
		ConfbridgeID: epConfbridgeID,
	}
}

func epConfbridge(otherIDs ...uuid.UUID) *cmconfbridge.Confbridge {
	ids := map[string]uuid.UUID{"transferer": epTransfererCallID}
	for i, id := range otherIDs {
		ids[string(rune('a'+i))] = id
	}
	return &cmconfbridge.Confbridge{
		Identity:       commonidentity.Identity{ID: epConfbridgeID},
		ChannelCallIDs: ids,
	}
}

func epFlow() *fmflow.Flow {
	return &fmflow.Flow{Identity: commonidentity.Identity{ID: epFlowID}}
}

func epAddresses() []commonaddress.Address {
	return []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550100"}}
}

// expectGroupcallCreate expects the groupcall creation of the execute step.
func (e *epEnv) expectGroupcallCreate(ctx context.Context, res *cmgroupcall.Groupcall, err error) *gomock.Call {
	return e.mockReq.EXPECT().CallV1GroupcallCreate(
		ctx,
		uuid.Nil,
		epCustomerID,
		epFlowID,
		gomock.Any(),
		epAddresses(),
		uuid.Nil,
		uuid.Nil,
		cmgroupcall.RingMethodRingAll,
		cmgroupcall.AnswerMethodHangupOthers,
		"",
		nil,
	).Return(res, err)
}

func Test_AttendedTransfer_blockFails(t *testing.T) {
	e := newEPEnv(t)
	ctx := context.Background()

	// nothing but the confbridge lookup is expected: no hold, no groupcall, no rollback.
	e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(nil, errEP)

	res, err := e.h.AttendedTransfer(ctx, epTransfererCall(), epFlow(), epAddresses())
	if !stderrors.Is(err, errEP) {
		t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_AttendedTransfer_executeFailsRollsBack(t *testing.T) {
	tests := []struct {
		name string

		// expectExecute sets the expectation of the execute step that fails.
		expectExecute func(e *epEnv, ctx context.Context)
	}{
		{
			name: "groupcall create fails",
			expectExecute: func(e *epEnv, ctx context.Context) {
				e.expectGroupcallCreate(ctx, nil, errEP)
			},
		},
		{
			name: "transfer create fails",
			expectExecute: func(e *epEnv, ctx context.Context) {
				e.expectGroupcallCreate(ctx, &cmgroupcall.Groupcall{Identity: commonidentity.Identity{ID: epGroupcallID}}, nil)
				e.mockUtl.EXPECT().UUIDCreate().Return(epTransferID)
				e.mockDB.EXPECT().TransferCreate(ctx, gomock.Any()).Return(errEP)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEPEnv(t)
			ctx := context.Background()
			cb := epConfbridge(epOtherCallID)

			// block (hold and mute the other party) and, after the failure, the rollback
			// (release both) on exactly the other party.
			e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(cb, nil).Times(2)
			e.mockReq.EXPECT().CallV1CallMusicOnHoldOn(ctx, epOtherCallID).Return(nil)
			e.mockReq.EXPECT().CallV1CallMuteOn(ctx, epOtherCallID, cmcall.MuteDirectionIn).Return(nil)
			tt.expectExecute(e, ctx)
			e.mockReq.EXPECT().CallV1CallMusicOnHoldOff(ctx, epOtherCallID).Return(nil)
			e.mockReq.EXPECT().CallV1CallMuteOff(ctx, epOtherCallID, cmcall.MuteDirectionIn).Return(nil)

			res, err := e.h.AttendedTransfer(ctx, epTransfererCall(), epFlow(), epAddresses())
			if !stderrors.Is(err, errEP) {
				t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_AttendedTransfer_rollbackFailureKeepsOriginalError(t *testing.T) {
	e := newEPEnv(t)
	ctx := context.Background()

	// the rollback itself can not read the confbridge: the execute error is still returned.
	gomock.InOrder(
		e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(epConfbridge(), nil),
		e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(nil, stderrors.New("rollback lookup failed")),
	)
	execErr := stderrors.New("groupcall failed")
	e.expectGroupcallCreate(ctx, nil, execErr)

	_, err := e.h.AttendedTransfer(ctx, epTransfererCall(), epFlow(), epAddresses())
	if !stderrors.Is(err, execErr) {
		t.Errorf("Wrong match. expect: wrapped %v, got: %v", execErr, err)
	}
}

func Test_attendedBlockUnblock_holdFailuresAreTolerated(t *testing.T) {
	e := newEPEnv(t)
	ctx := context.Background()
	cb := epConfbridge(epOtherCallID, epOtherCallID2)

	// every hold and mute call fails, yet both parties are still attempted and no error is
	// returned. The transferer itself is never touched (it has no expectation).
	e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(cb, nil).Times(2)
	for _, id := range []uuid.UUID{epOtherCallID, epOtherCallID2} {
		e.mockReq.EXPECT().CallV1CallMusicOnHoldOn(ctx, id).Return(errEP)
		e.mockReq.EXPECT().CallV1CallMuteOn(ctx, id, cmcall.MuteDirectionIn).Return(errEP)
		e.mockReq.EXPECT().CallV1CallMusicOnHoldOff(ctx, id).Return(errEP)
		e.mockReq.EXPECT().CallV1CallMuteOff(ctx, id, cmcall.MuteDirectionIn).Return(errEP)
	}

	if err := e.h.attendedBlock(ctx, epTransfererCall()); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := e.h.attendedUnblock(ctx, epTransfererCall()); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_attendedUnblock_confbridgeGetFails(t *testing.T) {
	e := newEPEnv(t)
	ctx := context.Background()

	e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(nil, errEP)
	if err := e.h.attendedUnblock(ctx, epTransfererCall()); !stderrors.Is(err, errEP) {
		t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
	}
}

func Test_BlindTransfer_errors(t *testing.T) {
	t.Run("block fails", func(t *testing.T) {
		e := newEPEnv(t)
		ctx := context.Background()

		// no hangup, no groupcall and no rollback are expected.
		e.mockReq.EXPECT().CallV1ConfbridgeFlagAdd(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(nil, errEP)

		res, err := e.h.BlindTransfer(ctx, epTransfererCall(), epFlow(), epAddresses())
		if !stderrors.Is(err, errEP) || res != nil {
			t.Errorf("Wrong match. expect: wrapped %v and nil, got: %v, %v", errEP, err, res)
		}
	})

	tests := []struct {
		name          string
		expectExecute func(e *epEnv, ctx context.Context)
		unblockErr    error
	}{
		{
			name: "hangup fails",
			expectExecute: func(e *epEnv, ctx context.Context) {
				e.mockReq.EXPECT().CallV1CallHangup(ctx, epTransfererCallID).Return(nil, errEP)
			},
		},
		{
			name: "groupcall create fails and ring failure is tolerated",
			expectExecute: func(e *epEnv, ctx context.Context) {
				e.mockReq.EXPECT().CallV1CallHangup(ctx, epTransfererCallID).Return(&cmcall.Call{}, nil)
				e.mockReq.EXPECT().CallV1ConfbridgeRing(ctx, epConfbridgeID).Return(stderrors.New("ring failed"))
				e.expectGroupcallCreate(ctx, nil, errEP)
			},
		},
		{
			name: "transfer create fails",
			expectExecute: func(e *epEnv, ctx context.Context) {
				e.mockReq.EXPECT().CallV1CallHangup(ctx, epTransfererCallID).Return(&cmcall.Call{}, nil)
				e.mockReq.EXPECT().CallV1ConfbridgeRing(ctx, epConfbridgeID).Return(nil)
				e.expectGroupcallCreate(ctx, &cmgroupcall.Groupcall{Identity: commonidentity.Identity{ID: epGroupcallID}}, nil)
				e.mockUtl.EXPECT().UUIDCreate().Return(epTransferID)
				e.mockDB.EXPECT().TransferCreate(ctx, gomock.Any()).Return(errEP)
			},
		},
		{
			name: "rollback failure keeps the original error",
			expectExecute: func(e *epEnv, ctx context.Context) {
				e.mockReq.EXPECT().CallV1CallHangup(ctx, epTransfererCallID).Return(nil, errEP)
			},
			unblockErr: stderrors.New("unblock failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEPEnv(t)
			ctx := context.Background()

			e.mockReq.EXPECT().CallV1ConfbridgeFlagAdd(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(&cmconfbridge.Confbridge{}, nil)
			tt.expectExecute(e, ctx)
			// the rollback must clear the no-auto-leave flag on the same confbridge.
			if tt.unblockErr != nil {
				e.mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(nil, tt.unblockErr)
			} else {
				e.mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(&cmconfbridge.Confbridge{}, nil)
			}

			res, err := e.h.BlindTransfer(ctx, epTransfererCall(), epFlow(), epAddresses())
			if !stderrors.Is(err, errEP) || res != nil {
				t.Errorf("Wrong match. expect: wrapped %v and nil, got: %v, %v", errEP, err, res)
			}
		})
	}
}

func Test_blindUnblock_removeFlagFails(t *testing.T) {
	e := newEPEnv(t)
	ctx := context.Background()

	e.mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(nil, errEP)
	if err := e.h.blindUnblock(ctx, epConfbridgeID); !stderrors.Is(err, errEP) {
		t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
	}
}

func Test_Transfer_unsupportedType(t *testing.T) {
	tr := &transfer.Transfer{Type: transfer.Type("unknown")}

	e := newEPEnv(t)
	ctx := context.Background()

	if err := e.h.TransfereeHangup(ctx, tr, &cmgroupcall.Groupcall{}); err == nil {
		t.Errorf("Wrong match. TransfereeHangup expect: error, got: ok")
	}
	if err := e.h.TransfereeAnswer(ctx, tr, &cmgroupcall.Groupcall{}); err == nil {
		t.Errorf("Wrong match. TransfereeAnswer expect: error, got: ok")
	}
}

func Test_TransfereeHangup_failures(t *testing.T) {
	ctx := context.Background()
	gc := &cmgroupcall.Groupcall{}

	t.Run("blind with terminate failure is tolerated", func(t *testing.T) {
		e := newEPEnv(t)
		tr := &transfer.Transfer{Type: transfer.TypeBlind, ConfbridgeID: epConfbridgeID}

		e.mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, epConfbridgeID).Return(nil, errEP)
		if err := e.h.TransfereeHangup(ctx, tr, gc); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("answered transferee leaves everything alone", func(t *testing.T) {
		for _, typ := range []transfer.Type{transfer.TypeBlind, transfer.TypeAttended} {
			e := newEPEnv(t)
			tr := &transfer.Transfer{Type: typ, TransfereeCallID: epOtherCallID}

			// no request is expected at all.
			if err := e.h.TransfereeHangup(ctx, tr, gc); err != nil {
				t.Errorf("Wrong match. type: %s, expect: ok, got: %v", typ, err)
			}
		}
	})

	t.Run("attended transferer call lookup fails", func(t *testing.T) {
		e := newEPEnv(t)
		tr := &transfer.Transfer{Type: transfer.TypeAttended, TransfererCallID: epTransfererCallID}

		e.mockReq.EXPECT().CallV1CallGet(ctx, epTransfererCallID).Return(nil, errEP)
		if err := e.h.TransfereeHangup(ctx, tr, gc); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})

	t.Run("attended unblock fails", func(t *testing.T) {
		e := newEPEnv(t)
		tr := &transfer.Transfer{Type: transfer.TypeAttended, TransfererCallID: epTransfererCallID}

		e.mockReq.EXPECT().CallV1CallGet(ctx, epTransfererCallID).Return(epTransfererCall(), nil)
		e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(nil, errEP)
		if err := e.h.TransfereeHangup(ctx, tr, gc); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})
}

func Test_TransfereeAnswer_blindFailures(t *testing.T) {
	ctx := context.Background()
	tr := &transfer.Transfer{
		Identity:     commonidentity.Identity{ID: epTransferID},
		Type:         transfer.TypeBlind,
		ConfbridgeID: epConfbridgeID,
	}
	gc := &cmgroupcall.Groupcall{AnswerCallID: epOtherCallID}

	t.Run("unblock failure is tolerated and the transfer is still completed", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(nil, errEP)
		e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(&transfer.Transfer{Identity: tr.Identity}, nil).Times(2)
		e.mockDB.EXPECT().TransferUpdate(ctx, &transfer.Transfer{Identity: tr.Identity, TransfereeCallID: epOtherCallID}).Return(nil)
		e.mockReq.EXPECT().CallV1ConfbridgeAnswer(ctx, epConfbridgeID).Return(errEP)

		if err := e.h.TransfereeAnswer(ctx, tr, gc); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("transferee call id update failure stops before answering", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1ConfbridgeFlagRemove(ctx, epConfbridgeID, cmconfbridge.FlagNoAutoLeave).Return(&cmconfbridge.Confbridge{}, nil)
		e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(nil, errEP)
		// CallV1ConfbridgeAnswer has no expectation: the confbridge must not be answered.

		if err := e.h.TransfereeAnswer(ctx, tr, gc); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})
}

func Test_TransfererHangup_unsupportedType(t *testing.T) {
	e := newEPEnv(t)

	err := e.h.TransfererHangup(context.Background(), &transfer.Transfer{Type: transfer.Type("unknown")}, epTransfererCall())
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_transfererHangupTypeAttended_failures(t *testing.T) {
	ctx := context.Background()
	tr := &transfer.Transfer{
		Identity:     commonidentity.Identity{ID: epTransferID},
		Type:         transfer.TypeAttended,
		GroupcallID:  epGroupcallID,
		ConfbridgeID: epConfbridgeID,
	}
	answered := &cmgroupcall.Groupcall{AnswerCallID: epOtherCallID}

	t.Run("no answer and terminate failure is tolerated", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1GroupcallGet(ctx, epGroupcallID).Return(&cmgroupcall.Groupcall{}, nil)
		e.mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, epConfbridgeID).Return(nil, errEP)
		if err := e.h.TransfererHangup(ctx, tr, epTransfererCall()); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("transferee call id update fails", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1GroupcallGet(ctx, epGroupcallID).Return(answered, nil)
		e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(nil, errEP)
		// the attended rollback must not run when the update failed (no confbridge lookup).
		if err := e.h.TransfererHangup(ctx, tr, epTransfererCall()); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})

	t.Run("attended unblock fails", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1GroupcallGet(ctx, epGroupcallID).Return(answered, nil)
		e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(&transfer.Transfer{Identity: tr.Identity}, nil).Times(2)
		e.mockDB.EXPECT().TransferUpdate(ctx, gomock.Any()).Return(nil)
		e.mockReq.EXPECT().CallV1ConfbridgeGet(ctx, epConfbridgeID).Return(nil, errEP)
		if err := e.h.TransfererHangup(ctx, tr, epTransfererCall()); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})
}

func Test_transferInit_failures(t *testing.T) {
	ctx := context.Background()

	t.Run("transferer call lookup fails", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1CallGet(ctx, epTransfererCallID).Return(nil, errEP)
		_, err := e.h.ServiceStart(ctx, transfer.TypeBlind, epTransfererCallID, epAddresses())
		if !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})

	t.Run("transferer call has no confbridge", func(t *testing.T) {
		e := newEPEnv(t)

		// no flow is created for a call that is not in a confbridge.
		e.mockReq.EXPECT().CallV1CallGet(ctx, epTransfererCallID).Return(&cmcall.Call{Identity: commonidentity.Identity{ID: epTransfererCallID}}, nil)
		if _, err := e.h.ServiceStart(ctx, transfer.TypeBlind, epTransfererCallID, epAddresses()); err == nil {
			t.Errorf("Wrong match. expect: error, got: ok")
		}
	})

	t.Run("flow create fails", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockReq.EXPECT().CallV1CallGet(ctx, epTransfererCallID).Return(epTransfererCall(), nil)
		e.mockReq.EXPECT().FlowV1FlowCreate(ctx, epCustomerID, fmflow.TypeTransfer, gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, errEP)
		if _, err := e.h.ServiceStart(ctx, transfer.TypeBlind, epTransfererCallID, epAddresses()); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})
}

func Test_get_mapsNotFound(t *testing.T) {
	ctx := context.Background()
	id := epTransferID

	getters := []struct {
		name   string
		expect func(e *epEnv, err error)
		call   func(e *epEnv) (*transfer.Transfer, error)
	}{
		{
			name:   "Get",
			expect: func(e *epEnv, err error) { e.mockDB.EXPECT().TransferGet(ctx, id).Return(nil, err) },
			call:   func(e *epEnv) (*transfer.Transfer, error) { return e.h.Get(ctx, id) },
		},
		{
			name:   "GetByGroupcallID",
			expect: func(e *epEnv, err error) { e.mockDB.EXPECT().TransferGetByGroupcallID(ctx, id).Return(nil, err) },
			call:   func(e *epEnv) (*transfer.Transfer, error) { return e.h.GetByGroupcallID(ctx, id) },
		},
		{
			name: "GetByTransfererCallID",
			expect: func(e *epEnv, err error) {
				e.mockDB.EXPECT().TransferGetByTransfererCallID(ctx, id).Return(nil, err)
			},
			call: func(e *epEnv) (*transfer.Transfer, error) { return e.h.GetByTransfererCallID(ctx, id) },
		},
	}

	for _, g := range getters {
		t.Run(g.name+" not found is a typed not found error", func(t *testing.T) {
			e := newEPEnv(t)
			g.expect(e, dbhandler.ErrNotFound)

			res, err := g.call(e)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}

			var ve *cerrors.VoipbinError
			if !stderrors.As(err, &ve) {
				t.Fatalf("Wrong match. expect: *VoipbinError, got: %T %v", err, err)
			}
			if !stderrors.Is(err, dbhandler.ErrNotFound) {
				t.Errorf("Wrong match. expect: the cause ErrNotFound is preserved, got: %v", err)
			}
		})

		t.Run(g.name+" other error is wrapped and not typed", func(t *testing.T) {
			e := newEPEnv(t)
			g.expect(e, errEP)

			_, err := g.call(e)
			if !stderrors.Is(err, errEP) {
				t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
			}
			var ve *cerrors.VoipbinError
			if stderrors.As(err, &ve) {
				t.Errorf("Wrong match. expect: an untyped error, got: %v", err)
			}
		})
	}
}

func Test_updateTransfereeCallID_failures(t *testing.T) {
	ctx := context.Background()
	base := &transfer.Transfer{Identity: commonidentity.Identity{ID: epTransferID}}

	t.Run("update fails", func(t *testing.T) {
		e := newEPEnv(t)

		e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(&transfer.Transfer{Identity: base.Identity}, nil)
		e.mockDB.EXPECT().TransferUpdate(ctx, &transfer.Transfer{Identity: base.Identity, TransfereeCallID: epOtherCallID}).Return(errEP)
		if _, err := e.h.updateTransfereeCallID(ctx, epTransferID, epOtherCallID); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})

	t.Run("re-read after update fails", func(t *testing.T) {
		e := newEPEnv(t)

		gomock.InOrder(
			e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(&transfer.Transfer{Identity: base.Identity}, nil),
			e.mockDB.EXPECT().TransferUpdate(ctx, gomock.Any()).Return(nil),
			e.mockDB.EXPECT().TransferGet(ctx, epTransferID).Return(nil, errEP),
		)
		if _, err := e.h.updateTransfereeCallID(ctx, epTransferID, epOtherCallID); !stderrors.Is(err, errEP) {
			t.Errorf("Wrong match. expect: wrapped %v, got: %v", errEP, err)
		}
	})
}
