package dbhandler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/cachehandler"
)

// errIteration is the error the fake driver returns while iterating rows.
var errIteration = errors.New("injected iteration error")

// iterErrConnector is a driver.Connector whose queries return rows that fail on the first Next.
type iterErrConnector struct{}

func (iterErrConnector) Connect(context.Context) (driver.Conn, error) { return iterErrConn{}, nil }
func (iterErrConnector) Driver() driver.Driver                        { return nil }

type iterErrConn struct{}

// Prepare and Begin are intentionally unsupported: only QueryContext is used by the tests.
func (iterErrConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (iterErrConn) Close() error                        { return nil }
func (iterErrConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (iterErrConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return iterErrRows{}, nil
}

type iterErrRows struct{}

func (iterErrRows) Columns() []string         { return []string{"id"} }
func (iterErrRows) Close() error              { return nil }
func (iterErrRows) Next([]driver.Value) error { return errIteration }

func Test_messageGetFromDB_iterationError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	db := sql.OpenDB(iterErrConnector{})
	defer func() { _ = db.Close() }()

	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          db,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	res, err := h.messageGetFromDB(context.Background(), uuid.FromStringOrNil("7c5d2f0e-b0b1-11ee-8d3a-6b1f4c2a9e01"))
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !errors.Is(err, errIteration) {
		t.Errorf("Wrong match. expect: errors.Is(err, errIteration), got: %v", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: not ErrNotFound, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_MessageList_iterationError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	db := sql.OpenDB(iterErrConnector{})
	defer func() { _ = db.Close() }()

	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          db,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}

	res, err := h.MessageList(context.Background(), "2021-02-26T18:26:49.000000Z", 10, map[message.Field]any{})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !errors.Is(err, errIteration) {
		t.Errorf("Wrong match. expect: errors.Is(err, errIteration), got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

// Test_canceledContext checks that every DB call of the handler stops on a canceled context.
// Each case uses its own UUID, so a row left behind by a call that ignores the context
// (before the context conversion) cannot affect the other cases of the shared in-memory database.
func Test_canceledContext(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
		fn   func(h *handler, ctx context.Context, id uuid.UUID) error
	}{
		{
			name: "MessageGet",
			id:   uuid.FromStringOrNil("a1c2e3f4-b0b2-11ee-9e11-0f3d5a7b9c01"),
			fn: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.MessageGet(ctx, id)
				return err
			},
		},
		{
			name: "MessageList",
			id:   uuid.FromStringOrNil("a1c2e3f4-b0b2-11ee-9e11-0f3d5a7b9c02"),
			fn: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.MessageList(ctx, "2021-02-26T18:26:49.000000Z", 10, map[message.Field]any{})
				return err
			},
		},
		{
			name: "MessageCreate",
			id:   uuid.FromStringOrNil("a1c2e3f4-b0b2-11ee-9e11-0f3d5a7b9c03"),
			fn: func(h *handler, ctx context.Context, id uuid.UUID) error {
				m := &message.Message{}
				m.ID = id
				return h.MessageCreate(ctx, m)
			},
		},
		{
			name: "MessageUpdate",
			id:   uuid.FromStringOrNil("a1c2e3f4-b0b2-11ee-9e11-0f3d5a7b9c04"),
			fn: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.MessageUpdate(ctx, id, map[message.Field]any{message.FieldText: "updated"})
			},
		},
		{
			name: "MessageDelete",
			id:   uuid.FromStringOrNil("a1c2e3f4-b0b2-11ee-9e11-0f3d5a7b9c05"),
			fn: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.MessageDelete(ctx, id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
			mockUtil.EXPECT().TimeGetCurTime().Return("2021-02-26T18:26:49.000000Z").AnyTimes()
			mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("not cached")).AnyTimes()
			mockCache.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := tt.fn(&h, ctx, tt.id)
			if !errors.Is(err, context.Canceled) {
				t.Errorf("Wrong match. expect: context.Canceled, got: %v", err)
			}
		})
	}
}
