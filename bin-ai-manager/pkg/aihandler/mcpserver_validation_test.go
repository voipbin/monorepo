package aihandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
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
		},
		{
			name: "nonexistent id rejects",
			ids:  []uuid.UUID{nonexistentServerID},
			setupMock: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().McpServerGet(gomock.Any(), nonexistentServerID).Return(nil, dbhandler.ErrNotFound)
			},
			wantError: true,
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
		})
	}
}
