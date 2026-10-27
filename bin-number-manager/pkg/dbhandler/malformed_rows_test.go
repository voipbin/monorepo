package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-number-manager/models/provider"
)

// Test_numberGetFromDB_malformedMetadata checks that a stored row that cannot be decoded is
// reported with the id, and is not mistaken for a missing row.
func Test_numberGetFromDB_malformedMetadata(t *testing.T) {
	h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t)}

	id := uuid.FromStringOrNil("6a1f0c1e-0201-11ee-8a01-000000000201")
	if _, err := h.db.Exec(
		`insert into number_numbers(id, number, status, metadata) values (?, ?, ?, ?)`,
		id.Bytes(), "+155****0201", "active", "{not-json",
	); err != nil {
		t.Fatalf("could not insert the row. err: %v", err)
	}

	res, err := h.numberGetFromDB(context.Background(), id)
	if err == nil {
		t.Fatalf("Wrong match. expect: a decode error, got: %v", res)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. a malformed row must not be reported as not found, got: %v", err)
	}
	if !strings.Contains(err.Error(), id.String()) {
		t.Errorf("Wrong match. expect the error to carry the id %s, got: %v", id, err)
	}
}

func Test_NumberProviderGet_malformedConfig(t *testing.T) {
	ctx := context.Background()
	h, _ := newProviderTestHandler(t)

	if _, err := h.db.Exec(
		`insert into number_providers(provider_name, enabled, priority, config) values (?, ?, ?, ?)`,
		"gamma", 1, 5, "{not-json",
	); err != nil {
		t.Fatalf("could not insert the row. err: %v", err)
	}

	res, err := h.NumberProviderGet(ctx, "gamma")
	if err == nil {
		t.Fatalf("Wrong match. expect: a decode error, got: %v", res)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. a malformed row must not be reported as not found, got: %v", err)
	}
	if !strings.Contains(err.Error(), "gamma") {
		t.Errorf("Wrong match. expect the error to carry the provider name, got: %v", err)
	}
}

func Test_NumberProviderUpdate_unencodableField(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)

	// a closed database makes any use of the database visible as an "exec failed" error.
	h := &handler{utilHandler: mockUtil, db: newClosedDB(t)}

	err := h.NumberProviderUpdate(context.Background(), "alpha", map[provider.Field]any{
		provider.FieldConfig: map[string]any{"k": make(chan int)},
	})
	if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
		t.Errorf("Wrong match. expect the prepare failure, got: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "exec failed") {
		t.Errorf("the database must not be reached. got: %v", err)
	}
}

func Test_isDuplicateKeyErr_mysql(t *testing.T) {
	tests := []struct {
		name string
		err  error

		expect bool
	}{
		{"duplicate entry", &gomysql.MySQLError{Number: 1062, Message: "Duplicate entry"}, true},
		{"wrapped duplicate entry", errors.Join(errors.New("outer"), &gomysql.MySQLError{Number: 1062}), true},
		{"another mysql error", &gomysql.MySQLError{Number: 1146, Message: "Table doesn't exist"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDuplicateKeyErr(tt.err); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
