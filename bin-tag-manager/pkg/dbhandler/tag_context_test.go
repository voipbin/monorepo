package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/cachehandler"
)

// newRowErrorRows returns a single-row result whose iteration fails at row 0.
func newRowErrorRows(rowErr error) *sqlmock.Rows {
	columns := commondatabasehandler.GetDBFields(&tag.Tag{})
	values := make([]driver.Value, len(columns))
	return sqlmock.NewRows(columns).AddRow(values...).RowError(0, rowErr)
}

func Test_tagGetFromDB_RowError(t *testing.T) {
	rowErr := errors.New("connection lost")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT").WillReturnRows(newRowErrorRows(rowErr))

	h := handler{db: db}
	res, err := h.tagGetFromDB(context.Background(), uuid.FromStringOrNil("11111111-50d7-11ec-a6b1-8f9671a9e70e"))
	if err == nil {
		t.Fatalf("Expected an error, got nil. res: %v", res)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("An iteration error must not be reported as ErrNotFound. err: %v", err)
	}
	if !errors.Is(err, rowErr) {
		t.Errorf("The cause must be preserved. expect: %v, got: %v", rowErr, err)
	}
}

func Test_TagList_RowError(t *testing.T) {
	rowErr := errors.New("connection lost")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT").WillReturnRows(newRowErrorRows(rowErr))

	h := handler{db: db}
	res, err := h.TagList(context.Background(), 10, utilhandler.TimeGetCurTime(), map[tag.Field]any{})
	if err == nil {
		t.Fatalf("Expected an error, got nil. res: %v", res)
	}
	if !errors.Is(err, rowErr) {
		t.Errorf("The cause must be preserved. expect: %v, got: %v", rowErr, err)
	}
	if res != nil {
		t.Errorf("Expected nil result on error. got: %v", res)
	}
}

// Test_DBCalls_CanceledContext checks that every DB call of the handler carries
// the given context: a context that is already canceled makes the call fail
// right away. It does not check the interruption of a running query.
// The cancel is applied only after the fixture row exists.
func Test_DBCalls_CanceledContext(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	newTag := func(id, customerID string) *tag.Tag {
		return &tag.Tag{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil(id),
				CustomerID: uuid.FromStringOrNil(customerID),
			},
			Name:   "name",
			Detail: "detail",
		}
	}

	tests := []struct {
		name string
		id   string
		cust string
		call func(h *handler, ctx context.Context, tg *tag.Tag) error
	}{
		{
			name: "TagCreate",
			id:   "c0000001-50d7-11ec-a6b1-8f9671a9e70e",
			cust: "c1000001-50d7-11ec-a6b1-8f9671a9e70e",
			call: func(h *handler, ctx context.Context, tg *tag.Tag) error {
				return h.TagCreate(ctx, tg)
			},
		},
		{
			name: "TagUpdate",
			id:   "c0000002-50d7-11ec-a6b1-8f9671a9e70e",
			cust: "c1000002-50d7-11ec-a6b1-8f9671a9e70e",
			call: func(h *handler, ctx context.Context, tg *tag.Tag) error {
				return h.TagUpdate(ctx, tg.ID, map[tag.Field]any{tag.FieldName: "updated"})
			},
		},
		{
			name: "TagDelete",
			id:   "c0000003-50d7-11ec-a6b1-8f9671a9e70e",
			cust: "c1000003-50d7-11ec-a6b1-8f9671a9e70e",
			call: func(h *handler, ctx context.Context, tg *tag.Tag) error {
				return h.TagDelete(ctx, tg.ID)
			},
		},
		{
			name: "tagGetFromDB",
			id:   "c0000004-50d7-11ec-a6b1-8f9671a9e70e",
			cust: "c1000004-50d7-11ec-a6b1-8f9671a9e70e",
			call: func(h *handler, ctx context.Context, tg *tag.Tag) error {
				_, err := h.tagGetFromDB(ctx, tg.ID)
				return err
			},
		},
		{
			name: "TagList",
			id:   "c0000005-50d7-11ec-a6b1-8f9671a9e70e",
			cust: "c1000005-50d7-11ec-a6b1-8f9671a9e70e",
			call: func(h *handler, ctx context.Context, tg *tag.Tag) error {
				_, err := h.TagList(ctx, 10, utilhandler.TimeGetCurTime(), map[tag.Field]any{
					tag.FieldCustomerID: tg.CustomerID,
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			// the fixture row exists, so a call that ignores the context would succeed
			mockUtil.EXPECT().TimeNow().Return(curTime).AnyTimes()
			mockCache.EXPECT().TagSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			tg := newTag(tt.id, tt.cust)
			if tt.name != "TagCreate" {
				if errCreate := h.TagCreate(context.Background(), tg); errCreate != nil {
					t.Fatalf("could not create the fixture row: %v", errCreate)
				}
			}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			if err := tt.call(h, ctx, tg); err == nil {
				t.Errorf("Expected an error for the canceled context, got nil")
			}
		})
	}
}
