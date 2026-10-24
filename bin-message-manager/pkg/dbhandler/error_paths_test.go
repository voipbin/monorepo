package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/cachehandler"
)

const (
	pathCurTime = "2021-02-26T18:26:49.000000Z"
)

// newClosedDB returns a handle whose every call fails because the connection pool is closed.
func newClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file:closed_"+uuid.Must(uuid.NewV4()).String()+"?mode=memory")
	if err != nil {
		t.Fatalf("Could not open the database. err: %v", err)
	}
	if errClose := db.Close(); errClose != nil {
		t.Fatalf("Could not close the database. err: %v", errClose)
	}
	return db
}

// insertCorruptRow adds a row whose source column is not valid JSON, so scanning it fails.
func insertCorruptRow(t *testing.T, id, customerID uuid.UUID) {
	t.Helper()

	_, err := dbTest.Exec(
		`insert into message_messages(id, customer_id, type, source, targets, medias, tm_create) values (?, ?, 'sms', '{bad', '[]', '[]', '2021-02-26 18:26:48')`,
		id.Bytes(), customerID.Bytes(),
	)
	if err != nil {
		t.Fatalf("Could not insert the row. err: %v", err)
	}
	t.Cleanup(func() {
		_, _ = dbTest.Exec(`delete from message_messages where id = ?`, id.Bytes())
	})
}

func newErrorPathHandler(mc *gomock.Controller, db *sql.DB) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	return &handler{utilHandler: mockUtil, db: db, cache: mockCache}, mockUtil, mockCache
}

func Test_MessageGet_cacheHit(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the database is closed: a cache hit must not touch it.
	h, _, mockCache := newErrorPathHandler(mc, newClosedDB(t))

	id := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c01")
	cached := &message.Message{Text: "from the cache"}
	cached.ID = id

	mockCache.EXPECT().MessageGet(gomock.Any(), id).Return(cached, nil)

	res, err := h.MessageGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != cached {
		t.Errorf("Wrong match. expect: the cached message, got: %v", res)
	}
}

func Test_MessageGet_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, mockCache := newErrorPathHandler(mc, dbTest)

	id := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c02")
	// no MessageSet expectation: nothing is cached for a missing row.
	mockCache.EXPECT().MessageGet(gomock.Any(), id).Return(nil, errors.New("not cached"))

	res, err := h.MessageGet(context.Background(), id)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_MessageGet_scanError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, mockCache := newErrorPathHandler(mc, dbTest)

	id := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c03")
	insertCorruptRow(t, id, uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c10"))
	mockCache.EXPECT().MessageGet(gomock.Any(), id).Return(nil, errors.New("not cached"))

	res, err := h.MessageGet(context.Background(), id)
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: scan error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_MessageList_scanError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, _ := newErrorPathHandler(mc, dbTest)

	customerID := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c11")
	insertCorruptRow(t, uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c04"), customerID)

	res, err := h.MessageList(context.Background(), pathCurTime, 10, map[message.Field]any{message.FieldCustomerID: customerID})
	if err == nil {
		t.Errorf("Wrong match. expect: scan error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_MessageList_unsupportedFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, _ := newErrorPathHandler(mc, dbTest)

	// a channel is not a filter value ApplyFields can turn into a condition.
	res, err := h.MessageList(context.Background(), pathCurTime, 10, map[message.Field]any{message.FieldCustomerID: make(chan int)})
	if err == nil {
		t.Errorf("Wrong match. expect: filter error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_MessageList_emptyTokenUsesCurrentTime(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockUtil, _ := newErrorPathHandler(mc, dbTest)

	// rows created before the injected current time are listed; the corrupt row proves the filter reached the database.
	customerID := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c12")
	insertCorruptRow(t, uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c05"), customerID)
	mockUtil.EXPECT().TimeGetCurTime().Return(pathCurTime)

	_, err := h.MessageList(context.Background(), "", 10, map[message.Field]any{message.FieldCustomerID: customerID})
	if err == nil {
		t.Errorf("Wrong match. expect: scan error from the listed row, got: nil")
	}
}

func Test_closedDatabase(t *testing.T) {
	id := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c06")
	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{
			name: "MessageCreate",
			fn: func(h *handler) error {
				m := &message.Message{}
				m.ID = id
				return h.MessageCreate(context.Background(), m)
			},
		},
		{
			name: "MessageGet",
			fn: func(h *handler) error {
				_, err := h.MessageGet(context.Background(), id)
				return err
			},
		},
		{
			name: "MessageList",
			fn: func(h *handler) error {
				_, err := h.MessageList(context.Background(), pathCurTime, 10, map[message.Field]any{})
				return err
			},
		},
		{
			name: "MessageUpdate",
			fn: func(h *handler) error {
				return h.MessageUpdate(context.Background(), id, map[message.Field]any{message.FieldText: "updated"})
			},
		},
		{
			name: "MessageDelete",
			fn: func(h *handler) error {
				return h.MessageDelete(context.Background(), id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockUtil, mockCache := newErrorPathHandler(mc, newClosedDB(t))

			mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
			mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("not cached")).AnyTimes()

			// MessageSet has no expectation: a failed write must not reach the cache.
			if err := tt.fn(h); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_cacheFailureIsIgnored(t *testing.T) {
	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)
	errCache := errors.New("cache failure")

	tests := []struct {
		name string
		id   uuid.UUID
		fn   func(h *handler, id uuid.UUID) error
	}{
		{
			name: "create ignores the cache set failure",
			id:   uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c07"),
			fn: func(h *handler, id uuid.UUID) error {
				m := &message.Message{}
				m.ID = id
				return h.MessageCreate(context.Background(), m)
			},
		},
		{
			name: "update of a missing row ignores the refresh failure",
			id:   uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c08"),
			fn: func(h *handler, id uuid.UUID) error {
				// no row exists, so the exec succeeds and the cache refresh finds nothing.
				return h.MessageUpdate(context.Background(), id, map[message.Field]any{message.FieldText: "updated"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockUtil, mockCache := newErrorPathHandler(mc, dbTest)

			mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(errCache).AnyTimes()
			t.Cleanup(func() {
				_, _ = dbTest.Exec(`delete from message_messages where id = ?`, tt.id.Bytes())
			})

			if err := tt.fn(h, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MessageGet_cacheSetFailureIsIgnored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockUtil, mockCache := newErrorPathHandler(mc, dbTest)

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)
	id := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c09")
	t.Cleanup(func() {
		_, _ = dbTest.Exec(`delete from message_messages where id = ?`, id.Bytes())
	})

	mockUtil.EXPECT().TimeNow().Return(&curTime)
	mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil)
	m := &message.Message{Text: "stored"}
	m.ID = id
	if err := h.MessageCreate(context.Background(), m); err != nil {
		t.Fatalf("Could not create the message. err: %v", err)
	}

	mockCache.EXPECT().MessageGet(gomock.Any(), id).Return(nil, errors.New("not cached"))
	mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(errors.New("cache failure"))

	res, err := h.MessageGet(context.Background(), id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.Text != "stored" {
		t.Errorf("Wrong match. expect: the stored message, got: %v", res)
	}
}
