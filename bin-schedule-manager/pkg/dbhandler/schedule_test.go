package dbhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
)

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewHandler(dbTest, mockCache)
	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

func Test_ScheduleCreate(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 809000000, time.UTC); return &t }()

	tests := []struct {
		name     string
		schedule *schedule.Schedule

		responseCurTime *time.Time
		expectRes       *schedule.Schedule
	}{
		{
			name: "normal",
			schedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("81b8b09c-6f11-11f0-a001-0242ac110002"),
					CustomerID: uuid.FromStringOrNil("81b8b394-6f11-11f0-a002-0242ac110002"),
				},
				Name:   "number-renew",
				Detail: "renew platform numbers",

				Type: schedule.TypeRPC,
				Cron: "0 1 * * *",

				TargetQueue:    "bin-manager.number-manager.request",
				TargetURI:      "/v1/numbers/renew",
				TargetMethod:   "POST",
				TargetDataType: "application/json",
				TargetData:     json.RawMessage(`{"days":28}`),

				TimeoutMS: 300000,
				RetryMax:  0,
				Enabled:   false,
			},

			responseCurTime: curTime,
			expectRes: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("81b8b09c-6f11-11f0-a001-0242ac110002"),
					CustomerID: uuid.FromStringOrNil("81b8b394-6f11-11f0-a002-0242ac110002"),
				},
				Name:   "number-renew",
				Detail: "renew platform numbers",

				Type: schedule.TypeRPC,
				Cron: "0 1 * * *",

				TargetQueue:    "bin-manager.number-manager.request",
				TargetURI:      "/v1/numbers/renew",
				TargetMethod:   "POST",
				TargetDataType: "application/json",
				TargetData:     json.RawMessage(`{"days":28}`),

				TimeoutMS: 300000,
				RetryMax:  0,
				Enabled:   false,

				TMCreate: curTime,
				TMUpdate: curTime,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
			if err := h.ScheduleCreate(ctx, tt.schedule); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ScheduleGet(ctx, tt.schedule.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
			res, err := h.ScheduleGet(ctx, tt.schedule.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ScheduleList(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 809000000, time.UTC); return &t }()
	customerID := uuid.FromStringOrNil("9c0a2c60-6f11-11f0-a003-0242ac110002")

	schedules := []*schedule.Schedule{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("9c0a3020-6f11-11f0-a004-0242ac110002"),
				CustomerID: customerID,
			},
			Name: "list-name1",
			Type: schedule.TypeRPC,
			Cron: "0 1 * * *",
		},
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("9c0a3336-6f11-11f0-a005-0242ac110002"),
				CustomerID: customerID,
			},
			Name: "list-name2",
			Type: schedule.TypeRPC,
			Cron: "0 2 * * *",
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	for _, s := range schedules {
		mockUtil.EXPECT().TimeNow().Return(curTime)
		mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
		if err := h.ScheduleCreate(ctx, s); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}

	filters := map[schedule.Field]any{
		schedule.FieldCustomerID: customerID,
		schedule.FieldDeleted:    false,
	}

	res, err := h.ScheduleList(ctx, 10, utilhandler.TimeGetCurTime(), filters)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", len(res))
	}

	// empty result must be an empty slice, never nil
	emptyFilters := map[schedule.Field]any{
		schedule.FieldCustomerID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		schedule.FieldDeleted:    false,
	}
	resEmpty, err := h.ScheduleList(ctx, 10, utilhandler.TimeGetCurTime(), emptyFilters)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if resEmpty == nil {
		t.Errorf("Expected non-nil empty slice, got nil")
	}
	if len(resEmpty) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(resEmpty))
	}
}

