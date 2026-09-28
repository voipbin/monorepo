package aihandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ValidateMcpServerIDs(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	otherCustomerID := uuid.Must(uuid.NewV4())

	sameCustomerServerID := uuid.Must(uuid.NewV4())
	crossCustomerServerID := uuid.Must(uuid.NewV4())
	nonexistentServerID := uuid.Must(uuid.NewV4())
	nilServerID := uuid.Must(uuid.NewV4())
	deletedServerID := uuid.Must(uuid.NewV4())
	secondDeletedServerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		ids       []uuid.UUID
		storedIDs []uuid.UUID
		setupMock func(*dbhandler.MockDBHandler)
		wantError bool
		// wantTyped asserts the error is a *cerrors.VoipbinError with
		// StatusInvalidArgument (-> HTTP 400). false for a case where an
		// error is expected but it must NOT be typed as InvalidArgument
		// (a genuine DB infra failure, which must fall through to 500).
		wantTyped bool
	}{
		{
			name: "valid same-customer id accepts",
			ids:  []uuid.UUID{sameCustomerServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), sameCustomerServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: sameCustomerServerID, CustomerID: customerID},
				}, nil)
			},
			wantError: false,
		},
		{
			name: "cross-customer id rejects",
			ids:  []uuid.UUID{crossCustomerServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), crossCustomerServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: crossCustomerServerID, CustomerID: otherCustomerID},
				}, nil)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			// The read path deliberately returns soft-deleted rows (GET on a
			// deleted server answers 200), so the TMDelete check has to live
			// here. Without it a customer can whitelist a server they already
			// deleted, leaving the AI carrying an id no consumer will honour.
			//
			// storedIDs is empty: this is an id being ADDED, which is the
			// case the rejection is for.
			name: "soft-deleted id rejects when newly added",
			ids:  []uuid.UUID{deletedServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				ts := time.Now()
				mockDB.EXPECT().McpServerGet(gomock.Any(), deletedServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: deletedServerID, CustomerID: customerID},
					TMDelete: &ts,
				}, nil)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			// Defensive arm: a (nil, nil) return from McpServerGet -- no row
			// and no error -- must still be rejected. Weakening the guard to
			// `srv != nil && srv.CustomerID != customerID`, or skipping nil
			// servers with a `continue`, silently ACCEPTS the id and bypasses
			// the IDOR check wholesale for that id. Same defensive arm the
			// sibling packages pin (aicallhandler's mcp_tool_test.go,
			// mcptoolhandler's Test_TransportRefusesNilServer).
			name: "nil server with no error rejects",
			ids:  []uuid.UUID{nilServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), nilServerID).Return(nil, nil)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			name: "nonexistent id rejects",
			ids:  []uuid.UUID{nonexistentServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), nonexistentServerID).Return(nil, dbhandler.ErrNotFound)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			// Pins the code review fix: a genuine DB infra failure (query
			// build/exec/scan error, NOT dbhandler.ErrNotFound) must NOT be
			// collapsed into a 400 InvalidArgument alongside not-found and
			// cross-customer -- it must remain untyped so errorResponse()
			// falls through to 500, matching the dbhandler.ErrNotFound-vs-
			// other-error split ActivateInsight already uses in db.go.
			name: "db infra failure is not mislabeled as invalid argument",
			ids:  []uuid.UUID{sameCustomerServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), sameCustomerServerID).Return(nil, errors.New("mcpserverGetFromDB: could not query. err: connection refused"))
			},
			wantError: true,
			wantTyped: false,
		},
		{
			name:      "empty ids accepts trivially",
			ids:       []uuid.UUID{},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {},
			wantError: false,
		},
		{
			// D21. Deleting an MCP server does not prune the id from any AI
			// that referenced it, and square-admin builds its picker from the
			// deleted:"false" list -- so the stale id is invisible and
			// unremovable in the UI while every PUT re-submits it. Rejecting
			// it would 400 every save of that AI forever, including edits to
			// unrelated fields. Already-carried deleted ids are tolerated.
			name:      "soft-deleted id accepts when already stored",
			ids:       []uuid.UUID{deletedServerID},
			storedIDs: []uuid.UUID{deletedServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				ts := time.Now()
				mockDB.EXPECT().McpServerGet(gomock.Any(), deletedServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: deletedServerID, CustomerID: customerID},
					TMDelete: &ts,
				}, nil)
			},
			wantError: false,
		},
		{
			// D25. The exemption covers deletion ONLY. A stored id whose row
			// now belongs to another customer is still rejected -- otherwise
			// a cross-customer reference, once stored, could never be caught
			// again, which is the IDOR this validation exists to stop.
			name:      "cross-customer id rejects even when already stored",
			ids:       []uuid.UUID{crossCustomerServerID},
			storedIDs: []uuid.UUID{crossCustomerServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), crossCustomerServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: crossCustomerServerID, CustomerID: otherCustomerID},
				}, nil)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			// D25. Existence is not exempted either: a stored id whose row is
			// gone entirely is rejected, since nothing can ever honour it and
			// (unlike a soft-deleted row) no UI state explains its presence.
			name:      "nonexistent id rejects even when already stored",
			ids:       []uuid.UUID{nonexistentServerID},
			storedIDs: []uuid.UUID{nonexistentServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), nonexistentServerID).Return(nil, dbhandler.ErrNotFound)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			// The exemption is per id, not per request: carrying one deleted
			// id must not let a DIFFERENT deleted id be added in the same PUT.
			name:      "newly added deleted id rejects alongside a stored deleted id",
			ids:       []uuid.UUID{deletedServerID, secondDeletedServerID},
			storedIDs: []uuid.UUID{deletedServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				ts := time.Now()
				mockDB.EXPECT().McpServerGet(gomock.Any(), deletedServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: deletedServerID, CustomerID: customerID},
					TMDelete: &ts,
				}, nil)
				mockDB.EXPECT().McpServerGet(gomock.Any(), secondDeletedServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: secondDeletedServerID, CustomerID: customerID},
					TMDelete: &ts,
				}, nil)
			},
			wantError: true,
			wantTyped: true,
		},
		{
			// Guards against keying the exemption on position or count rather
			// than identity: the stored list holds a different id than the
			// one being added, so the addition must still be rejected.
			name:      "deleted id rejects when a different id is stored",
			ids:       []uuid.UUID{deletedServerID},
			storedIDs: []uuid.UUID{sameCustomerServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				ts := time.Now()
				mockDB.EXPECT().McpServerGet(gomock.Any(), deletedServerID).Return(&mcpserver.McpServer{
					Identity: identity.Identity{ID: deletedServerID, CustomerID: customerID},
					TMDelete: &ts,
				}, nil)
			},
			wantError: true,
			wantTyped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			tt.setupMock(mockDB)

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			err := h.ValidateMcpServerIDs(context.Background(), customerID, tt.ids, tt.storedIDs)
			if tt.wantError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantError {
				var ve *cerrors.VoipbinError
				isTyped := errors.As(err, &ve)
				if tt.wantTyped {
					// Must be a typed *cerrors.VoipbinError so listenhandler's
					// errorResponse() maps it to HTTP 400, not an opaque 500
					// for what is a client-input mistake (invalid/cross-
					// customer mcp_server_id).
					if !isTyped {
						t.Fatalf("expected a *cerrors.VoipbinError, got %T: %v", err, err)
					}
					if ve.Status != cerrors.StatusInvalidArgument {
						t.Fatalf("expected StatusInvalidArgument, got %v", ve.Status)
					}
				} else if isTyped {
					t.Fatalf("expected a plain (untyped) error for a DB infra failure so it falls through to 500, got *cerrors.VoipbinError: %v", ve)
				}
			}
		})
	}
}
