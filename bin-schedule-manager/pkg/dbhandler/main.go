package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	ScheduleCreate(ctx context.Context, s *schedule.Schedule) error
	ScheduleGet(ctx context.Context, id uuid.UUID) (*schedule.Schedule, error)
	ScheduleList(ctx context.Context, size uint64, token string, filters map[schedule.Field]any) ([]*schedule.Schedule, error)
	ScheduleUpdate(ctx context.Context, id uuid.UUID, fields map[schedule.Field]any) error
	ScheduleDelete(ctx context.Context, id uuid.UUID) error
	ScheduleGetByCustomerIDName(ctx context.Context, customerID uuid.UUID, name string) (*schedule.Schedule, error)
	ScheduleListDue(ctx context.Context, now time.Time, limit uint64) ([]*schedule.Schedule, error)
	ScheduleListUninitialized(ctx context.Context, limit uint64) ([]*schedule.Schedule, error)
	ScheduleInitNextRun(ctx context.Context, id uuid.UUID, next time.Time) (bool, error)

	ScheduleClaimAndCreateExecution(ctx context.Context, s *schedule.Schedule, next time.Time, now time.Time, triggerType execution.TriggerType, tmScheduled time.Time) (*execution.Execution, error)
	ExecutionGet(ctx context.Context, id uuid.UUID) (*execution.Execution, error)
	ExecutionHasRunning(ctx context.Context, scheduleID uuid.UUID) (bool, error)
	ExecutionComplete(ctx context.Context, id uuid.UUID, status execution.Status, statusCode int, errStr string, result string, attemptCount int, durationMS int, now time.Time) (bool, error)
	ExecutionReapAbandoned(ctx context.Context, now time.Time) (int64, error)
	ExecutionList(ctx context.Context, size uint64, token string, filters map[execution.Field]any) ([]*execution.Execution, error)
	ExecutionPrune(ctx context.Context, cutoff time.Time, batch uint64) (int64, error)
	ExecutionCountAll(ctx context.Context) (int64, error)
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = fmt.Errorf("record not found")
)

// mysqlErrDupEntry is MySQL's ER_DUP_ENTRY errno, raised on a UNIQUE/PK
// constraint violation.
const mysqlErrDupEntry = 1062

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}

// isDuplicateKeyErr reports whether the given error is a duplicate-key/unique
// constraint violation. It covers MySQL (errno 1062, matched via the typed
// *mysql.MySQLError so an unrelated message containing "1062" cannot be
// misclassified) and sqlite (used by the test harness).
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}

	var myErr *gomysql.MySQLError
	if errors.As(err, &myErr) && myErr.Number == mysqlErrDupEntry {
		return true
	}

	// sqlite (test harness) surfaces a textual error, not a typed code.
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