func Test_ScheduleUpdate(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 809000000, time.UTC); return &t }()
	updateTime := func() *time.Time { t := time.Date(2026, 7, 1, 11, 0, 0, 809000000, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b2f0b09c-6f11-11f0-a006-0242ac110002"),
			CustomerID: uuid.FromStringOrNil("b2f0b394-6f11-11f0-a007-0242ac110002"),
		},
		Name: "update-name",
		Type: schedule.TypeRPC,
		Cron: "0 1 * * *",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleCreate(ctx, s); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	fields := map[schedule.Field]any{
		schedule.FieldName:   "update-name-changed",
		schedule.FieldDetail: "changed detail",
		schedule.FieldCron:   "30 4 * * *",
	}

	mockUtil.EXPECT().TimeNow().Return(updateTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleUpdate(ctx, s.ID, fields); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().ScheduleGet(ctx, s.ID).Return(nil, fmt.Errorf(""))
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	res, err := h.ScheduleGet(ctx, s.ID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.Name != "update-name-changed" || res.Detail != "changed detail" || res.Cron != "30 4 * * *" {
		t.Errorf("Wrong match. got: %v", res)
	}
	if !reflect.DeepEqual(res.TMUpdate, updateTime) {
		t.Errorf("Wrong match. expect: %v, got: %v", updateTime, res.TMUpdate)
	}
}

func Test_ScheduleDelete(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 809000000, time.UTC); return &t }()
	deleteTime := func() *time.Time { t := time.Date(2026, 7, 1, 12, 0, 0, 809000000, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c5e0b09c-6f11-11f0-a008-0242ac110002"),
			CustomerID: uuid.FromStringOrNil("c5e0b394-6f11-11f0-a009-0242ac110002"),
		},
		Name: "delete-name",
		Type: schedule.TypeRPC,
		Cron: "0 1 * * *",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleCreate(ctx, s); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(deleteTime)
	mockCache.EXPECT().ScheduleDelete(ctx, s.ID)
	if err := h.ScheduleDelete(ctx, s.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockCache.EXPECT().ScheduleGet(ctx, s.ID).Return(nil, fmt.Errorf(""))
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	res, err := h.ScheduleGet(ctx, s.ID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if !reflect.DeepEqual(res.TMDelete, deleteTime) {
		t.Errorf("Wrong match. expect: %v, got: %v", deleteTime, res.TMDelete)
	}
}

func Test_ScheduleGetByCustomerIDName(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 809000000, time.UTC); return &t }()
	customerID := uuid.FromStringOrNil("d7c0b09c-6f11-11f0-a00a-0242ac110002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	// deleted schedule with the target name — must NOT be returned
	sDeleted := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d7c0b394-6f11-11f0-a00b-0242ac110002"),
			CustomerID: customerID,
		},
		Name: "byname",
		Type: schedule.TypeRPC,
		Cron: "0 1 * * *",
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleCreate(ctx, sDeleted); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleDelete(ctx, sDeleted.ID)
	if err := h.ScheduleDelete(ctx, sDeleted.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// with only the deleted row present, active lookup must return ErrNotFound
	if _, err := h.ScheduleGetByCustomerIDName(ctx, customerID, "byname"); err != ErrNotFound {
		t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
	}

	// active schedule with the same name (name reuse after soft delete)
	sActive := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d7c0b678-6f11-11f0-a00c-0242ac110002"),
			CustomerID: customerID,
		},
		Name: "byname",
		Type: schedule.TypeRPC,
		Cron: "0 2 * * *",
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleCreate(ctx, sActive); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.ScheduleGetByCustomerIDName(ctx, customerID, "byname")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != sActive.ID {
		t.Errorf("Wrong match. expect: %v, got: %v", sActive.ID, res.ID)
	}
}

// scheduleIDSet extracts the id set from a schedule list.
func scheduleIDSet(schedules []*schedule.Schedule) map[uuid.UUID]bool {
	res := map[uuid.UUID]bool{}
	for _, s := range schedules {
		res[s.ID] = true
	}
	return res
}

func Test_ScheduleListDue(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC); return &t }()
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	customerID := uuid.FromStringOrNil("e8a0b09c-6f11-11f0-a00d-0242ac110002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	idDuePast := uuid.FromStringOrNil("e8a0b101-6f11-11f0-a00e-0242ac110002")
	idDueExact := uuid.FromStringOrNil("e8a0b202-6f11-11f0-a00f-0242ac110002")
	idFuture := uuid.FromStringOrNil("e8a0b303-6f11-11f0-a010-0242ac110002")
	idDisabled := uuid.FromStringOrNil("e8a0b404-6f11-11f0-a011-0242ac110002")
	idNoNextRun := uuid.FromStringOrNil("e8a0b505-6f11-11f0-a012-0242ac110002")
	idDeleted := uuid.FromStringOrNil("e8a0b606-6f11-11f0-a013-0242ac110002")

	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	type seed struct {
		id      uuid.UUID
		enabled bool
		nextRun *time.Time
		deleted bool
	}
	seeds := []seed{
		{id: idDuePast, enabled: true, nextRun: &past},
		{id: idDueExact, enabled: true, nextRun: &now}, // boundary: tm_next_run == now is due
		{id: idFuture, enabled: true, nextRun: &future},
		{id: idDisabled, enabled: false, nextRun: &past},
		{id: idNoNextRun, enabled: true, nextRun: nil},
		{id: idDeleted, enabled: true, nextRun: &past, deleted: true},
	}

	for i, sd := range seeds {
		s := &schedule.Schedule{
			Identity: commonidentity.Identity{
				ID:         sd.id,
				CustomerID: customerID,
			},
			Name:      fmt.Sprintf("due-%d", i),
			Type:      schedule.TypeRPC,
			Cron:      "0 1 * * *",
			Enabled:   sd.enabled,
			TMNextRun: sd.nextRun,
		}

		mockUtil.EXPECT().TimeNow().Return(curTime)
		mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
		if err := h.ScheduleCreate(ctx, s); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}

		if sd.deleted {
			mockUtil.EXPECT().TimeNow().Return(curTime)
			mockCache.EXPECT().ScheduleDelete(ctx, sd.id)
			if err := h.ScheduleDelete(ctx, sd.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		}
	}

	res, err := h.ScheduleListDue(ctx, now, 100)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	got := scheduleIDSet(res)
	for _, id := range []uuid.UUID{idDuePast, idDueExact} {
		if !got[id] {
			t.Errorf("Expected due schedule missing. id: %v", id)
		}
	}
	for _, id := range []uuid.UUID{idFuture, idDisabled, idNoNextRun, idDeleted} {
		if got[id] {
			t.Errorf("Unexpected schedule in due list. id: %v", id)
		}
	}
}

