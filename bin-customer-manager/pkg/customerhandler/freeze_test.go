package customerhandler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Freeze(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseCustomerGet *customer.Customer

		expectDBFreeze       bool
		expectDBGet2         bool
		expectPublish        bool
		expectErr            bool
		responseCustomerGet2 *customer.Customer
	}{
		{
			name: "active customer - success",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
			expectDBFreeze: true,
			expectDBGet2:   true,
			expectPublish:  true,
			expectErr:      false,
			responseCustomerGet2: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
		},
		{
			name: "already frozen - idempotent",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
			expectDBFreeze: false,
			expectDBGet2:   false,
			expectPublish:  false,
			expectErr:      false,
		},
		{
			name: "deleted customer - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			expectDBFreeze: false,
			expectDBGet2:   false,
			expectPublish:  false,
			expectErr:      true,
		},
		{
			name: "deleted customer with tm_delete - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: func() *customer.Customer {
				tmDelete := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				return &customer.Customer{
					ID:       uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
					Status:   customer.StatusActive,
					TMDelete: &tmDelete,
				}
			}(),
			expectDBFreeze: false,
			expectDBGet2:   false,
			expectPublish:  false,
			expectErr:      true,
		},
		{
			name:                "get customer fails - error",
			id:                  uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: nil,
			expectDBFreeze:      false,
			expectDBGet2:        false,
			expectPublish:       false,
			expectErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// initial Get
			if tt.responseCustomerGet == nil {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf("not found"))
			} else {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet, nil)
			}

			if tt.expectDBFreeze {
				mockDB.EXPECT().CustomerFreeze(gomock.Any(), tt.id).Return(nil)
			}

			if tt.expectDBGet2 {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet2, nil)
			}

			if tt.expectPublish {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerFrozen, tt.responseCustomerGet2).Return()
			}

			res, err := h.Freeze(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if res == nil {
				t.Errorf("Expected result, got nil")
			}
		})
	}
}

func Test_Freeze_DBFreezeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusActive,
	}, nil)
	mockDB.EXPECT().CustomerFreeze(gomock.Any(), id).Return(fmt.Errorf("db error"))

	_, err := h.Freeze(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_Freeze_GetAfterFreezeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusActive,
	}, nil)
	mockDB.EXPECT().CustomerFreeze(gomock.Any(), id).Return(nil)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Freeze(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_FreezeAndDelete(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		// Freeze() internally calls Get then conditionally freezes.
		// We mock the full Freeze() call chain here.
		responseCustomerGet *customer.Customer // first Get inside Freeze()

		expectDBFreeze            bool
		responseCustomerGetFreeze *customer.Customer // Get after freeze DB call

		expectDBAnonymize            bool
		expectDBGetAfterAnonymize    bool
		responseCustomerGetAnonymize *customer.Customer // Get after anonymize

		expectPublishFrozen  bool
		expectPublishDeleted bool
		expectErr            bool
	}{
		{
			name: "active customer - freeze and delete success",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
			expectDBFreeze: true,
			responseCustomerGetFreeze: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
			expectPublishFrozen:       true,
			expectDBAnonymize:         true,
			expectDBGetAfterAnonymize: true,
			responseCustomerGetAnonymize: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			expectPublishDeleted: true,
			expectErr:            false,
		},
		{
			name: "already frozen - delete success",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
			// Freeze() returns early for already-frozen (no DB freeze, no second Get, no publish)
			expectDBFreeze:            false,
			responseCustomerGetFreeze: nil,
			expectPublishFrozen:       false,
			expectDBAnonymize:         true,
			expectDBGetAfterAnonymize: true,
			responseCustomerGetAnonymize: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			expectPublishDeleted: true,
			expectErr:            false,
		},
		{
			name: "deleted customer - freeze returns error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			// Freeze() rejects deleted customers with an error
			expectDBFreeze:            false,
			expectDBAnonymize:         false,
			expectDBGetAfterAnonymize: false,
			expectPublishFrozen:       false,
			expectPublishDeleted:      false,
			expectErr:                 true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// Freeze() internal: Get
			mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet, nil)

			if tt.expectDBFreeze {
				mockDB.EXPECT().CustomerFreeze(gomock.Any(), tt.id).Return(nil)
				// Freeze() internal: Get after DB freeze
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGetFreeze, nil)
			}

			if tt.expectPublishFrozen {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerFrozen, tt.responseCustomerGetFreeze).Return()
			}

			if tt.expectDBAnonymize {
				shortID := tt.id.String()[:8]
				anonName := fmt.Sprintf("deleted_user_%s", shortID)
				anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)
				mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), tt.id, anonName, anonEmail).Return(nil)
			}

			if tt.expectDBGetAfterAnonymize {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGetAnonymize, nil)
			}

			if tt.expectPublishDeleted {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, tt.responseCustomerGetAnonymize).Return()
			}

			res, err := h.FreezeAndDelete(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if res == nil {
				t.Errorf("Expected result, got nil")
			}
		})
	}
}

