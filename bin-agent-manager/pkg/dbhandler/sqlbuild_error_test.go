package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-agent-manager/models/agent"
)

// failingSqlizer is an integer kind value, so the field preparation accepts it,
// whose SQL rendering always fails.
type failingSqlizer int

func (failingSqlizer) ToSql() (string, []interface{}, error) {
	return "", nil, errTestBoom
}

// failingValuer is a driver value whose conversion always fails.
type failingValuer struct{}

func (failingValuer) Value() (driver.Value, error) {
	return nil, errTestBoom
}

func Test_agentUpdate_buildSQLFailure(t *testing.T) {
	h, mock, _ := newSQLMockHandler(t)

	err := h.agentUpdate(context.Background(), testSeqID(130), map[agent.Field]any{agent.FieldStatus: failingSqlizer(1)})
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !errors.Is(err, errTestBoom) {
		t.Errorf("expected errors.Is(%v), got: %v", errTestBoom, err)
	}
	if !strings.Contains(err.Error(), "build SQL failed") {
		t.Errorf("expected the build failure, got: %v", err)
	}
	// no statement may be sent when the SQL could not be built.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no query was expected: %v", err)
	}
}

func Test_AgentList_buildQueryFailure(t *testing.T) {
	h, mock, _ := newSQLMockHandler(t)

	filters := map[agent.Field]any{agent.FieldName: commondatabasehandler.NotEq{Value: failingValuer{}}}

	res, err := h.AgentList(context.Background(), 10, "", filters)
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if res != nil {
		t.Errorf("expected no result on failure, got: %v", res)
	}
	if !errors.Is(err, errTestBoom) {
		t.Errorf("expected errors.Is(%v), got: %v", errTestBoom, err)
	}
	if !strings.Contains(err.Error(), "could not build query. AgentList") {
		t.Errorf("expected the build failure, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("no query was expected: %v", err)
	}
}
