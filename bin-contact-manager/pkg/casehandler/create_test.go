package casehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_Create_HappyPath verifies design VOIP-1243 §3.3: a plain insert
// with Status=StatusOpen, PreviousCaseID=nil, Owner left zero-value (no
// auto-assignment), and Name/Detail persisted.
func Test_Create_HappyPath(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9721-9721-9721-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9721-9721-9721-000000000002")
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().UUIDCreate().Return(caseID)
	mockUtil.EXPECT().TimeNow().Return(&now)

	res, err := h.Create(
		ctx, customerID,
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0701"},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****9701"}, "call",
		"VIP escalation", "customer called about billing",
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if res.ID != caseID {
		t.Errorf("unexpected ID: %v", res.ID)
	}
	if res.CustomerID != customerID {
		t.Errorf("unexpected CustomerID: %v", res.CustomerID)
	}
	if res.Status != kase.StatusOpen {
		t.Errorf("expected StatusOpen, got: %v", res.Status)
	}
	if res.PreviousCaseID != nil {
		t.Errorf("expected nil PreviousCaseID, got: %v", res.PreviousCaseID)
	}
	if res.OwnerType != "" || res.OwnerID != uuid.Nil {
		t.Errorf("expected zero-value Owner (no auto-assignment), got: type=%v id=%v", res.OwnerType, res.OwnerID)
	}
	if res.Name != "VIP escalation" || res.Detail != "customer called about billing" {
		t.Errorf("unexpected Name/Detail: %q / %q", res.Name, res.Detail)
	}

	// Confirm it was actually persisted.
	fetched, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		t.Fatalf("CaseGetByID() error = %v", err)
	}
	if fetched.Name != "VIP escalation" || fetched.Detail != "customer called about billing" {
		t.Errorf("unexpected persisted Name/Detail: %q / %q", fetched.Name, fetched.Detail)
	}
}

// Test_Create_DuplicateOpenPeer_TranslatesToAlreadyExists verifies
// design §3.3 step 2: a uq_case_open_peer violation (dbhandler.ErrDuplicate)
// is translated to a typed cerrors.AlreadyExists, never a raw error.
func Test_Create_DuplicateOpenPeer_TranslatesToAlreadyExists(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9722-9722-9722-000000000001")
	firstCaseID := uuid.FromStringOrNil("f1b2c3d4-9722-9722-9722-000000000002")
	secondCaseID := uuid.FromStringOrNil("f1b2c3d4-9722-9722-9722-000000000003")
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	self := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0702"}
	peer := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****9702"}
	referenceType := "call"

	mockUtil.EXPECT().UUIDCreate().Return(firstCaseID)
	mockUtil.EXPECT().TimeNow().Return(&now)
	if _, err := h.Create(ctx, customerID, self, peer, referenceType, "", ""); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	mockUtil.EXPECT().UUIDCreate().Return(secondCaseID)
	mockUtil.EXPECT().TimeNow().Return(&now)
	_, err := h.Create(ctx, customerID, self, peer, referenceType, "", "")

	var ve *cerrors.VoipbinError
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusAlreadyExists {
		t.Errorf("expected StatusAlreadyExists, got: %v", ve.Status)
	}
}

// Test_Create_Deadlock_TranslatesToUnavailable verifies design §3.3's
// ErrDeadlock handling: Create does NOT retry (unlike GetOrCreate); it
// translates dbhandler.ErrDeadlock to a typed cerrors.Unavailable.
// Simulated via a fake dbhandler.DBHandler double returning ErrDeadlock
// directly from CaseInsert.
func Test_Create_Deadlock_TranslatesToUnavailable(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-9723-9723-9723-000000000001")
	caseID := uuid.FromStringOrNil("f1b2c3d4-9723-9723-9723-000000000002")
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().UUIDCreate().Return(caseID)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CaseInsert(ctx, gomock.Any()).Return(dbhandler.ErrDeadlock)

	_, err := h.Create(
		ctx, customerID,
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0703"},
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****9703"}, "call", "", "",
	)

	var ve *cerrors.VoipbinError
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusUnavailable {
		t.Errorf("expected StatusUnavailable, got: %v", ve.Status)
	}
}