func Test_FreezeAndDelete_AnonymizeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	// Freeze succeeds (active -> frozen)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusActive,
	}, nil)
	mockDB.EXPECT().CustomerFreeze(gomock.Any(), id).Return(nil)
	frozenCustomer := &customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(frozenCustomer, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerFrozen, frozenCustomer).Return()

	// Anonymize fails
	shortID := id.String()[:8]
	anonName := fmt.Sprintf("deleted_user_%s", shortID)
	anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)
	mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), id, anonName, anonEmail).Return(fmt.Errorf("db error"))

	_, err := h.FreezeAndDelete(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_FreezeAndDelete_GetAfterAnonymizeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	// Freeze succeeds (active -> frozen)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusActive,
	}, nil)
	mockDB.EXPECT().CustomerFreeze(gomock.Any(), id).Return(nil)
	frozenCustomer := &customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(frozenCustomer, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerFrozen, frozenCustomer).Return()

	// Anonymize succeeds
	shortID := id.String()[:8]
	anonName := fmt.Sprintf("deleted_user_%s", shortID)
	anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)
	mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), id, anonName, anonEmail).Return(nil)

	// Get after anonymize fails
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("not found"))

	_, err := h.FreezeAndDelete(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

// Test_FreezeAndDelete_ConcurrentRace exercises the real, observable behavior
// change design §5.5 calls out for this out-of-scope-but-affected caller:
// today two concurrent FreezeAndDelete calls on the same customer both
// silently succeed and both publish customer_deleted (the same duplicate-
// cascade bug the CustomerAnonymizePII status guard fixes for the scheduled
// sweep); after the guard, the loser receives dbhandler.ErrNotFound and does
// not publish a second event.
//
// Both goroutines see the customer as already frozen, so Freeze() takes the
// idempotent early-return path (a single Get, no DB freeze, no frozen-event
// publish) and only CustomerAnonymizePII actually races.
func Test_FreezeAndDelete_ConcurrentRace(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	frozenCustomer := &customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}
	anonymizedCustomer := &customer.Customer{
		ID:     id,
		Status: customer.StatusDeleted,
	}

	shortID := id.String()[:8]
	anonName := fmt.Sprintf("deleted_user_%s", shortID)
	anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)

	// CustomerGet must reflect whatever the (mocked) DB state actually is at
	// call time, not call registration order — two goroutines calling
	// Freeze()'s initial Get and the post-anonymize Get interleave
	// unpredictably, so the response is driven off the same atomic flag that
	// CustomerAnonymizePII flips, mirroring real DB read-your-writes
	// semantics instead of asserting a fixed call count per response.
	//
	// The two pre-anonymize Get calls (one per goroutine, from Freeze()'s
	// idempotent-check) are additionally barriered so both goroutines are
	// guaranteed to observe "frozen" and reach FreezeAndDelete's anonymize
	// step before either one's CustomerAnonymizePII call can flip the
	// anonymized flag — otherwise a goroutine that runs to completion before
	// the other even starts would make the race deterministic instead of
	// contested, and the second goroutine would see an already-deleted
	// customer at the Freeze() step instead of racing CustomerAnonymizePII.
	var anonymized int32
	var preAnonymizeGets int32
	preAnonymizeBarrier := make(chan struct{})
	var closeBarrierOnce sync.Once
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*customer.Customer, error) {
			if atomic.LoadInt32(&anonymized) == 1 {
				return anonymizedCustomer, nil
			}
			if atomic.AddInt32(&preAnonymizeGets, 1) >= 2 {
				closeBarrierOnce.Do(func() { close(preAnonymizeBarrier) })
			} else {
				<-preAnonymizeBarrier
			}
			return frozenCustomer, nil
		},
	).AnyTimes()

	var claimed int32
	mockDB.EXPECT().CustomerAnonymizePII(gomock.Any(), id, anonName, anonEmail).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, _, _ string) error {
			if atomic.CompareAndSwapInt32(&claimed, 0, 1) {
				atomic.StoreInt32(&anonymized, 1)
				return nil
			}
			return dbhandler.ErrNotFound
		},
	).Times(2)

	mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerDeleted, anonymizedCustomer).Times(1).Return()

	const n = 2
	var wg sync.WaitGroup
	results := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := h.FreezeAndDelete(ctx, id)
			results[idx] = err
		}(i)
	}
	wg.Wait()

	winners := 0
	losers := 0
	for _, err := range results {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, dbhandler.ErrNotFound):
			losers++
		default:
			t.Errorf("Unexpected error: %v", err)
		}
	}
	if winners != 1 {
		t.Errorf("Wrong match. expect: exactly 1 winner, got: %d", winners)
	}
	if losers != 1 {
		t.Errorf("Wrong match. expect: exactly 1 loser, got: %d", losers)
	}
}

