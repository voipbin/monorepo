package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/cachehandler"
)

var (
	errTestCacheMiss = errors.New("test cache miss")
	errTestCacheSet  = errors.New("test cache set failure")

	testListToken = "2999-01-01 00:00:00.000000"
	testRowTime   = "2023-01-01 00:00:00.000000"
)

// newPrivateDB opens a private in-memory SQLite database (not the shared one of
// the other tests) with the test schema applied.
func newPrivateDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the private database. err: %v", err)
	}
	// every connection of an in-memory database is a separate database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	dir := filepath.Join("..", "..", "scripts", "database_scripts_test")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read the schema directory. err: %v", err)
	}
	for _, e := range entries {
		contents, errRead := os.ReadFile(filepath.Join(dir, e.Name()))
		if errRead != nil {
			t.Fatalf("could not read the schema file. err: %v", errRead)
		}
		if _, errExec := db.Exec(string(contents)); errExec != nil {
			t.Fatalf("could not apply the schema file. file: %s, err: %v", e.Name(), errExec)
		}
	}

	return db
}

func newClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db := newPrivateDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("could not close the private database. err: %v", err)
	}
	return db
}

func newErrTestHandler(mc *gomock.Controller, db *sql.DB) (*handler, *cachehandler.MockCacheHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	now := time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	return &handler{utilHandler: mockUtil, db: db, cache: mockCache}, mockCache
}

// dbEntity describes one table so that the same failure cases run for the
// widget, session and message operations.
type dbEntity struct {
	name  string
	table string

	insertValidSQL string // takes the id as its only argument
	insertBadSQL   string // a row that can not be scanned, takes the id

	hasUpdate bool

	create func(h *handler, ctx context.Context, id uuid.UUID) error
	get    func(h *handler, ctx context.Context, id uuid.UUID) error
	list   func(h *handler, ctx context.Context) error
	update func(h *handler, ctx context.Context, id uuid.UUID) error
	del    func(h *handler, ctx context.Context, id uuid.UUID) error

	expectGetMiss func(c *cachehandler.MockCacheHandler)
	expectSetErr  func(c *cachehandler.MockCacheHandler)
}

