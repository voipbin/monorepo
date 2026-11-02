package casehandler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func newDispositionTestHandler(t *testing.T) (*caseHandler, dbhandler.DBHandler, *utilhandler.MockUtilHandler) {
	t.Helper()
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	return h, db, mockUtil
}

func insertOpenCase(t *testing.T, db dbhandler.DBHandler, customerID, caseID uuid.UUID, target string) {
	t.Helper()
	opened := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	c := &kase.Case{
		ID: caseID, CustomerID: customerID,
		Peer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: target},
		ReferenceType: "call",
		Status:        kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(context.Background(), c); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}
}

// Verify (1): every enum value is stored and echoed in the response.
func Test_Close_Disposition_AllValuesStored(t *testing.T) {
	values := []string{
		kase.DispositionResolved, kase.DispositionUnresolved, kase.DispositionFollowUpNeeded,
		kase.DispositionSpam, kase.DispositionOther,
	}
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	for i, v := range values {
		t.Run(v, func(t *testing.T) {
			h, db, mockUtil := newDispositionTestHandler(t)
			ctx := context.Background()
			customerID := uuid.FromStringOrNil("f1b2c3d4-a001-a001-a001-000000000001")
			caseID := uuid.Must(uuid.NewV4())
			agentID := uuid.Must(uuid.NewV4())
			insertOpenCase(t, db, customerID, caseID, "+155****10"+string(rune('0'+i)))

			mockUtil.EXPECT().TimeNow().Return(&now)
			res, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, agentID, v)
			if err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if res.Case.Disposition != v {
				t.Errorf("response disposition = %q, want %q", res.Case.Disposition, v)
			}
			if res.Case.ClosedReason != kase.ClosedReasonAgentClosed {
				t.Errorf("closed_reason = %q, want agent_closed", res.Case.ClosedReason)
			}
			got, err := db.CaseGetByID(ctx, caseID)
			if err != nil {
				t.Fatalf("CaseGetByID() error = %v", err)
			}
			if got.Disposition != v {
				t.Errorf("stored disposition = %q, want %q", got.Disposition, v)
			}
		})
	}
}

// Verify (2): no disposition keeps the current behavior.
func Test_Close_Disposition_Omitted(t *testing.T) {
	h, db, mockUtil := newDispositionTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("f1b2c3d4-a002-a002-a002-000000000001")
	caseID := uuid.Must(uuid.NewV4())
	insertOpenCase(t, db, customerID, caseID, "+155****2001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&now)
	res, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, uuid.Must(uuid.NewV4()), "")
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if res.Case.Disposition != "" {
		t.Errorf("disposition = %q, want empty", res.Case.Disposition)
	}
	if res.ClosedReason != kase.ClosedReasonAgentClosed {
		t.Errorf("closed_reason = %q, want agent_closed", res.ClosedReason)
	}
	if res.Case.Status != kase.StatusClosed {
		t.Errorf("status = %q, want closed", res.Case.Status)
	}
}

// Verify (3): a value outside the enum is InvalidArgument and nothing changes.
func Test_Close_Disposition_Invalid(t *testing.T) {
	h, db, _ := newDispositionTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("f1b2c3d4-a003-a003-a003-000000000001")
	caseID := uuid.Must(uuid.NewV4())
	insertOpenCase(t, db, customerID, caseID, "+155****3001")

	// No TimeNow expectation: validation must fail before any write.
	res, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, uuid.Must(uuid.NewV4()), "not_a_value")
	if err == nil || res != nil {
		t.Fatalf("Close() = %v, %v; want nil, error", res, err)
	}
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("error = %v, want InvalidArgument VoipbinError", err)
	}

	got, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if got.Status != kase.StatusOpen || got.Disposition != "" {
		t.Errorf("case changed: status=%q disposition=%q", got.Status, got.Disposition)
	}
}

// Verify (4): the second close reports AlreadyClosed and keeps the first value.
func Test_Close_Disposition_SecondCloseKeepsFirst(t *testing.T) {
	h, db, mockUtil := newDispositionTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("f1b2c3d4-a004-a004-a004-000000000001")
	caseID := uuid.Must(uuid.NewV4())
	insertOpenCase(t, db, customerID, caseID, "+155****4001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	if _, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, uuid.Must(uuid.NewV4()), kase.DispositionResolved); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	second, err := h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, uuid.Must(uuid.NewV4()), kase.DispositionSpam)
	if err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if !second.AlreadyClosed {
		t.Errorf("AlreadyClosed = false, want true")
	}
	if second.Case.Disposition != kase.DispositionResolved {
		t.Errorf("disposition = %q, want first value %q", second.Case.Disposition, kase.DispositionResolved)
	}
}

// Verify (5): with concurrent closes exactly one writer wins.
func Test_Close_Disposition_ConcurrentOneWinner(t *testing.T) {
	h, db, mockUtil := newDispositionTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("f1b2c3d4-a005-a005-a005-000000000001")
	caseID := uuid.Must(uuid.NewV4())
	insertOpenCase(t, db, customerID, caseID, "+155****5001")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	values := []string{kase.DispositionResolved, kase.DispositionUnresolved, kase.DispositionOther, kase.DispositionSpam}
	mockUtil.EXPECT().TimeNow().Return(&now).Times(len(values))

	var wg sync.WaitGroup
	results := make([]*CloseResult, len(values))
	errs := make([]error, len(values))
	for i, v := range values {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = h.Close(ctx, customerID, caseID, commonidentity.OwnerTypeAgent, uuid.Must(uuid.NewV4()), v)
		}()
	}
	wg.Wait()

	winners := 0
	var stored string
	for i := range values {
		if errs[i] != nil {
			continue // sqlite may report a lock error for a losing writer
		}
		if !results[i].AlreadyClosed {
			winners++
		}
	}
	got, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	stored = got.Disposition
	if winners != 1 {
		t.Errorf("winners = %d, want 1", winners)
	}
	for i := range values {
		if errs[i] == nil && results[i].Case.Disposition != stored {
			t.Errorf("result %d disposition = %q, want stored %q", i, results[i].Case.Disposition, stored)
		}
	}
}
