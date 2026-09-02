package campaignhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

// captureWarnLogs installs a local logrus hook on the standard logger
// (which package-level logrus.WithFields(...) calls in reconcile.go write
// through) and returns a function that reports whether any warn-level
// entry was captured. Callers must invoke the returned cleanup themselves
// via defer.
func captureWarnLogs() (hadWarn func() bool, cleanup func()) {
	hook := logrustest.NewLocal(logrus.StandardLogger())
	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)

	hadWarn = func() bool {
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.WarnLevel {
				return true
			}
		}
		return false
	}
	cleanup = func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
		logrus.SetLevel(origLevel)
	}
	return hadWarn, cleanup
}

func Test_ReconcileOrphanedFlows_cleansGenuineOrphan(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111101")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222201"),
		TMDelete: timePtr(now.Add(-time.Hour)),
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: nil}, nil)
	mockReq.EXPECT().FlowV1FlowDelete(gomock.Any(), c.FlowID).Return(&fmflow.Flow{}, nil)

	beforeCleaned := testutil.ToFloat64(promCampaignFlowReconcileCleanedTotal)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Cleaned != 1 {
		t.Errorf("Wrong match. expect Cleaned: 1, got: %d", res.Cleaned)
	}
	if res.Skipped != 0 || res.Failed != 0 {
		t.Errorf("Wrong match. expect Skipped=0 Failed=0, got: %+v", res)
	}

	afterCleaned := testutil.ToFloat64(promCampaignFlowReconcileCleanedTotal)
	if afterCleaned != beforeCleaned+1 {
		t.Errorf("Wrong match. expect cleaned metric delta 1, got: %v", afterCleaned-beforeCleaned)
	}
}

func Test_ReconcileOrphanedFlows_alreadyCleanFlow_skipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111102")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222202"),
		TMDelete: timePtr(now.Add(-time.Hour)),
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	// flow already deleted -- FlowV1FlowDelete must NOT be called; no
	// EXPECT() means gomock fails the test on any unexpected call.
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: timePtr(now)}, nil)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Skipped != 1 {
		t.Errorf("Wrong match. expect Skipped: 1, got: %d", res.Skipped)
	}
	if res.Cleaned != 0 || res.Failed != 0 {
		t.Errorf("Wrong match. expect Cleaned=0 Failed=0, got: %+v", res)
	}
}

func Test_ReconcileOrphanedFlows_typedNotFound_skipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111103")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222203"),
		TMDelete: timePtr(now.Add(-time.Hour)),
	}

	notFoundErr := cerrors.NotFound(commonoutline.ServiceNameFlowManager, "FLOW_NOT_FOUND", "flow not found")

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(nil, notFoundErr)

	beforeFailed := testutil.ToFloat64(promCampaignFlowReconcileFailedTotal)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Skipped != 1 {
		t.Errorf("Wrong match. expect Skipped: 1, got: %d", res.Skipped)
	}
	if res.Failed != 0 {
		t.Errorf("Wrong match. expect Failed: 0 (typed not-found is a clean state, not a failure), got: %d", res.Failed)
	}

	afterFailed := testutil.ToFloat64(promCampaignFlowReconcileFailedTotal)
	if afterFailed != beforeFailed {
		t.Errorf("Wrong match. expect no failed-metric increment for typed not-found, got delta: %v", afterFailed-beforeFailed)
	}
}

func Test_ReconcileOrphanedFlows_transientGetError_failed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111104")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222204"),
		TMDelete: timePtr(now.Add(-time.Hour)),
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(nil, fmt.Errorf("rpc timeout"))

	beforeFailed := testutil.ToFloat64(promCampaignFlowReconcileFailedTotal)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Failed != 1 {
		t.Errorf("Wrong match. expect Failed: 1, got: %d", res.Failed)
	}
	if res.Cleaned != 0 || res.Skipped != 0 {
		t.Errorf("Wrong match. expect Cleaned=0 Skipped=0, got: %+v", res)
	}

	afterFailed := testutil.ToFloat64(promCampaignFlowReconcileFailedTotal)
	if afterFailed != beforeFailed+1 {
		t.Errorf("Wrong match. expect failed metric delta 1, got: %v", afterFailed-beforeFailed)
	}
}

