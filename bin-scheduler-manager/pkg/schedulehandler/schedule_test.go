package schedulehandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {
	tests := []struct {
		name string

		customerID     uuid.UUID
		scheduleName   string
		detail         string
		cronExpr       string
		targetQueue    string
		targetURI      string
		targetMethod   string
		targetDataType string
		targetData     json.RawMessage
		timeoutMS      int
		retryMax       int
		enabled        bool

		responseUUID     uuid.UUID
		responseSchedule *schedule.Schedule

		expectSchedule *schedule.Schedule
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName:   "number-renew",
			detail:         "daily number renew",
			cronExpr:       "0 1 * * *",
			targetQueue:    "bin-manager.number-manager.request",
			targetURI:      "/v1/numbers/renew",
			targetMethod:   "POST",
			targetDataType: "application/json",
			targetData:     json.RawMessage(`{"days":28}`),
			timeoutMS:      300000,
			retryMax:       0,
			enabled:        true,

			responseUUID: uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d"),
			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d"),
					CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
				},
				Name: "number-renew",
			},

			expectSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d"),
					CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
				},

				Name:   "number-renew",
				Detail: "daily number renew",

				Type: schedule.TypeRPC,
				Cron: "0 1 * * *",

				TargetQueue:    "bin-manager.number-manager.request",
				TargetURI:      "/v1/numbers/renew",
				TargetMethod:   "POST",
				TargetDataType: "application/json",
				TargetData:     json.RawMessage(`{"days":28}`),

				TimeoutMS: 300000,
				RetryMax:  0,
				Enabled:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &scheduleHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ScheduleGetByCustomerIDName(ctx, tt.customerID, tt.scheduleName).Return(nil, dbhandler.ErrNotFound)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ScheduleCreate(ctx, tt.expectSchedule).Return(nil)
			mockDB.EXPECT().ScheduleGet(ctx, tt.responseUUID).Return(tt.responseSchedule, nil)
			mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeScheduleCreated, tt.responseSchedule)

			res, err := h.Create(ctx, tt.customerID, tt.scheduleName, tt.detail, tt.cronExpr, tt.targetQueue, tt.targetURI, tt.targetMethod, tt.targetDataType, tt.targetData, tt.timeoutMS, tt.retryMax, tt.enabled)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseSchedule) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseSchedule, res)
			}
		})
	}
}

