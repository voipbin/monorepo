package aihandler

import (
	"context"
	"errors"
	"testing"

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

	tests := []struct {
		name      string
		ids       []uuid.UUID
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

			err := h.ValidateMcpServerIDs(context.Background(), customerID, tt.ids)
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
