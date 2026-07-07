package casehandler

import (
	"context"
	"sync"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_GetOrCreate_ConcurrentGoroutines_ExactlyOneCaseSurvives is the
// dedicated concurrency test the implementation plan calls for (Task 3.3
// sub-task 5): two goroutines race GetOrCreate for the SAME
// (customerID, peerType, peerTarget, referenceType) with no prior Case
// at all. Exactly one Case must be created for that peer/reference_type
// -- both goroutines must resolve to the SAME case_id (design §4.2: the
// unique index correctly rejects the loser's INSERT, and the loser's
// retry-select finds and uses the winner).
func Test_GetOrCreate_ConcurrentGoroutines_ExactlyOneCaseSurvives(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockCache.EXPECT().ContactGet(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	db := dbhandler.NewHandler(dbTest, mockCache)

	customerID := uuid.FromStringOrNil("f1b2c3d4-700a-700a-700a-000000000001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	const numGoroutines = 8
	results := make([]uuid.UUID, numGoroutines)
	errs := make([]error, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Each goroutine gets its own utilHandler mock (UUIDCreate is
			// called per-goroutine, not shared) but the SAME real
			// dbhandler/DB, which is what actually exercises the
			// uq_case_open_peer race.
			gc := gomock.NewController(t)
			mockUtil := utilhandler.NewMockUtilHandler(gc)
			mockReq := requesthandler.NewMockRequestHandler(gc)
			mockNotify := notifyhandler.NewMockNotifyHandler(gc)
			h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}

			mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
			mockUtil.EXPECT().UUIDCreate().DoAndReturn(func() uuid.UUID {
				return uuid.Must(uuid.NewV4())
			}).AnyTimes()

			res, err := h.GetOrCreate(context.Background(), customerID, commonaddress.Address{}, commonaddress.TypeTel, "+15551190001", "call", nil)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = res.ID
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: GetOrCreate() error = %v", i, err)
		}
	}

	first := results[0]
	for i, id := range results {
		if id != first {
			t.Errorf("goroutine %d resolved to a DIFFERENT case_id (%s) than goroutine 0 (%s) -- race allowed two cases for the same peer", i, id, first)
		}
	}

	// Verify exactly one open Case exists in the DB for this peer.
	unresolved, err := db.CaseListUnresolved(context.Background(), customerID)
	if err != nil {
		t.Fatalf("CaseListUnresolved() error = %v", err)
	}
	count := 0
	for _, c := range unresolved {
		if c.PeerTarget == "+15551190001" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 case in the DB for this peer, found: %d", count)
	}
}