func testEntities() []dbEntity {
	return []dbEntity{
		{
			name:  "widget",
			table: "webchat_widgets",
			insertValidSQL: "INSERT INTO webchat_widgets(id, name, status, session_idle_timeout, tm_create) " +
				"VALUES (?, 'w', 'active', 1800, '" + testRowTime + "')",
			insertBadSQL: "INSERT INTO webchat_widgets(id, name, status, session_idle_timeout, tm_create) " +
				"VALUES (?, 'w', 'active', 'abc', '" + testRowTime + "')",
			hasUpdate: true,
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.WidgetCreate(ctx, &widget.Widget{Identity: commonidentity.Identity{ID: id}, Name: "w", Status: widget.StatusActive})
			},
			get: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.WidgetGet(ctx, id)
				return err
			},
			list: func(h *handler, ctx context.Context) error {
				_, err := h.WidgetList(ctx, 10, testListToken, nil)
				return err
			},
			update: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.WidgetUpdate(ctx, id, map[widget.Field]any{widget.FieldName: "renamed"})
			},
			del: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.WidgetDelete(ctx, id)
			},
			expectGetMiss: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().WidgetGet(gomock.Any(), gomock.Any()).Return(nil, errTestCacheMiss)
			},
			expectSetErr: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().WidgetSet(gomock.Any(), gomock.Any()).Return(errTestCacheSet).Times(1)
			},
		},
		{
			name:  "session",
			table: "webchat_sessions",
			insertValidSQL: "INSERT INTO webchat_sessions(id, status, tm_create) " +
				"VALUES (?, 'active', '" + testRowTime + "')",
			insertBadSQL: "INSERT INTO webchat_sessions(id, status, peer, tm_create) " +
				"VALUES (?, 'active', 'not-json', '" + testRowTime + "')",
			hasUpdate: true,
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.SessionCreate(ctx, &session.Session{Identity: commonidentity.Identity{ID: id}, Status: session.StatusActive})
			},
			get: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.SessionGet(ctx, id)
				return err
			},
			list: func(h *handler, ctx context.Context) error {
				_, err := h.SessionList(ctx, 10, testListToken, nil)
				return err
			},
			update: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.SessionUpdate(ctx, id, map[session.Field]any{session.FieldStatus: session.StatusEnded})
			},
			del: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.SessionDelete(ctx, id)
			},
			expectGetMiss: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().SessionGet(gomock.Any(), gomock.Any()).Return(nil, errTestCacheMiss)
			},
			expectSetErr: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().SessionSet(gomock.Any(), gomock.Any()).Return(errTestCacheSet).Times(1)
			},
		},
		{
			name:  "message",
			table: "webchat_messages",
			insertValidSQL: "INSERT INTO webchat_messages(id, direction, status, text, tm_create) " +
				"VALUES (?, 'inbound', 'sent', 'hello', '" + testRowTime + "')",
			insertBadSQL: "INSERT INTO webchat_messages(id, direction, status, text, tm_create, tm_delete) " +
				"VALUES (?, 'inbound', 'sent', 'hello', '" + testRowTime + "', 'garbage')",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.MessageCreate(ctx, &message.Message{Identity: commonidentity.Identity{ID: id}, Direction: message.DirectionInbound, Status: message.StatusSent})
			},
			get: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.MessageGet(ctx, id)
				return err
			},
			list: func(h *handler, ctx context.Context) error {
				_, err := h.MessageList(ctx, 10, testListToken, nil)
				return err
			},
			del: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.MessageDelete(ctx, id)
			},
			expectGetMiss: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, errTestCacheMiss)
			},
			expectSetErr: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().MessageSet(gomock.Any(), gomock.Any()).Return(errTestCacheSet).Times(1)
			},
		},
	}
}

var testEntityID = uuid.FromStringOrNil("a1b2c3d4-e5f6-4a07-8b19-c2d3e4f50617")

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()

	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("could not execute the setup statement. query: %s, err: %v", query, err)
	}
}

func countRows(t *testing.T, db *sql.DB, query string) int {
	t.Helper()

	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("could not count rows. query: %s, err: %v", query, err)
	}
	return n
}

func assertErrContains(t *testing.T, err error, expect string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Wrong match. expect: error containing %q, got: nil", expect)
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("Wrong error. expect: a database failure, got: ErrNotFound")
	}
	if !strings.Contains(err.Error(), expect) {
		t.Errorf("Wrong error. expect: containing %q, got: %v", expect, err)
	}
}

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewHandler(dbTest, mockCache)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.db != dbTest {
		t.Errorf("Wrong database stored.")
	}
	if h.cache != mockCache {
		t.Errorf("Wrong cache stored.")
	}
	if h.utilHandler == nil {
		t.Errorf("Util handler must not be nil.")
	}
}

func Test_closedDatabase_returnsWrappedErrors(t *testing.T) {
	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("create", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, _ := newErrTestHandler(mc, newClosedDB(t))

				assertErrContains(t, e.create(h, ctx, testEntityID), "could not execute query")
			})

			t.Run("get", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, c := newErrTestHandler(mc, newClosedDB(t))
				e.expectGetMiss(c)

				assertErrContains(t, e.get(h, ctx, testEntityID), "could not query")
			})

			t.Run("list", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, _ := newErrTestHandler(mc, newClosedDB(t))

				assertErrContains(t, e.list(h, ctx), "could not query")
			})

			t.Run("delete", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, _ := newErrTestHandler(mc, newClosedDB(t))

				assertErrContains(t, e.del(h, ctx, testEntityID), "exec failed")
			})

			if e.hasUpdate {
				t.Run("update", func(t *testing.T) {
					mc := gomock.NewController(t)
					defer mc.Finish()
					h, _ := newErrTestHandler(mc, newClosedDB(t))

					assertErrContains(t, e.update(h, ctx, testEntityID), "exec failed")
				})
			}
		})
	}
}