func Test_ReconcileOrphanedFlows_deleteError_failed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111105")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222205"),
		TMDelete: timePtr(now.Add(-time.Hour)),
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: nil}, nil)
	mockReq.EXPECT().FlowV1FlowDelete(gomock.Any(), c.FlowID).Return(nil, fmt.Errorf("backend error"))

	beforeFailed := testutil.ToFloat64(promCampaignFlowReconcileFailedTotal)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Failed != 1 {
		t.Errorf("Wrong match. expect Failed: 1, got: %d", res.Failed)
	}
	if res.Cleaned != 0 {
		t.Errorf("Wrong match. expect Cleaned: 0, got: %d", res.Cleaned)
	}

	afterFailed := testutil.ToFloat64(promCampaignFlowReconcileFailedTotal)
	if afterFailed != beforeFailed+1 {
		t.Errorf("Wrong match. expect failed metric delta 1 (same shared counter as the FlowV1FlowGet-error case), got: %v", afterFailed-beforeFailed)
	}
}

func Test_ReconcileOrphanedFlows_defensiveGuardNilTMDelete_skipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111106")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222206"),
		TMDelete: nil, // should never happen given the query's own filter -- defensive guard
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	// no FlowV1FlowGet/FlowV1FlowDelete EXPECT() -- either call fails the test

	hadWarn, cleanup := captureWarnLogs()
	defer cleanup()

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Skipped != 1 {
		t.Errorf("Wrong match. expect Skipped: 1, got: %d", res.Skipped)
	}
	if !hadWarn() {
		t.Errorf("Wrong match. expect a warn log for the defensive guard, got none")
	}
}

func Test_ReconcileOrphanedFlows_saturatedSignal(t *testing.T) {
	origScanLimit := scanLimit
	scanLimit = 3
	defer func() { scanLimit = origScanLimit }()

	tests := []struct {
		name            string
		candidateCount  int
		expectSaturated bool
	}{
		{"batch reaches scanLimit", 3, true},
		{"batch below scanLimit", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

			now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
			candidates := make([]*campaign.Campaign, 0, tt.candidateCount)
			for i := 0; i < tt.candidateCount; i++ {
				candidates = append(candidates, &campaign.Campaign{
					Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
					FlowID:   uuid.Must(uuid.NewV4()),
					// well outside any recentInterval used below
					TMDelete: timePtr(now.Add(-48 * time.Hour)),
				})
			}

			mockUtil.EXPECT().TimeNow().Return(&now)
			mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return(candidates, nil)
			mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(&fmflow.Flow{TMDelete: timePtr(now)}, nil).Times(tt.candidateCount)

			res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.Saturated != tt.expectSaturated {
				t.Errorf("Wrong match. expect Saturated: %v, got: %v", tt.expectSaturated, res.Saturated)
			}
		})
	}
}

