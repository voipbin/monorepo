package dbhandler

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/smotes/purse"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

var (
	errCacheMiss = errors.New("cache miss")
	errCacheDown = errors.New("cache down")

	// identifiers used only by the tests of this file
	covCustomerA = uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4a01")
	covCustomerB = uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4a02")
	covCustomerC = uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4a03")
	covMissingID = uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4aff")
)

// openClosedDB returns a database handle that is already closed, so every statement fails.
func openClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("could not close the database: %v", err)
	}

	return db
}

// openIsolatedDB returns a private in-memory database with the test schema, so a test never shares rows with another test or another run.
func openIsolatedDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	// every connection of an in-memory database is a separate database
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	ps, err := purse.New(filepath.Join("../../scripts/database_scripts_test"))
	if err != nil {
		t.Fatalf("could not load the schema scripts: %v", err)
	}
	for _, file := range ps.Files() {
		contents, ok := ps.Get(file)
		if !ok {
			t.Fatalf("the schema script %s was not loaded", file)
		}
		if _, err := db.Exec(contents); err != nil {
			t.Fatalf("could not apply the schema script %s: %v", file, err)
		}
	}

	return db
}

func covTime() *time.Time {
	t := time.Date(2021, 3, 4, 5, 6, 7, 123000000, time.UTC)
	return &t
}

func Test_closedDatabase(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4b01")

	tests := []struct {
		name string

		expectTimeNow  bool
		expectCacheGet func(c *cachehandler.MockCacheHandler)
		call           func(h *handler) error
		expectErr      string
	}{
		{
			name:          "account create",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.AccountCreate(ctx, &account.Account{Identity: commonidentity.Identity{ID: id}})
			},
			expectErr: "could not execute query. AccountCreate",
		},
		{
			name: "account get from the database",
			expectCacheGet: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().AccountGet(ctx, id).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error {
				_, err := h.AccountGet(ctx, id)
				return err
			},
			expectErr: "could not query. accountGetFromDB",
		},
		{
			name: "account list",
			call: func(h *handler) error {
				_, err := h.AccountList(ctx, 10, "2999-01-01 00:00:00.000000", map[account.Field]any{account.FieldCustomerID: covCustomerA})
				return err
			},
			expectErr: "could not query. AccountGets",
		},
		{
			name:          "account update",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.AccountUpdate(ctx, id, map[account.Field]any{account.FieldName: "changed"})
			},
			expectErr: "AccountUpdate: exec failed",
		},
		{
			name:          "account delete",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.AccountDelete(ctx, id)
			},
			expectErr: "AccountDelete: exec failed",
		},
		{
			name:          "conversation create",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.ConversationCreate(ctx, &conversation.Conversation{Identity: commonidentity.Identity{ID: id}})
			},
			expectErr: "could not execute query. ConversationCreate",
		},
		{
			name: "conversation get from the database",
			expectCacheGet: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().ConversationGet(ctx, id).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error {
				_, err := h.ConversationGet(ctx, id)
				return err
			},
			expectErr: "database is closed",
		},
		{
			name: "conversation get by self and peer",
			call: func(h *handler) error {
				_, err := h.ConversationGetBySelfAndPeer(ctx, commonaddress.Address{Type: commonaddress.TypeLine, Target: "self"}, commonaddress.Address{Type: commonaddress.TypeLine, Target: "peer"})
				return err
			},
			expectErr: "could not query. ConversationGetBySelfAndPeer",
		},
		{
			name: "conversation list",
			call: func(h *handler) error {
				_, err := h.ConversationList(ctx, 10, "2999-01-01 00:00:00.000000", map[conversation.Field]any{conversation.FieldCustomerID: covCustomerA})
				return err
			},
			expectErr: "could not query. ConversationGets",
		},
		{
			name:          "conversation update",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.ConversationUpdate(ctx, id, map[conversation.Field]any{conversation.FieldName: "changed"})
			},
			expectErr: "ConversationUpdate: exec failed",
		},
		{
			name:          "media create",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.MediaCreate(ctx, &media.Media{Identity: commonidentity.Identity{ID: id}})
			},
			expectErr: "could not execute query. MediaCreate",
		},
		{
			name: "media get from the database",
			expectCacheGet: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().MediaGet(ctx, id).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error {
				_, err := h.MediaGet(ctx, id)
				return err
			},
			expectErr: "could not query. mediaGetFromDB",
		},
		{
			name:          "message create",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.MessageCreate(ctx, &message.Message{Identity: commonidentity.Identity{ID: id}})
			},
			expectErr: "could not execute query. MessageCreate",
		},
		{
			name: "message get from the database",
			expectCacheGet: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().MessageGet(ctx, id).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error {
				_, err := h.MessageGet(ctx, id)
				return err
			},
			expectErr: "could not query. messageGetFromDB",
		},
		{
			name: "message list",
			call: func(h *handler) error {
				_, err := h.MessageList(ctx, "2999-01-01 00:00:00.000000", 10, map[message.Field]any{message.FieldCustomerID: covCustomerA})
				return err
			},
			expectErr: "could not query. MessageGets",
		},
		{
			name: "message gets by transaction id",
			call: func(h *handler) error {
				_, err := h.MessageGetsByTransactionID(ctx, "transaction", "2999-01-01 00:00:00.000000", 10)
				return err
			},
			expectErr: "could not query. MessageGetsByTransactionID",
		},
		{
			name:          "message update",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.MessageUpdate(ctx, id, map[message.Field]any{message.FieldStatus: message.StatusDone})
			},
			expectErr: "MessageUpdate: exec failed",
		},
		{
			name:          "message delete",
			expectTimeNow: true,
			call: func(h *handler) error {
				return h.MessageDelete(ctx, id)
			},
			expectErr: "MessageDelete: exec failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the cache and the clock are strict mocks: a call that is not declared fails the test
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &handler{utilHandler: mockUtil, db: openClosedDB(t), cache: mockCache}

			if tt.expectTimeNow {
				mockUtil.EXPECT().TimeNow().Return(covTime())
			}
			if tt.expectCacheGet != nil {
				tt.expectCacheGet(mockCache)
			}

			err := tt.call(h)
			if err == nil {
				t.Fatalf("error = nil, want %q", tt.expectErr)
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.expectErr)
			}
			if !strings.Contains(err.Error(), "database is closed") {
				t.Errorf("error = %q, want the cause of the failure to be kept", err.Error())
			}
		})
	}
}

