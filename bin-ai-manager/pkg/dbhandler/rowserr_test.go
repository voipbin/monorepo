package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// failingDriver is a minimal database/sql driver whose queries succeed but whose
// row iteration fails at once, the way a context cancelled while the rows are
// being read surfaces: Next returns false and the cause is only in rows.Err().
type failingDriver struct{}

func (failingDriver) Open(string) (driver.Conn, error) { return failingConn{}, nil }

type failingConn struct{}

func (failingConn) Prepare(string) (driver.Stmt, error) { return failingStmt{}, nil }
func (failingConn) Close() error                        { return nil }
func (failingConn) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }

type failingStmt struct{}

func (failingStmt) Close() error  { return nil }
func (failingStmt) NumInput() int { return -1 }
func (failingStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, errors.New("unsupported")
}
func (failingStmt) Query([]driver.Value) (driver.Rows, error) { return failingRows{}, nil }

type failingRows struct{}

func (failingRows) Columns() []string         { return []string{"id"} }
func (failingRows) Close() error              { return nil }
func (failingRows) Next([]driver.Value) error { return errRowsBroken }

func init() {
	sql.Register("dbhandler_failing_rows", failingDriver{})
}

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial (here empty) result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	db, err := sql.Open("dbhandler_failing_rows", "")
	if err != nil {
		t.Fatalf("could not open the fake db. err: %v", err)
	}
	defer func() { _ = db.Close() }()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          db,
		cache:       mockCache,
	}

	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()
	mockCache.EXPECT().AIcallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errors.New("miss")).AnyTimes()

	ctx := context.Background()
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"get ai", func() error { _, err := h.aiGetFromDB(ctx, id); return err }},
		{"get aiaudit", func() error { _, err := h.aiauditGetFromDB(ctx, id); return err }},
		{"get aicall", func() error { _, err := h.aicallGetFromDB(ctx, id); return err }},
		{"get aicall by reference id", func() error { _, err := h.AIcallGetByReferenceID(ctx, id); return err }},
		{"get aiprompthistory", func() error { _, err := h.aiPromptHistoryGetFromDB(ctx, id); return err }},
		{"get aipromptproposal", func() error { _, err := h.aipromptproposalGetFromDB(ctx, id); return err }},
		{"get message", func() error { _, err := h.messageGetFromDB(ctx, id); return err }},
		{"get summary", func() error { _, err := h.summaryGetFromDB(ctx, id); return err }},
		{"get team", func() error { _, err := h.teamGetFromDB(ctx, id); return err }},
		{"list ai", func() error { _, err := h.AIList(ctx, 10, "", nil); return err }},
		{"list aiaudit", func() error { _, err := h.AIAuditList(ctx, 10, "", nil); return err }},
		{"list aicall", func() error { _, err := h.AIcallList(ctx, 10, "", nil); return err }},
		{"list aiprompthistory", func() error { _, err := h.AIPromptHistoryGetsByAIID(ctx, id, 10, ""); return err }},
		{"list aipromptproposal", func() error { _, err := h.AIPromptProposalList(ctx, 10, "", nil); return err }},
		{"list message", func() error { _, err := h.MessageList(ctx, 10, "", nil); return err }},
		{"list summary", func() error { _, err := h.SummaryList(ctx, 10, "", nil); return err }},
		{"list team", func() error { _, err := h.TeamList(ctx, 10, "", nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !strings.Contains(err.Error(), errRowsBroken.Error()) {
				t.Errorf("expected the iteration failure in the error, got: %v", err)
			}
		})
	}
}