// The database stays alive in these cases, so a cache write after the failed
// statement would reach the strict cache mock, which has no expectation.
func Test_failedWrite_doesNotTouchTheCache(t *testing.T) {
	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("create", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				db := newPrivateDB(t)
				h, _ := newErrTestHandler(mc, db)
				mustExec(t, db, "CREATE TRIGGER block_insert BEFORE INSERT ON "+e.table+" BEGIN SELECT RAISE(ABORT, 'blocked'); END;")

				assertErrContains(t, e.create(h, ctx, testEntityID), "could not execute query")
				if n := countRows(t, db, "SELECT count(*) FROM "+e.table); n != 0 {
					t.Errorf("Wrong row count. expect: 0, got: %d", n)
				}
			})

			t.Run("delete", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				db := newPrivateDB(t)
				h, _ := newErrTestHandler(mc, db)
				mustExec(t, db, e.insertValidSQL, testEntityID.Bytes())
				mustExec(t, db, "CREATE TRIGGER block_update BEFORE UPDATE ON "+e.table+" BEGIN SELECT RAISE(ABORT, 'blocked'); END;")

				assertErrContains(t, e.del(h, ctx, testEntityID), "exec failed")
				if n := countRows(t, db, "SELECT count(*) FROM "+e.table+" WHERE tm_delete IS NOT NULL"); n != 0 {
					t.Errorf("Wrong deleted row count. expect: 0, got: %d", n)
				}
			})

			if e.hasUpdate {
				t.Run("update", func(t *testing.T) {
					mc := gomock.NewController(t)
					defer mc.Finish()
					db := newPrivateDB(t)
					h, _ := newErrTestHandler(mc, db)
					mustExec(t, db, e.insertValidSQL, testEntityID.Bytes())
					mustExec(t, db, "CREATE TRIGGER block_update BEFORE UPDATE ON "+e.table+" BEGIN SELECT RAISE(ABORT, 'blocked'); END;")

					assertErrContains(t, e.update(h, ctx, testEntityID), "exec failed")
				})
			}
		})
	}
}

// Updating or deleting an id that does not exist succeeds at the statement
// level; the refresh read finds nothing and nothing is cached.
func Test_absentID_writeSucceedsWithoutCaching(t *testing.T) {
	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("delete", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				h, _ := newErrTestHandler(mc, newPrivateDB(t))

				if err := e.del(h, ctx, testEntityID); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			})

			if e.hasUpdate {
				t.Run("update", func(t *testing.T) {
					mc := gomock.NewController(t)
					defer mc.Finish()
					h, _ := newErrTestHandler(mc, newPrivateDB(t))

					if err := e.update(h, ctx, testEntityID); err != nil {
						t.Errorf("Wrong match. expect: ok, got: %v", err)
					}
				})
			}
		})
	}
}

// A cache write error after a successful statement is ignored by the caller.
func Test_cacheSetFailure_isIgnoredAfterSuccessfulWrite(t *testing.T) {
	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("delete", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				db := newPrivateDB(t)
				h, c := newErrTestHandler(mc, db)
				mustExec(t, db, e.insertValidSQL, testEntityID.Bytes())
				e.expectSetErr(c)

				if err := e.del(h, ctx, testEntityID); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				if n := countRows(t, db, "SELECT count(*) FROM "+e.table+" WHERE tm_delete IS NOT NULL"); n != 1 {
					t.Errorf("Wrong deleted row count. expect: 1, got: %d", n)
				}
			})

			if e.hasUpdate {
				t.Run("update", func(t *testing.T) {
					mc := gomock.NewController(t)
					defer mc.Finish()
					db := newPrivateDB(t)
					h, c := newErrTestHandler(mc, db)
					mustExec(t, db, e.insertValidSQL, testEntityID.Bytes())
					e.expectSetErr(c)

					if err := e.update(h, ctx, testEntityID); err != nil {
						t.Errorf("Wrong match. expect: ok, got: %v", err)
					}
				})
			}

			t.Run("get", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				db := newPrivateDB(t)
				h, c := newErrTestHandler(mc, db)
				mustExec(t, db, e.insertValidSQL, testEntityID.Bytes())
				e.expectGetMiss(c)
				e.expectSetErr(c)

				if err := e.get(h, ctx, testEntityID); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			})
		})
	}
}