func Test_get_notFound(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(c *cachehandler.MockCacheHandler)
		call  func(h *handler) error
	}{
		{
			name: "account",
			setup: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().AccountGet(ctx, covMissingID).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error { _, err := h.AccountGet(ctx, covMissingID); return err },
		},
		{
			name: "conversation",
			setup: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().ConversationGet(ctx, covMissingID).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error { _, err := h.ConversationGet(ctx, covMissingID); return err },
		},
		{
			name: "media",
			setup: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().MediaGet(ctx, covMissingID).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error { _, err := h.MediaGet(ctx, covMissingID); return err },
		},
		{
			name: "message",
			setup: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().MessageGet(ctx, covMissingID).Return(nil, errCacheMiss)
			},
			call: func(h *handler) error { _, err := h.MessageGet(ctx, covMissingID); return err },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: openIsolatedDB(t), cache: mockCache}
			tt.setup(mockCache)

			if err := tt.call(h); !errors.Is(err, ErrNotFound) {
				t.Errorf("error = %v, want %v", err, ErrNotFound)
			}
		})
	}
}

func Test_get_cacheHitSkipsDatabase(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4c01")

	mc := gomock.NewController(t)
	defer mc.Finish()

	// the database is closed, so any access to it would fail the call
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: openClosedDB(t), cache: mockCache}

	cachedAccount := &account.Account{Identity: commonidentity.Identity{ID: id}, Name: "cached account"}
	mockCache.EXPECT().AccountGet(ctx, id).Return(cachedAccount, nil)
	resAccount, err := h.AccountGet(ctx, id)
	if err != nil || resAccount != cachedAccount {
		t.Errorf("AccountGet() = %v, %v, want the cached value", resAccount, err)
	}

	cachedConversation := &conversation.Conversation{Identity: commonidentity.Identity{ID: id}, Name: "cached conversation"}
	mockCache.EXPECT().ConversationGet(ctx, id).Return(cachedConversation, nil)
	resConversation, err := h.ConversationGet(ctx, id)
	if err != nil || resConversation != cachedConversation {
		t.Errorf("ConversationGet() = %v, %v, want the cached value", resConversation, err)
	}

	cachedMedia := &media.Media{Identity: commonidentity.Identity{ID: id}, Filename: "cached.wav"}
	mockCache.EXPECT().MediaGet(ctx, id).Return(cachedMedia, nil)
	resMedia, err := h.MediaGet(ctx, id)
	if err != nil || resMedia != cachedMedia {
		t.Errorf("MediaGet() = %v, %v, want the cached value", resMedia, err)
	}

	cachedMessage := &message.Message{Identity: commonidentity.Identity{ID: id}}
	mockCache.EXPECT().MessageGet(ctx, id).Return(cachedMessage, nil)
	resMessage, err := h.MessageGet(ctx, id)
	if err != nil || resMessage != cachedMessage {
		t.Errorf("MessageGet() = %v, %v, want the cached value", resMessage, err)
	}
}

