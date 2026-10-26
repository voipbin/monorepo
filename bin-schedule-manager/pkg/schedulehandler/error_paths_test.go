package schedulehandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

var errSynthetic = stderrors.New("synthetic failure")

func newErrorPathsHandler(mc *gomock.Controller) (*scheduleHandler, *dbhandler.MockDBHandler, *utilhandler.MockUtilHandler, *notifyhandler.MockNotifyHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &scheduleHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	return h, mockDB, mockUtil, mockNotify
}

func Test_Create_db_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")
	newID := uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d")

	tests := []struct {
		name string

		createErr error
		getErr    error
	}{
		{
			name:      "create fails",
			createErr: errSynthetic,
		},
		{
			name:   "read back fails",
			getErr: errSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockUtil, _ := newErrorPathsHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().ScheduleGetByCustomerIDName(ctx, customerID, "number-renew").Return(nil, dbhandler.ErrNotFound)
			mockUtil.EXPECT().UUIDCreate().Return(newID)
			mockDB.EXPECT().ScheduleCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				mockDB.EXPECT().ScheduleGet(ctx, newID).Return(nil, tt.getErr)
			}
			// PublishEvent has no expectation: no event is published on failure

			res, err := h.Create(ctx, customerID, "number-renew", "", "0 1 * * *", "bin-manager.number-manager.request", "/v1/numbers/renew", "POST", "application/json", json.RawMessage(`{}`), 1000, 0, true)
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if !stderrors.Is(err, errSynthetic) {
				t.Errorf("Wrong match. expect: the original error reachable, got: %v", err)
			}
		})
	}
}

func Test_Get_other_error_passes_through(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newErrorPathsHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("57e3a4de-1a02-11ef-8ff5-931db3286d2e")

	mockDB.EXPECT().ScheduleGet(ctx, id).Return(nil, errSynthetic)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if err != errSynthetic {
		t.Errorf("Wrong match. expect: the unwrapped original error, got: %v", err)
	}
	var vErr *cerrors.VoipbinError
	if stderrors.As(err, &vErr) {
		t.Errorf("Wrong match. expect: a plain error (only not-found is mapped), got: %v", vErr)
	}
}

func Test_Gets_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newErrorPathsHandler(mc)
	ctx := context.Background()
	filters := map[schedule.Field]any{schedule.FieldDeleted: false}

	mockDB.EXPECT().ScheduleList(ctx, uint64(10), "token", filters).Return(nil, errSynthetic)

	res, err := h.Gets(ctx, 10, "token", filters)
	if res != nil || err != errSynthetic {
		t.Errorf("Wrong match. expect: nil and the original error, got: %v and %v", res, err)
	}
}

func Test_Update_and_Delete_db_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8")
	cur := &schedule.Schedule{
		Identity: commonidentity.Identity{ID: id},
		Name:     "number-renew",
	}
	fields := map[schedule.Field]any{schedule.FieldDetail: "updated detail"}

	tests := []struct {
		name string

		firstGetErr error
		writeErr    error
		readErr     error
		isUpdate    bool
	}{
		{
			name:        "update: existence check fails",
			isUpdate:    true,
			firstGetErr: errSynthetic,
		},
		{
			name:     "update: write fails",
			isUpdate: true,
			writeErr: errSynthetic,
		},
		{
			name:     "update: read back fails",
			isUpdate: true,
			readErr:  errSynthetic,
		},
		{
			name:     "delete: write fails",
			writeErr: errSynthetic,
		},
		{
			name:    "delete: read back fails",
			readErr: errSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorPathsHandler(mc)
			ctx := context.Background()

			if tt.firstGetErr != nil {
				mockDB.EXPECT().ScheduleGet(ctx, id).Return(nil, tt.firstGetErr)
			} else {
				mockDB.EXPECT().ScheduleGet(ctx, id).Return(cur, nil)
				if tt.isUpdate {
					mockDB.EXPECT().ScheduleUpdate(ctx, id, gomock.Any()).Return(tt.writeErr)
				} else {
					mockDB.EXPECT().ScheduleDelete(ctx, id).Return(tt.writeErr)
				}
				if tt.writeErr == nil {
					mockDB.EXPECT().ScheduleGet(ctx, id).Return(nil, tt.readErr)
				}
			}
			// PublishEvent has no expectation: no event is published on failure

			var err error
			if tt.isUpdate {
				_, err = h.Update(ctx, id, fields)
			} else {
				_, err = h.Delete(ctx, id)
			}
			if !stderrors.Is(err, errSynthetic) {
				t.Errorf("Wrong match. expect: the original error reachable, got: %v", err)
			}
		})
	}
}

