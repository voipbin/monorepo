package casehandler

import (
	"context"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_GetOrCreate_ValidHint_UsesIt verifies design §4.3: a valid
// case_id hint (correct tenant, still open) is used directly, skipping
// peer/reference_type matching entirely.
func Test_GetOrCreate_ValidHint_UsesIt(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	config.SetCaseTimeoutHoursForTest(24)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	db := dbhandler.NewHandler(dbTest, mockCache)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("f1b2c3d4-7002-7002-7002-000000000001")
	hintedCaseID := uuid.FromStringOrNil("f1b2c3d4-7002-7002-7002-000000000002")
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	opened := now.Add(-30 * time.Minute)

	hinted := &kase.Case{
		ID: hintedCaseID, CustomerID: customerID,
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130001"}, ReferenceType: "conversation_message",
		Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
	}
	if err := db.CaseInsert(ctx, hinted); err != nil {
		t.Fatalf("CaseInsert() error = %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)

	// Different peer/reference_type on purpose: proves the hint path
	// short-circuits peer matching entirely (if it fell through to peer
	// matching, no open case would be found for this peer and a NEW case
	// would be inserted instead of reusing hintedCaseID).
	res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: "+19999999999"}, "call", &hintedCaseID, "")
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if res == nil || res.ID != hintedCaseID {
		t.Errorf("expected to use the valid hint case %s, got: %v", hintedCaseID, res)
	}
}

// Test_GetOrCreate_StaleHint_FallsThrough verifies design §4.3's
// mandatory validation: a hint referencing a WRONG TENANT's case, a
// CLOSED case, or a NON-EXISTENT case must never be trusted -- each
// falls through to normal peer/reference_type resolution as if no hint
// were given, never surfaced as an error.
func Test_GetOrCreate_StaleHint_FallsThrough(t *testing.T) {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		peerTarget string
		setupHint  func(t *testing.T, db dbhandler.DBHandler, ctx context.Context, ownTenant uuid.UUID) *uuid.UUID
	}{
		{
			name:       "wrong tenant",
			peerTarget: "+15551130097",
			setupHint: func(t *testing.T, db dbhandler.DBHandler, ctx context.Context, ownTenant uuid.UUID) *uuid.UUID {
				otherTenant := uuid.FromStringOrNil("f1b2c3d4-7003-9999-9999-000000000099")
				wrongTenantCaseID := uuid.FromStringOrNil("f1b2c3d4-7003-7003-7003-000000000010")
				opened := now.Add(-30 * time.Minute)
				c := &kase.Case{
					ID: wrongTenantCaseID, CustomerID: otherTenant,
					Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551139999"}, ReferenceType: "call",
					Status: kase.StatusOpen, OpenedAt: &opened, TMCreate: &opened, TMUpdate: &opened,
				}
				if err := db.CaseInsert(ctx, c); err != nil {
					t.Fatalf("CaseInsert() error = %v", err)
				}
				return &wrongTenantCaseID
			},
		},
		{
			name:       "closed case",
			peerTarget: "+15551130096",
			setupHint: func(t *testing.T, db dbhandler.DBHandler, ctx context.Context, ownTenant uuid.UUID) *uuid.UUID {
				closedCaseID := uuid.FromStringOrNil("f1b2c3d4-7003-7003-7003-000000000011")
				opened := now.Add(-2 * time.Hour)
				closedAt := now.Add(-1 * time.Hour)
				c := &kase.Case{
					ID: closedCaseID, CustomerID: ownTenant,
					Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551130098"}, ReferenceType: "call",
					Status: kase.StatusClosed, OpenedAt: &opened, ClosedAt: &closedAt,
					ClosedReason: kase.ClosedReasonAgentClosed, TMCreate: &opened, TMUpdate: &closedAt,
				}
				if err := db.CaseInsert(ctx, c); err != nil {
					t.Fatalf("CaseInsert() error = %v", err)
				}
				return &closedCaseID
			},
		},
		{
			name:       "non-existent case",
			peerTarget: "+15551130095",
			setupHint: func(t *testing.T, db dbhandler.DBHandler, ctx context.Context, ownTenant uuid.UUID) *uuid.UUID {
				ghost := uuid.FromStringOrNil("f1b2c3d4-7003-7003-7003-000000000012")
				return &ghost
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			config.SetCaseTimeoutHoursForTest(24)

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			db := dbhandler.NewHandler(dbTest, mockCache)
			h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: db, notifyHandler: mockNotify}
			ctx := context.Background()

			customerID := uuid.FromStringOrNil("f1b2c3d4-7003-7003-7003-000000000001")
			hint := tt.setupHint(t, db, ctx, customerID)

			mockUtil.EXPECT().TimeNow().Return(&now)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4()))

			// No open case exists for this peer -> falls through to a
			// fresh insert, proving the hint was NOT trusted.
			res, err := h.GetOrCreate(ctx, customerID, commonaddress.Address{}, commonaddress.Address{Type: commonaddress.TypeTel, Target: tt.peerTarget}, "call", hint, "")
			if err != nil {
				t.Fatalf("GetOrCreate() error = %v", err)
			}
			if res == nil {
				t.Fatal("expected a freshly created case, got nil")
			}
			if hint != nil && res.ID == *hint {
				t.Errorf("must NOT have used the invalid hint case %s", *hint)
			}
			if res.Peer.Target != tt.peerTarget {
				t.Errorf("expected fresh case scoped to the real peer, got peer_target: %s", res.Peer.Target)
			}
		})
	}
}
