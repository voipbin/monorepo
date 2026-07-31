package schedulehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

func Test_ExecutionGets(t *testing.T) {
	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[execution.Field]any

		responseExecutions []*execution.Execution
	}{
		{
			name: "normal",

			size:  10,
			token: "2026-08-01 00:00:00.000000",
			filters: map[execution.Field]any{
				execution.FieldScheduleID: uuid.FromStringOrNil("32f0a89e-6f1b-11f0-9c60-1f8f8f4bafcd"),
				execution.FieldStatus:     execution.StatusSuccess,
			},

			responseExecutions: []*execution.Execution{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("33b0e2fa-6f1b-11f0-8b9f-7f3ab5a5c2ae"),
					},
					ScheduleID: uuid.FromStringOrNil("32f0a89e-6f1b-11f0-9c60-1f8f8f4bafcd"),
					Status:     execution.StatusSuccess,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &scheduleHandler{
				utilHandler: mockUtil,
				db:          mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ExecutionList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseExecutions, nil)

			res, err := h.ExecutionGets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseExecutions) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExecutions, res)
			}
		})
	}
}

func Test_ExecutionGets_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &scheduleHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}

	ctx := context.Background()
	filters := map[execution.Field]any{}

	mockDB.EXPECT().ExecutionList(ctx, uint64(10), "token", filters).Return(nil, fmt.Errorf("db error"))

	if _, err := h.ExecutionGets(ctx, 10, "token", filters); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_ExecutionPrune(t *testing.T) {
	now := time.Date(2026, 8, 1, 3, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		retentionDays int

		responseBatches []int64

		expectCutoff time.Time
		expectRes    int64
	}{
		{
			name: "multiple batches",

			retentionDays: 90,

			responseBatches: []int64{1000, 250, 0},

			expectCutoff: now.AddDate(0, 0, -90),
			expectRes:    1250,
		},
		{
			name: "nothing to prune",

			retentionDays: 30,

			responseBatches: []int64{0},

			expectCutoff: now.AddDate(0, 0, -30),
			expectRes:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &scheduleHandler{
				utilHandler: mockUtil,
				db:          mockDB,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&now)
			for _, batch := range tt.responseBatches {
				mockDB.EXPECT().ExecutionPrune(ctx, tt.expectCutoff, uint64(pruneBatchSize)).Return(batch, nil)
			}

			res, err := h.ExecutionPrune(ctx, tt.retentionDays)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %d\ngot: %d", tt.expectRes, res)
			}
		})
	}
}

func Test_ExecutionPrune_error(t *testing.T) {
	now := time.Date(2026, 8, 1, 3, 0, 0, 0, time.UTC)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &scheduleHandler{
		utilHandler: mockUtil,
		db:          mockDB,
	}

	ctx := context.Background()
	cutoff := now.AddDate(0, 0, -90)

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().ExecutionPrune(ctx, cutoff, uint64(pruneBatchSize)).Return(int64(0), fmt.Errorf("db error"))

	if _, err := h.ExecutionPrune(ctx, 90); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}
