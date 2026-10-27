package callhandler

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
)

// Test_chainedCallID_errors verifies that every failure of the chained call id add and remove
// operations ends the transaction without commit (or never reaches the later steps) and
// returns the cause to the caller.
func Test_chainedCallID_errors(t *testing.T) {
	id := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	chainedID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000002")
	customerID := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000003")
	errCause := fmt.Errorf("storage failure")

	masterCall := func(status call.Status) *call.Call {
		return &call.Call{
			Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
			Status:   status,
		}
	}
	tx := &sql.Tx{}

	operations := []struct {
		name string

		run        func(h *callHandler, ctx context.Context) (*call.Call, error)
		expectTXOp func(m *covBMocks, ctx context.Context, err error)
		masterArg  uuid.UUID // the master call id the chained call is updated with
	}{
		{
			name: "add",
			run: func(h *callHandler, ctx context.Context) (*call.Call, error) {
				return h.ChainedCallIDAdd(ctx, id, chainedID)
			},
			expectTXOp: func(m *covBMocks, ctx context.Context, err error) {
				m.db.EXPECT().CallTXAddChainedCallID(ctx, tx, id, chainedID).Return(err)
			},
			masterArg: id,
		},
		{
			name: "remove",
			run: func(h *callHandler, ctx context.Context) (*call.Call, error) {
				return h.ChainedCallIDRemove(ctx, id, chainedID)
			},
			expectTXOp: func(m *covBMocks, ctx context.Context, err error) {
				m.db.EXPECT().CallTXRemoveChainedCallID(ctx, tx, id, chainedID).Return(err)
			},
			masterArg: uuid.Nil,
		},
	}

	for _, op := range operations {
		t.Run(op.name+" transaction start failure", func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			m.db.EXPECT().CallTXStart(ctx, id).Return(nil, nil, errCause)

			res, err := op.run(h, ctx)
			if err != errCause || res != nil {
				t.Errorf("Wrong match. expect: cause error, got: %v, %v", res, err)
			}
		})

		for _, status := range []call.Status{call.StatusTerminating, call.StatusCanceling, call.StatusHangup} {
			t.Run(op.name+" master call is "+string(status), func(t *testing.T) {
				h, m := newCovBHandler(t)
				ctx := context.Background()

				m.db.EXPECT().CallTXStart(ctx, id).Return(tx, masterCall(status), nil)
				m.db.EXPECT().CallTXFinish(tx, false)

				res, err := op.run(h, ctx)
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: invalid status error, got: %v, %v", res, err)
				}
			})
		}

		t.Run(op.name+" chain update failure", func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			m.db.EXPECT().CallTXStart(ctx, id).Return(tx, masterCall(call.StatusProgressing), nil)
			op.expectTXOp(m, ctx, errCause)
			m.db.EXPECT().CallTXFinish(tx, false)

			res, err := op.run(h, ctx)
			if err != errCause || res != nil {
				t.Errorf("Wrong match. expect: cause error, got: %v, %v", res, err)
			}
		})

		t.Run(op.name+" master call id update failure", func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			m.db.EXPECT().CallTXStart(ctx, id).Return(tx, masterCall(call.StatusProgressing), nil)
			op.expectTXOp(m, ctx, nil)
			m.db.EXPECT().CallSetMasterCallID(ctx, chainedID, op.masterArg).Return(errCause)
			m.db.EXPECT().CallTXFinish(tx, false)

			res, err := op.run(h, ctx)
			if err != errCause || res != nil {
				t.Errorf("Wrong match. expect: cause error, got: %v, %v", res, err)
			}
		})

		t.Run(op.name+" master call reload failure", func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			m.db.EXPECT().CallTXStart(ctx, id).Return(tx, masterCall(call.StatusProgressing), nil)
			op.expectTXOp(m, ctx, nil)
			m.db.EXPECT().CallSetMasterCallID(ctx, chainedID, op.masterArg).Return(nil)
			m.db.EXPECT().CallTXFinish(tx, true)
			m.db.EXPECT().CallGet(ctx, id).Return(nil, errCause)

			res, err := op.run(h, ctx)
			if err != errCause || res != nil {
				t.Errorf("Wrong match. expect: cause error, got: %v, %v", res, err)
			}
		})

		t.Run(op.name+" chained call reload failure keeps the committed change but reports the error", func(t *testing.T) {
			h, m := newCovBHandler(t)
			ctx := context.Background()

			updated := masterCall(call.StatusProgressing)
			m.db.EXPECT().CallTXStart(ctx, id).Return(tx, masterCall(call.StatusProgressing), nil)
			op.expectTXOp(m, ctx, nil)
			m.db.EXPECT().CallSetMasterCallID(ctx, chainedID, op.masterArg).Return(nil)
			m.db.EXPECT().CallTXFinish(tx, true)
			m.db.EXPECT().CallGet(ctx, id).Return(updated, nil)
			m.notify.EXPECT().PublishWebhookEvent(ctx, customerID, call.EventTypeCallUpdated, updated)
			m.db.EXPECT().CallGet(ctx, chainedID).Return(nil, errCause)

			res, err := op.run(h, ctx)
			if err != errCause || res != nil {
				t.Errorf("Wrong match. expect: cause error, got: %v, %v", res, err)
			}
		})
	}
}