func Test_Create_error(t *testing.T) {
	tests := []struct {
		name string

		customerID   uuid.UUID
		scheduleName string
		cronExpr     string
		targetQueue  string
		targetMethod string

		timeoutMS  int
		retryMax   int
		targetData []byte

		responseExisting    *schedule.Schedule
		responseExistingErr error

		expectStatus cerrors.Status
	}{
		{
			name: "non-positive timeout",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",
			timeoutMS:    -5,

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "name exceeds column width",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: strings.Repeat("a", 256),
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "invalid target data json",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",
			targetData:   []byte("{not json"),

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "negative retry_max",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",
			retryMax:     -1,

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "empty name",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "unparseable cron",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "not a cron",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "never matching cron",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "0 0 30 2 *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "invalid method",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "PATCH",

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "unknown target queue",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "test-schedule",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.nonexistent-manager.request",
			targetMethod: "POST",

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "duplicated name",

			customerID:   uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName: "number-renew",
			cronExpr:     "0 1 * * *",
			targetQueue:  "bin-manager.number-manager.request",
			targetMethod: "POST",

			responseExisting: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b39d0c2a-19f6-11ef-9c25-d7ba9f2c4c62"),
				},
				Name: "number-renew",
			},

			expectStatus: cerrors.StatusAlreadyExists,
		},
		{
			name: "name uniqueness check db error",

			customerID:          uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
			scheduleName:        "test-schedule",
			cronExpr:            "0 1 * * *",
			targetQueue:         "bin-manager.number-manager.request",
			targetMethod:        "POST",
			responseExistingErr: stderrors.New("db down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &scheduleHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			if tt.responseExisting != nil || tt.responseExistingErr != nil {
				mockDB.EXPECT().ScheduleGetByCustomerIDName(ctx, tt.customerID, tt.scheduleName).Return(tt.responseExisting, tt.responseExistingErr)
			}

			timeoutMS := tt.timeoutMS
			if timeoutMS == 0 {
				timeoutMS = 30000
			}

			_, err := h.Create(ctx, tt.customerID, tt.scheduleName, "", tt.cronExpr, tt.targetQueue, "/v1/test", tt.targetMethod, "application/json", tt.targetData, timeoutMS, tt.retryMax, true)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
				return
			}

			if tt.expectStatus != "" {
				var vErr *cerrors.VoipbinError
				if !stderrors.As(err, &vErr) {
					t.Errorf("Wrong match. expect: VoipbinError, got: %v", err)
					return
				}
				if vErr.Status != tt.expectStatus {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, vErr.Status)
				}
			}
		})
	}
}

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseSchedule *schedule.Schedule
		responseErr      error

		expectErr    bool
		expectStatus cerrors.Status
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("2e97a6f8-1a01-11ef-9db8-2f14fe4c832a"),

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2e97a6f8-1a01-11ef-9db8-2f14fe4c832a"),
				},
			},
		},
		{
			name: "not found",

			id: uuid.FromStringOrNil("3f2b81aa-1a01-11ef-a3c3-cbb59a5da4de"),

			responseErr: dbhandler.ErrNotFound,

			expectErr:    true,
			expectStatus: cerrors.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &scheduleHandler{
				db: mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(tt.responseSchedule, tt.responseErr)

			res, err := h.Get(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
					return
				}
				var vErr *cerrors.VoipbinError
				if !stderrors.As(err, &vErr) {
					t.Errorf("Wrong match. expect: VoipbinError, got: %v", err)
					return
				}
				if vErr.Status != tt.expectStatus {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, vErr.Status)
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseSchedule) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseSchedule, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {
	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[schedule.Field]any

		responseSchedules []*schedule.Schedule
	}{
		{
			name: "normal",

			size:  10,
			token: "2026-08-01 00:00:00.000000",
			filters: map[schedule.Field]any{
				schedule.FieldDeleted: false,
			},

			responseSchedules: []*schedule.Schedule{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("57e3a4de-1a02-11ef-8ff5-931db3286d2e"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &scheduleHandler{
				db: mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ScheduleList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseSchedules, nil)

			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseSchedules) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseSchedules, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {
	curSchedule := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
		},
		Name: "number-renew",
	}

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[schedule.Field]any

		expectFields map[schedule.Field]any
	}{
		{
			name: "cron change resets tm_next_run",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldCron: "0 2 * * *",
			},

			expectFields: map[schedule.Field]any{
				schedule.FieldCron:      "0 2 * * *",
				schedule.FieldTMNextRun: nil,
			},
		},
		{
			name: "enabled change resets tm_next_run",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldEnabled: true,
			},

			expectFields: map[schedule.Field]any{
				schedule.FieldEnabled:   true,
				schedule.FieldTMNextRun: nil,
			},
		},
		{
			name: "detail only change keeps tm_next_run",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldDetail: "updated detail",
			},

			expectFields: map[schedule.Field]any{
				schedule.FieldDetail: "updated detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &scheduleHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			responseSchedule := &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: tt.id,
				},
			}

			mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(curSchedule, nil)
			mockDB.EXPECT().ScheduleUpdate(ctx, tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(responseSchedule, nil)
			mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeScheduleUpdated, responseSchedule)

			res, err := h.Update(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, responseSchedule) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", responseSchedule, res)
			}
		})
	}
}