func Test_ReconcileOrphanedFlows_recentSaturatedDistinguishesFromSaturated(t *testing.T) {
	origScanLimit := scanLimit
	scanLimit = 3
	defer func() { scanLimit = origScanLimit }()

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	recentIntervalSec := int64(3600) // 1 hour

	tests := []struct {
		name                  string
		deletedAgo            []time.Duration
		expectSaturated       bool
		expectRecentSaturated bool
	}{
		{
			"all rows within recentInterval -- rate-risk signal fires",
			[]time.Duration{10 * time.Minute, 20 * time.Minute, 30 * time.Minute},
			true,
			true,
		},
		{
			"batch full but rows mostly older than recentInterval -- informational only",
			[]time.Duration{2 * time.Hour, 3 * time.Hour, 4 * time.Hour},
			true,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

			candidates := make([]*campaign.Campaign, 0, len(tt.deletedAgo))
			for _, ago := range tt.deletedAgo {
				candidates = append(candidates, &campaign.Campaign{
					Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
					FlowID:   uuid.Must(uuid.NewV4()),
					TMDelete: timePtr(now.Add(-ago)),
				})
			}

			mockUtil.EXPECT().TimeNow().Return(&now)
			mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return(candidates, nil)
			mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).Return(&fmflow.Flow{TMDelete: timePtr(now)}, nil).Times(len(candidates))

			res, err := h.ReconcileOrphanedFlows(context.Background(), recentIntervalSec)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.Saturated != tt.expectSaturated {
				t.Errorf("Wrong match. expect Saturated: %v, got: %v", tt.expectSaturated, res.Saturated)
			}
			if res.RecentSaturated != tt.expectRecentSaturated {
				t.Errorf("Wrong match. expect RecentSaturated: %v, got: %v", tt.expectRecentSaturated, res.RecentSaturated)
			}
		})
	}
}

// Test_ReconcileOrphanedFlows_saturationComputedBeforeTimeout exercises the
// design-round-4 fix: Saturated/RecentSaturated must be computed in a
// dedicated, RPC-free pass before the RPC loop begins, not interleaved with
// it -- otherwise the self-imposed pass timeout could cut the count short
// of scanLimit even on a genuinely saturated batch. Uses a
// reconcilePassTimeout long enough to survive CampaignListDeletedSince (an
// instant mock call) but short enough to expire partway through the RPC
// loop -- never a 0-duration timeout, which would fail the query itself
// before there is any ReconcileResult to assert against.
func Test_ReconcileOrphanedFlows_saturationComputedBeforeTimeout(t *testing.T) {
	origScanLimit := scanLimit
	origTimeout := reconcilePassTimeout
	scanLimit = 2
	reconcilePassTimeout = 30 * time.Millisecond
	defer func() {
		scanLimit = origScanLimit
		reconcilePassTimeout = origTimeout
	}()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	candidates := []*campaign.Campaign{
		{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111107")},
			FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222207"),
			TMDelete: timePtr(now.Add(-10 * time.Minute)),
		},
		{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111108")},
			FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222208"),
			TMDelete: timePtr(now.Add(-20 * time.Minute)),
		},
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return(candidates, nil)
	// The first candidate's FlowV1FlowGet sleeps past reconcilePassTimeout,
	// so the second candidate never gets its RPC issued -- Saturated and
	// RecentSaturated must already reflect the full 2-row batch, since they
	// were computed before this call, not during the loop.
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*fmflow.Flow, error) {
			time.Sleep(60 * time.Millisecond)
			return &fmflow.Flow{TMDelete: timePtr(now)}, nil
		},
	).Times(1)

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !res.Saturated {
		t.Errorf("Wrong match. expect Saturated: true, got: false")
	}
	if !res.RecentSaturated {
		t.Errorf("Wrong match. expect RecentSaturated: true, got: false")
	}
	if !res.Partial {
		t.Errorf("Wrong match. expect Partial: true, got: false")
	}
}

func Test_ReconcileOrphanedFlows_recentIntervalFallback(t *testing.T) {
	tests := []int64{0, -1, -100}

	for _, recentIntervalSec := range tests {
		t.Run(fmt.Sprintf("recentIntervalSec_%d", recentIntervalSec), func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

			now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
			c := &campaign.Campaign{
				Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
				FlowID:   uuid.Must(uuid.NewV4()),
				TMDelete: timePtr(now.Add(-2 * time.Hour)),
			}

			mockUtil.EXPECT().TimeNow().Return(&now)
			mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
			mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: timePtr(now)}, nil)

			hadWarn, cleanup := captureWarnLogs()
			defer cleanup()

			res, err := h.ReconcileOrphanedFlows(context.Background(), recentIntervalSec)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.Skipped != 1 {
				t.Errorf("Wrong match. expect Skipped: 1, got: %d", res.Skipped)
			}
			if !hadWarn() {
				t.Errorf("Wrong match. expect a warn log for non-positive recentIntervalSec, got none")
			}
		})
	}
}

