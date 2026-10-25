package dbhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/cachehandler"
)

// insertMalformedIDRow inserts a row whose id is not a 16 byte value, which the model scan rejects.
func insertMalformedIDRow(t *testing.T, h *handler) {
	t.Helper()

	_, err := h.db.Exec(
		`insert into number_numbers(id, number, status, tm_create, tm_renew) values (?, ?, ?, ?, ?)`,
		"short-id", "+155****0101", string(number.StatusActive), "2021-02-26 18:26:49.000000", "2021-02-26 18:26:49.000000",
	)
	if err != nil {
		t.Fatalf("could not insert the row. err: %v", err)
	}
}

func Test_malformedRow_scanErrors(t *testing.T) {
	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{
			"list",
			func(h *handler) error {
				_, err := h.NumberList(context.Background(), 10, "2030-01-01 00:00:00.000000", nil)
				return err
			},
		},
		{
			"get by renew time",
			func(h *handler) error {
				_, err := h.NumberGetsByTMRenew(context.Background(), "2030-01-01 00:00:00.000000", 10, nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &handler{utilHandler: utilhandler.NewUtilHandler(), db: newPrivateDB(t), cache: cachehandler.NewMockCacheHandler(mc)}
			insertMalformedIDRow(t, h)

			err := tt.fn(h)
			if err == nil {
				t.Fatal("expected a scan error, got nil")
			}
			if !strings.Contains(err.Error(), "could not scan the row") {
				t.Errorf("the scan failure is not carried in the error. got: %v", err)
			}
		})
	}
}

// Test_NumberUpdate_unencodableField checks that a field value that cannot be encoded is rejected
// while the update is being prepared, before the database is used.
func Test_NumberUpdate_unencodableField(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	now := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)

	// a closed database makes any use of the database visible as an "exec failed" error.
	h := &handler{utilHandler: mockUtil, db: newClosedDB(t)}

	err := h.NumberUpdate(
		context.Background(),
		uuid.FromStringOrNil("6a1f0c1e-0103-11ee-8a01-000000000103"),
		map[number.Field]any{number.FieldName: map[string]any{"k": make(chan int)}},
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "prepare fields failed") {
		t.Errorf("expected the prepare failure. got: %v", err)
	}
	if strings.Contains(err.Error(), "exec failed") {
		t.Errorf("the database must not be reached. got: %v", err)
	}
}