// A failing cache must not fail a write that the database accepted: the cache is only an accelerator.
func Test_create_cacheFailureIsIgnored(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		id     uuid.UUID
		setup  func(c *cachehandler.MockCacheHandler)
		create func(h *handler, id uuid.UUID) error
		check  func(h *handler, id uuid.UUID) error
	}{
		{
			name:  "account",
			id:    uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4d01"),
			setup: func(c *cachehandler.MockCacheHandler) { c.EXPECT().AccountSet(ctx, gomock.Any()).Return(errCacheDown) },
			create: func(h *handler, id uuid.UUID) error {
				return h.AccountCreate(ctx, &account.Account{Identity: commonidentity.Identity{ID: id, CustomerID: covCustomerB}})
			},
			check: func(h *handler, id uuid.UUID) error { _, err := h.accountGetFromDB(ctx, id); return err },
		},
		{
			name: "conversation",
			id:   uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4d02"),
			setup: func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().ConversationSet(ctx, gomock.Any()).Return(errCacheDown)
			},
			create: func(h *handler, id uuid.UUID) error {
				return h.ConversationCreate(ctx, &conversation.Conversation{Identity: commonidentity.Identity{ID: id, CustomerID: covCustomerB}})
			},
			check: func(h *handler, id uuid.UUID) error { _, err := h.conversationGetFromDB(ctx, id); return err },
		},
		{
			name:  "media",
			id:    uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4d03"),
			setup: func(c *cachehandler.MockCacheHandler) { c.EXPECT().MediaSet(ctx, gomock.Any()).Return(errCacheDown) },
			create: func(h *handler, id uuid.UUID) error {
				return h.MediaCreate(ctx, &media.Media{Identity: commonidentity.Identity{ID: id, CustomerID: covCustomerB}})
			},
			check: func(h *handler, id uuid.UUID) error { _, err := h.mediaGetFromDB(ctx, id); return err },
		},
		{
			name:  "message",
			id:    uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4d04"),
			setup: func(c *cachehandler.MockCacheHandler) { c.EXPECT().MessageSet(ctx, gomock.Any()).Return(errCacheDown) },
			create: func(h *handler, id uuid.UUID) error {
				return h.MessageCreate(ctx, &message.Message{Identity: commonidentity.Identity{ID: id, CustomerID: covCustomerB}})
			},
			check: func(h *handler, id uuid.UUID) error { _, err := h.messageGetFromDB(ctx, id); return err },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &handler{utilHandler: mockUtil, db: openIsolatedDB(t), cache: mockCache}

			mockUtil.EXPECT().TimeNow().Return(covTime())
			tt.setup(mockCache)

			if err := tt.create(h, tt.id); err != nil {
				t.Fatalf("create error = %v, want nil", err)
			}
			if err := tt.check(h, tt.id); err != nil {
				t.Errorf("the row was not stored: %v", err)
			}
		})
	}
}

// A row that exists only in the database is returned and written to the cache, and a cache failure on that write is not an error.
func Test_get_fromDatabaseIgnoresCacheSetFailure(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4e01")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &handler{utilHandler: mockUtil, db: openIsolatedDB(t), cache: mockCache}

	mockUtil.EXPECT().TimeNow().Return(covTime())
	mockCache.EXPECT().ConversationSet(ctx, gomock.Any()).Return(nil)
	if err := h.ConversationCreate(ctx, &conversation.Conversation{Identity: commonidentity.Identity{ID: id, CustomerID: covCustomerB}, DialogID: "dialog-cov"}); err != nil {
		t.Fatalf("create error = %v", err)
	}

	mockCache.EXPECT().ConversationGet(ctx, id).Return(nil, errCacheMiss)
	mockCache.EXPECT().ConversationSet(ctx, gomock.Any()).Return(errCacheDown)
	res, err := h.ConversationGet(ctx, id)
	if err != nil {
		t.Fatalf("ConversationGet() error = %v, want nil", err)
	}
	if res.ID != id || res.DialogID != "dialog-cov" {
		t.Errorf("ConversationGet() = %v, want the stored row", res)
	}
}