func Test_ScheduleListUninitialized(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC); return &t }()
	next := time.Date(2026, 7, 2, 1, 0, 0, 0, time.UTC)
	customerID := uuid.FromStringOrNil("f1a0b09c-6f11-11f0-a014-0242ac110002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	idUninit := uuid.FromStringOrNil("f1a0b101-6f11-11f0-a015-0242ac110002")
	idInitialized := uuid.FromStringOrNil("f1a0b202-6f11-11f0-a016-0242ac110002")
	idDisabled := uuid.FromStringOrNil("f1a0b303-6f11-11f0-a017-0242ac110002")

	type seed struct {
		id      uuid.UUID
		enabled bool
		nextRun *time.Time
	}
	seeds := []seed{
		{id: idUninit, enabled: true, nextRun: nil},
		{id: idInitialized, enabled: true, nextRun: &next},
		{id: idDisabled, enabled: false, nextRun: nil},
	}

	for i, sd := range seeds {
		s := &schedule.Schedule{
			Identity: commonidentity.Identity{
				ID:         sd.id,
				CustomerID: customerID,
			},
			Name:      fmt.Sprintf("uninit-%d", i),
			Type:      schedule.TypeRPC,
			Cron:      "0 1 * * *",
			Enabled:   sd.enabled,
			TMNextRun: sd.nextRun,
		}

		mockUtil.EXPECT().TimeNow().Return(curTime)
		mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
		if err := h.ScheduleCreate(ctx, s); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}

	res, err := h.ScheduleListUninitialized(ctx, 100)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	got := scheduleIDSet(res)
	if !got[idUninit] {
		t.Errorf("Expected uninitialized schedule missing. id: %v", idUninit)
	}
	for _, id := range []uuid.UUID{idInitialized, idDisabled} {
		if got[id] {
			t.Errorf("Unexpected schedule in uninitialized list. id: %v", id)
		}
	}
}

func Test_ScheduleInitNextRun(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC); return &t }()
	next := time.Date(2026, 7, 2, 1, 0, 0, 0, time.UTC)
	nextOther := time.Date(2026, 7, 3, 1, 0, 0, 0, time.UTC)
	customerID := uuid.FromStringOrNil("0aa0b09c-6f12-11f0-a018-0242ac110002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("0aa0b101-6f12-11f0-a019-0242ac110002"),
			CustomerID: customerID,
		},
		Name:    "init-next-run",
		Type:    schedule.TypeRPC,
		Cron:    "0 1 * * *",
		Enabled: true,
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleCreate(ctx, s); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// success: tm_next_run NULL → set
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	ok, err := h.ScheduleInitNextRun(ctx, s.ID, next)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !ok {
		t.Errorf("Wrong match. expect: true, got: false")
	}

	res, err := h.scheduleGetFromDB(ctx, s.ID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if res.TMNextRun == nil || !res.TMNextRun.Equal(next) {
		t.Errorf("Wrong match. expect: %v, got: %v", next, res.TMNextRun)
	}
	if res.TMLastRun != nil {
		t.Errorf("ScheduleInitNextRun must not touch tm_last_run. got: %v", res.TMLastRun)
	}

	// already set: CAS must lose
	mockUtil.EXPECT().TimeNow().Return(curTime)
	ok, err = h.ScheduleInitNextRun(ctx, s.ID, nextOther)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}

	// disabled schedule: must not initialize
	sDisabled := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("0aa0b202-6f12-11f0-a01a-0242ac110002"),
			CustomerID: customerID,
		},
		Name:    "init-next-run-disabled",
		Type:    schedule.TypeRPC,
		Cron:    "0 1 * * *",
		Enabled: false,
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().ScheduleSet(ctx, gomock.Any())
	if err := h.ScheduleCreate(ctx, sDisabled); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	ok, err = h.ScheduleInitNextRun(ctx, sDisabled.ID, next)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}
}