func Test_get_absentID_returnsErrNotFound(t *testing.T) {
	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, c := newErrTestHandler(mc, newPrivateDB(t))
			e.expectGetMiss(c)

			if err := e.get(h, context.Background(), testEntityID); !errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong error. expect: ErrNotFound, got: %v", err)
			}
		})
	}
}

func Test_scanFailure_isReportedForGetAndList(t *testing.T) {
	for _, e := range testEntities() {
		t.Run(e.name, func(t *testing.T) {
			ctx := context.Background()

			t.Run("get", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				db := newPrivateDB(t)
				h, c := newErrTestHandler(mc, db)
				mustExec(t, db, e.insertBadSQL, testEntityID.Bytes())
				e.expectGetMiss(c)

				assertErrContains(t, e.get(h, ctx, testEntityID), "could not scan the row")
			})

			t.Run("list", func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()
				db := newPrivateDB(t)
				h, _ := newErrTestHandler(mc, db)
				mustExec(t, db, e.insertBadSQL, testEntityID.Bytes())

				assertErrContains(t, e.list(h, ctx), "could not scan the row")
			})
		})
	}
}

func Test_get_cacheHit_doesNotTouchTheDatabase(t *testing.T) {
	ctx := context.Background()

	t.Run("widget", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, c := newErrTestHandler(mc, newClosedDB(t))

		cached := &widget.Widget{Identity: commonidentity.Identity{ID: testEntityID}, Name: "cached"}
		c.EXPECT().WidgetGet(ctx, testEntityID).Return(cached, nil)

		res, err := h.WidgetGet(ctx, testEntityID)
		if err != nil || res != cached {
			t.Errorf("Wrong match. expect: the cached widget, got: %v, err: %v", res, err)
		}
	})

	t.Run("session", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, c := newErrTestHandler(mc, newClosedDB(t))

		cached := &session.Session{Identity: commonidentity.Identity{ID: testEntityID}, Status: session.StatusActive}
		c.EXPECT().SessionGet(ctx, testEntityID).Return(cached, nil)

		res, err := h.SessionGet(ctx, testEntityID)
		if err != nil || res != cached {
			t.Errorf("Wrong match. expect: the cached session, got: %v, err: %v", res, err)
		}
	})

	t.Run("message", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, c := newErrTestHandler(mc, newClosedDB(t))

		cached := &message.Message{Identity: commonidentity.Identity{ID: testEntityID}, Text: "cached"}
		c.EXPECT().MessageGet(ctx, testEntityID).Return(cached, nil)

		res, err := h.MessageGet(ctx, testEntityID)
		if err != nil || res != cached {
			t.Errorf("Wrong match. expect: the cached message, got: %v, err: %v", res, err)
		}
	})
}

func Test_update_emptyFields_isANoOp(t *testing.T) {
	ctx := context.Background()

	t.Run("widget", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		// closed database: any statement would fail
		h, _ := newErrTestHandler(mc, newClosedDB(t))

		if err := h.WidgetUpdate(ctx, testEntityID, map[widget.Field]any{}); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("session", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _ := newErrTestHandler(mc, newClosedDB(t))

		if err := h.SessionUpdate(ctx, testEntityID, nil); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}