func Test_rowsAffectedZero_returnsNotFound(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{name: "account delete", call: func(h *handler) error { return h.AccountDelete(ctx, covMissingID) }},
		{name: "message update", call: func(h *handler) error {
			return h.MessageUpdate(ctx, covMissingID, map[message.Field]any{message.FieldStatus: message.StatusDone})
		}},
		{name: "message delete", call: func(h *handler) error { return h.MessageDelete(ctx, covMissingID) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no cache call is declared: nothing was changed, so nothing may be cached
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &handler{utilHandler: mockUtil, db: openIsolatedDB(t), cache: cachehandler.NewMockCacheHandler(mc)}
			mockUtil.EXPECT().TimeNow().Return(covTime())

			if err := tt.call(h); !errors.Is(err, ErrNotFound) {
				t.Errorf("error = %v, want %v", err, ErrNotFound)
			}
		})
	}
}

func Test_update_emptyFieldsDoNothing(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e4f01")

	mc := gomock.NewController(t)
	defer mc.Finish()

	// the clock, the cache and the closed database all reject any use
	h := &handler{utilHandler: utilhandler.NewMockUtilHandler(mc), db: openClosedDB(t), cache: cachehandler.NewMockCacheHandler(mc)}

	if err := h.AccountUpdate(ctx, id, map[account.Field]any{}); err != nil {
		t.Errorf("AccountUpdate() error = %v, want nil", err)
	}
	if err := h.ConversationUpdate(ctx, id, nil); err != nil {
		t.Errorf("ConversationUpdate() error = %v, want nil", err)
	}
	if err := h.MessageUpdate(ctx, id, map[message.Field]any{}); err != nil {
		t.Errorf("MessageUpdate() error = %v, want nil", err)
	}
}

// With an empty page token the lists use the current time, so rows created before now are returned.
func Test_list_emptyTokenUsesCurrentTime(t *testing.T) {
	ctx := context.Background()
	const now = "2999-01-01 00:00:00.000000"

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &handler{utilHandler: mockUtil, db: openIsolatedDB(t), cache: mockCache}

	accountID := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e5001")
	conversationID := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e5002")
	messageID := uuid.FromStringOrNil("5c0e3a6e-7d52-4a1b-9c1a-0f6b8d2e5003")

	mockUtil.EXPECT().TimeNow().Return(covTime()).Times(3)
	mockCache.EXPECT().AccountSet(ctx, gomock.Any()).Return(nil)
	mockCache.EXPECT().ConversationSet(ctx, gomock.Any()).Return(nil)
	mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
	if err := h.AccountCreate(ctx, &account.Account{Identity: commonidentity.Identity{ID: accountID, CustomerID: covCustomerC}}); err != nil {
		t.Fatalf("account create error = %v", err)
	}
	if err := h.ConversationCreate(ctx, &conversation.Conversation{Identity: commonidentity.Identity{ID: conversationID, CustomerID: covCustomerC}}); err != nil {
		t.Fatalf("conversation create error = %v", err)
	}
	if err := h.MessageCreate(ctx, &message.Message{Identity: commonidentity.Identity{ID: messageID, CustomerID: covCustomerC}, TransactionID: "cov-transaction"}); err != nil {
		t.Fatalf("message create error = %v", err)
	}

	mockUtil.EXPECT().TimeGetCurTime().Return(now).Times(4)

	accounts, err := h.AccountList(ctx, 10, "", map[account.Field]any{account.FieldCustomerID: covCustomerC})
	if err != nil || len(accounts) != 1 || accounts[0].ID != accountID {
		t.Errorf("AccountList() = %v, %v, want the created account", accounts, err)
	}

	conversations, err := h.ConversationList(ctx, 10, "", map[conversation.Field]any{conversation.FieldCustomerID: covCustomerC})
	if err != nil || len(conversations) != 1 || conversations[0].ID != conversationID {
		t.Errorf("ConversationList() = %v, %v, want the created conversation", conversations, err)
	}

	messages, err := h.MessageList(ctx, "", 10, map[message.Field]any{message.FieldCustomerID: covCustomerC})
	if err != nil || len(messages) != 1 || messages[0].ID != messageID {
		t.Errorf("MessageList() = %v, %v, want the created message", messages, err)
	}

	byTransaction, err := h.MessageGetsByTransactionID(ctx, "cov-transaction", "", 10)
	if err != nil || len(byTransaction) != 1 || byTransaction[0].ID != messageID {
		t.Errorf("MessageGetsByTransactionID() = %v, %v, want the created message", byTransaction, err)
	}
}