func Test_Recover(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		responseCustomerGet *customer.Customer

		expectDBRecover      bool
		expectDBGet2         bool
		expectPublish        bool
		expectErr            bool
		responseCustomerGet2 *customer.Customer
	}{
		{
			name: "frozen customer - success",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusFrozen,
			},
			expectDBRecover: true,
			expectDBGet2:    true,
			expectPublish:   true,
			expectErr:       false,
			responseCustomerGet2: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
		},
		{
			name: "active customer - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusActive,
			},
			expectDBRecover: false,
			expectDBGet2:    false,
			expectPublish:   false,
			expectErr:       true,
		},
		{
			name: "deleted customer - error",
			id:   uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: &customer.Customer{
				ID:     uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
				Status: customer.StatusDeleted,
			},
			expectDBRecover: false,
			expectDBGet2:    false,
			expectPublish:   false,
			expectErr:       true,
		},
		{
			name:                "get customer fails - error",
			id:                  uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125"),
			responseCustomerGet: nil,
			expectDBRecover:     false,
			expectDBGet2:        false,
			expectPublish:       false,
			expectErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &customerHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			// initial Get
			if tt.responseCustomerGet == nil {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf("not found"))
			} else {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet, nil)
			}

			if tt.expectDBRecover {
				mockDB.EXPECT().CustomerRecover(gomock.Any(), tt.id).Return(nil)
			}

			if tt.expectDBGet2 {
				mockDB.EXPECT().CustomerGet(gomock.Any(), tt.id).Return(tt.responseCustomerGet2, nil)
			}

			if tt.expectPublish {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), customer.EventTypeCustomerRecovered, tt.responseCustomerGet2).Return()
			}

			res, err := h.Recover(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if res == nil {
				t.Errorf("Expected result, got nil")
			}
		})
	}
}

func Test_Recover_DBRecoverError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}, nil)
	mockDB.EXPECT().CustomerRecover(gomock.Any(), id).Return(fmt.Errorf("db error"))

	_, err := h.Recover(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func Test_Recover_GetAfterRecoverError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &customerHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("4cd23368-7cb7-11ec-9466-8318ef5a7125")

	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(&customer.Customer{
		ID:     id,
		Status: customer.StatusFrozen,
	}, nil)
	mockDB.EXPECT().CustomerRecover(gomock.Any(), id).Return(nil)
	mockDB.EXPECT().CustomerGet(gomock.Any(), id).Return(nil, fmt.Errorf("not found"))

	_, err := h.Recover(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