func Test_Update_error(t *testing.T) {
	curSchedule := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
		},
		Name: "number-renew",
	}

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[schedule.Field]any

		expectNameLookup bool
		responseExisting *schedule.Schedule

		expectStatus cerrors.Status
	}{
		{
			name: "invalid cron",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldCron: "not a cron",
			},

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "invalid method",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldTargetMethod: "PATCH",
			},

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "unknown target queue",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldTargetQueue: "bin-manager.nonexistent-manager.request",
			},

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "type change rejected",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldType: "flow",
			},

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "non-positive timeout",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldTimeoutMS: 0,
			},

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "negative retry_max from json float",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldRetryMax: float64(-1),
			},

			expectStatus: cerrors.StatusInvalidArgument,
		},
		{
			name: "duplicated name",

			id: uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8"),
			fields: map[schedule.Field]any{
				schedule.FieldName: "database-backup",
			},

			expectNameLookup: true,
			responseExisting: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b39d0c2a-19f6-11ef-9c25-d7ba9f2c4c62"),
				},
				Name: "database-backup",
			},

			expectStatus: cerrors.StatusAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &scheduleHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(curSchedule, nil)
			if tt.expectNameLookup {
				mockDB.EXPECT().ScheduleGetByCustomerIDName(ctx, curSchedule.CustomerID, tt.fields[schedule.FieldName]).Return(tt.responseExisting, nil)
			}

			_, err := h.Update(ctx, tt.id, tt.fields)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
				return
			}

			var vErr *cerrors.VoipbinError
			if !stderrors.As(err, &vErr) {
				t.Errorf("Wrong match. expect: VoipbinError, got: %v", err)
				return
			}
			if vErr.Status != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, vErr.Status)
			}
		})
	}
}

func Test_Update_name_to_self_allowed(t *testing.T) {
	id := uuid.FromStringOrNil("6f24b566-1a03-11ef-9be0-cf12a2d0a5a8")
	curSchedule := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
		},
		Name: "number-renew",
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &scheduleHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	fields := map[schedule.Field]any{
		schedule.FieldName: "number-renew",
	}

	mockDB.EXPECT().ScheduleGet(ctx, id).Return(curSchedule, nil)
	mockDB.EXPECT().ScheduleGetByCustomerIDName(ctx, curSchedule.CustomerID, "number-renew").Return(curSchedule, nil)
	mockDB.EXPECT().ScheduleUpdate(ctx, id, fields).Return(nil)
	mockDB.EXPECT().ScheduleGet(ctx, id).Return(curSchedule, nil)
	mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeScheduleUpdated, curSchedule)

	if _, err := h.Update(ctx, id, fields); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseGetErr error

		expectErr    bool
		expectStatus cerrors.Status
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("9d5b7c9c-1a05-11ef-a06f-7f2b6f7b1a3e"),
		},
		{
			name: "not found",

			id: uuid.FromStringOrNil("a8f70c3e-1a05-11ef-b1cd-eb0e5be2f2a4"),

			responseGetErr: dbhandler.ErrNotFound,

			expectErr:    true,
			expectStatus: cerrors.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &scheduleHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			responseSchedule := &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: tt.id,
				},
			}

			if tt.responseGetErr != nil {
				mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(nil, tt.responseGetErr)
			} else {
				mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(responseSchedule, nil)
				mockDB.EXPECT().ScheduleDelete(ctx, tt.id).Return(nil)
				mockDB.EXPECT().ScheduleGet(ctx, tt.id).Return(responseSchedule, nil)
				mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeScheduleDeleted, responseSchedule)
			}

			res, err := h.Delete(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: err, got: ok")
					return
				}
				var vErr *cerrors.VoipbinError
				if !stderrors.As(err, &vErr) {
					t.Errorf("Wrong match. expect: VoipbinError, got: %v", err)
					return
				}
				if vErr.Status != tt.expectStatus {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, vErr.Status)
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, responseSchedule) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", responseSchedule, res)
			}
		})
	}
}