func Test_ReconcileOrphanedFlows_partialOnSelfTimeout(t *testing.T) {
	origTimeout := reconcilePassTimeout
	reconcilePassTimeout = 30 * time.Millisecond
	defer func() { reconcilePassTimeout = origTimeout }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	candidates := []*campaign.Campaign{
		{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111109")},
			FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222209"),
			TMDelete: timePtr(now.Add(-time.Hour)),
		},
		{
			Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111110")},
			FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222210"),
			TMDelete: timePtr(now.Add(-time.Hour)),
		},
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return(candidates, nil)
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*fmflow.Flow, error) {
			time.Sleep(60 * time.Millisecond)
			return &fmflow.Flow{TMDelete: timePtr(now)}, nil
		},
	).Times(1)

	hadWarn, cleanup := captureWarnLogs()
	defer cleanup()

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !res.Partial {
		t.Errorf("Wrong match. expect Partial: true, got: false")
	}
	if res.Skipped != 1 {
		t.Errorf("Wrong match. expect Skipped: 1 (only the first candidate was processed), got: %d", res.Skipped)
	}
	if !hadWarn() {
		t.Errorf("Wrong match. expect a warn log for the self-imposed timeout bail-out, got none")
	}
}

func Test_ReconcileOrphanedFlows_emptySet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{}, nil)
	// no FlowV1FlowGet/FlowV1FlowDelete EXPECT() -- zero RPCs expected

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := campaign.ReconcileResult{}
	if res != expect {
		t.Errorf("Wrong match. expect: %+v, got: %+v", expect, res)
	}
}

func Test_ReconcileOrphanedFlows_idempotentRerun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222211"),
		TMDelete: timePtr(now.Add(-time.Hour)),
	}

	mockUtil.EXPECT().TimeNow().Return(&now).Times(2)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil).Times(2)

	gomock.InOrder(
		// first pass: flow is live -> genuinely orphaned, cleaned
		mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: nil}, nil),
		mockReq.EXPECT().FlowV1FlowDelete(gomock.Any(), c.FlowID).Return(&fmflow.Flow{}, nil),
		// second pass: flow is now deleted -> skipped, no second delete call
		mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: timePtr(now)}, nil),
	)

	res1, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res1.Cleaned != 1 {
		t.Errorf("Wrong match. expect first-pass Cleaned: 1, got: %d", res1.Cleaned)
	}

	res2, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res2.Cleaned != 0 || res2.Skipped != 1 {
		t.Errorf("Wrong match. expect second-pass Cleaned=0 Skipped=1 (no re-delete), got: %+v", res2)
	}
}

func Test_ReconcileOrphanedFlows_windowEdgeWarning(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &campaignHandler{util: mockUtil, db: mockDB, reqHandler: mockReq}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	// deleted just inside the window's outer edge (well within the 10%
	// margin from the cutoff) -- must trigger the window-edge warning.
	c := &campaign.Campaign{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111112")},
		FlowID:   uuid.FromStringOrNil("22222222-2222-2222-2222-222222222212"),
		TMDelete: timePtr(now.Add(-window + time.Minute)),
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().CampaignListDeletedSince(gomock.Any(), gomock.Any(), scanLimit).Return([]*campaign.Campaign{c}, nil)
	mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), c.FlowID).Return(&fmflow.Flow{TMDelete: nil}, nil)
	mockReq.EXPECT().FlowV1FlowDelete(gomock.Any(), c.FlowID).Return(&fmflow.Flow{}, nil)

	hadWarn, cleanup := captureWarnLogs()
	defer cleanup()

	res, err := h.ReconcileOrphanedFlows(context.Background(), 3600)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Cleaned != 1 {
		t.Errorf("Wrong match. expect Cleaned: 1, got: %d", res.Cleaned)
	}
	if !hadWarn() {
		t.Errorf("Wrong match. expect a window-edge warn log, got none")
	}
}