func Test_Update_field_value_types(t *testing.T) {
	id := uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8")
	cur := &schedule.Schedule{
		Identity: commonidentity.Identity{ID: id},
		Name:     "number-renew",
	}

	tests := []struct {
		name string

		fields map[schedule.Field]any

		expectReason string
	}{
		{
			name:         "target uri is not a string",
			fields:       map[schedule.Field]any{schedule.FieldTargetURI: 123},
			expectReason: "SCHEDULE_TARGET_URI_EMPTY",
		},
		{
			name:         "target data as bytes is not valid json",
			fields:       map[schedule.Field]any{schedule.FieldTargetData: []byte(`{bad`)},
			expectReason: "SCHEDULE_TARGET_DATA_INVALID",
		},
		{
			name:         "target data as string is not valid json",
			fields:       map[schedule.Field]any{schedule.FieldTargetData: `{bad`},
			expectReason: "SCHEDULE_TARGET_DATA_INVALID",
		},
		{
			name:         "timeout as int64 zero",
			fields:       map[schedule.Field]any{schedule.FieldTimeoutMS: int64(0)},
			expectReason: "SCHEDULE_TIMEOUT_INVALID",
		},
		{
			name:         "timeout of an unsupported type",
			fields:       map[schedule.Field]any{schedule.FieldTimeoutMS: "1000"},
			expectReason: "SCHEDULE_TIMEOUT_INVALID",
		},
		{
			name:         "retry max as int64 negative",
			fields:       map[schedule.Field]any{schedule.FieldRetryMax: int64(-1)},
			expectReason: "SCHEDULE_RETRY_MAX_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorPathsHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().ScheduleGet(ctx, id).Return(cur, nil)

			_, err := h.Update(ctx, id, tt.fields)

			var vErr *cerrors.VoipbinError
			if !stderrors.As(err, &vErr) {
				t.Fatalf("Wrong match. expect: VoipbinError, got: %v", err)
			}
			if vErr.Status != cerrors.StatusInvalidArgument || vErr.Reason != tt.expectReason {
				t.Errorf("Wrong match. expect: %s/%s, got: %s/%s", cerrors.StatusInvalidArgument, tt.expectReason, vErr.Status, vErr.Reason)
			}
		})
	}
}

func Test_toInt(t *testing.T) {
	tests := []struct {
		name string

		value any

		expectInt int
		expectOK  bool
	}{
		{"int", 7, 7, true},
		{"int64", int64(8), 8, true},
		{"float64 truncates", float64(9.9), 9, true},
		{"string is rejected", "10", 0, false},
		{"nil is rejected", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := toInt(tt.value)
			if res != tt.expectInt || ok != tt.expectOK {
				t.Errorf("Wrong match. expect: %d/%v, got: %d/%v", tt.expectInt, tt.expectOK, res, ok)
			}
		})
	}
}

func Test_EventCustomerDeleted_failed_delete_does_not_stop_the_rest(t *testing.T) {
	customerID := uuid.FromStringOrNil("a2c9b8fe-6f1c-11f0-96b1-df0b0e5cbd7a")
	curTime := "2026-08-01 03:00:00.000000"

	failing := &schedule.Schedule{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a3800b8a-6f1c-11f0-b6bd-93a1b21f5a1e"), CustomerID: customerID},
		Name:     "customer-schedule-failing",
	}
	healthy := &schedule.Schedule{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a3f4b7a2-6f1c-11f0-9d3f-ff8b03bcd0ba"), CustomerID: customerID},
		Name:     "customer-schedule-healthy",
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockUtil, mockNotify := newErrorPathsHandler(mc)
	ctx := context.Background()

	filters := map[schedule.Field]any{
		schedule.FieldCustomerID: customerID,
		schedule.FieldDeleted:    false,
	}
	mockUtil.EXPECT().TimeGetCurTime().Return(curTime)
	mockDB.EXPECT().ScheduleList(ctx, uint64(eventListLimit), curTime, filters).Return([]*schedule.Schedule{failing, healthy}, nil)

	// the first schedule fails at the existence check, so it is never deleted
	mockDB.EXPECT().ScheduleGet(ctx, failing.ID).Return(nil, errSynthetic)

	mockDB.EXPECT().ScheduleGet(ctx, healthy.ID).Return(healthy, nil)
	mockDB.EXPECT().ScheduleDelete(ctx, healthy.ID).Return(nil)
	mockDB.EXPECT().ScheduleGet(ctx, healthy.ID).Return(healthy, nil)
	mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeScheduleDeleted, healthy)

	if err := h.EventCustomerDeleted(ctx, &cmcustomer.Customer{ID: customerID}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
