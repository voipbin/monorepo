package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// newRowsErrRows builds sqlmock rows with the given columns. The first row is
// scannable and the failure is raised when the row at failIndex is read, so
// Next returns false and the cause is only reported by rows.Err().
func newRowsErrRows(columns []string, rowCount int, failIndex int) *sqlmock.Rows {
	rows := sqlmock.NewRows(columns)
	for i := 0; i < rowCount; i++ {
		values := make([]driver.Value, len(columns))
		for j, c := range columns {
			switch c {
			case "id", "customer_id":
				values[j] = uuid.Must(uuid.NewV4()).Bytes()
			case "stack_map", "current_action":
				values[j] = []byte("{}")
			case "executed_actions", "actions":
				values[j] = []byte("[]")
			}
		}
		rows.AddRow(values...)
	}
	return rows.RowError(failIndex, errRowsBroken)
}

func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	flowColumns := commondatabasehandler.GetDBFields(&flow.Flow{})
	activeflowColumns := commondatabasehandler.GetDBFields(&activeflow.Activeflow{})

	tests := []struct {
		name string

		columns   []string
		rowCount  int
		failIndex int
		fn        func(h *handler) (any, error)
	}{
		{"get activeflow", activeflowColumns, 1, 0, func(h *handler) (any, error) {
			return h.activeflowGetFromDB(context.Background(), id)
		}},
		{"get flow", flowColumns, 1, 0, func(h *handler) (any, error) {
			return h.flowGetFromDB(context.Background(), id)
		}},
		{"list activeflow after the first row", activeflowColumns, 2, 1, func(h *handler) (any, error) {
			return h.ActiveflowList(context.Background(), "", 10, nil)
		}},
		{"list flow after the first row", flowColumns, 2, 1, func(h *handler) (any, error) {
			return h.FlowList(context.Background(), "", 10, nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create sqlmock: %v", err)
			}
			defer func() { _ = db.Close() }()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

			h := &handler{
				util:  mockUtil,
				db:    db,
				cache: cachehandler.NewMockCacheHandler(mc),
			}

			mock.ExpectQuery("SELECT").WillReturnRows(newRowsErrRows(tt.columns, tt.rowCount, tt.failIndex))

			res, err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil. res: %v", res)
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			// the cause is wrapped with %v, so it is checked by its text to make
			// sure the failure is the injected one and not a scan failure.
			if !strings.Contains(err.Error(), errRowsBroken.Error()) {
				t.Errorf("expected the injected rows failure. err: %v", err)
			}
			if !isNilResult(res) {
				t.Errorf("no result is expected with the error. res: %v", res)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// Test_ContextCanceled_WriteAborted verifies that a canceled context stops the
// write before it reaches the database and leaves the row untouched.
func Test_ContextCanceled_WriteAborted(t *testing.T) {
	responseCurTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	newHandler := func(t *testing.T) (*handler, *cachehandler.MockCacheHandler, *utilhandler.MockUtilHandler) {
		mc := gomock.NewController(t)
		t.Cleanup(mc.Finish)

		mockUtil := utilhandler.NewMockUtilHandler(mc)
		mockCache := cachehandler.NewMockCacheHandler(mc)
		return &handler{util: mockUtil, db: dbTest, cache: mockCache}, mockCache, mockUtil
	}

	t.Run("ActiveflowUpdate", func(t *testing.T) {
		h, mockCache, mockUtil := newHandler(t)
		setupCtx := context.Background()

		af := &activeflow.Activeflow{}
		af.ID = uuid.Must(uuid.NewV4())

		mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
		mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
		if err := h.ActiveflowCreate(setupCtx, af); err != nil {
			t.Fatalf("could not create the activeflow: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
		err := h.ActiveflowUpdate(ctx, af.ID, map[activeflow.Field]any{activeflow.FieldExecuteCount: 7})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}

		got, errGet := h.activeflowGetFromDB(setupCtx, af.ID)
		if errGet != nil {
			t.Fatalf("could not read the row: %v", errGet)
		}
		if got.ExecuteCount != 0 {
			t.Errorf("the row must not be updated. execute_count: %d", got.ExecuteCount)
		}
	})

	t.Run("FlowUpdate", func(t *testing.T) {
		h, mockCache, mockUtil := newHandler(t)
		setupCtx := context.Background()

		f := &flow.Flow{Name: "before"}
		f.ID = uuid.Must(uuid.NewV4())

		mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
		mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
		if err := h.FlowCreate(setupCtx, f); err != nil {
			t.Fatalf("could not create the flow: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
		err := h.FlowUpdate(ctx, f.ID, map[flow.Field]any{flow.FieldName: "after"})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}

		got, errGet := h.flowGetFromDB(setupCtx, f.ID)
		if errGet != nil {
			t.Fatalf("could not read the row: %v", errGet)
		}
		if got.Name != "before" {
			t.Errorf("the row must not be updated. name: %s", got.Name)
		}
	})

	// FlowDelete wraps the cause with %v, so errors.Is can not be asserted.
	t.Run("FlowDelete", func(t *testing.T) {
		h, mockCache, mockUtil := newHandler(t)
		setupCtx := context.Background()

		f := &flow.Flow{Name: "to be kept"}
		f.ID = uuid.Must(uuid.NewV4())

		mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
		mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
		if err := h.FlowCreate(setupCtx, f); err != nil {
			t.Fatalf("could not create the flow: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
		if err := h.FlowDelete(ctx, f.ID); err == nil {
			t.Fatalf("expected an error, got nil")
		}

		got, errGet := h.flowGetFromDB(setupCtx, f.ID)
		if errGet != nil {
			t.Fatalf("could not read the row: %v", errGet)
		}
		if got.TMDelete != nil {
			t.Errorf("the row must not be deleted. tm_delete: %v", got.TMDelete)
		}
	})
}

func isNilResult(res any) bool {
	switch v := res.(type) {
	case nil:
		return true
	case *activeflow.Activeflow:
		return v == nil
	case *flow.Flow:
		return v == nil
	case []*activeflow.Activeflow:
		return v == nil
	case []*flow.Flow:
		return v == nil
	}
	return false
}
