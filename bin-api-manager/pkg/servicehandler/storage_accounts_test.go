package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smaccount "monorepo/bin-storage-manager/models/account"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_StorageAccountGet(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		storageAccountID uuid.UUID

		responseStorageAccount *smaccount.Account
		expectRes              *smaccount.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c42a90e0-2c01-11ef-9a00-7b5a2f8e1d3a"),
					CustomerID: uuid.FromStringOrNil("c45e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			storageAccountID: uuid.FromStringOrNil("c4923d44-2c01-11ef-b8c6-2b9e1a5f4d7c"),

			responseStorageAccount: &smaccount.Account{
				ID:         uuid.FromStringOrNil("c4923d44-2c01-11ef-b8c6-2b9e1a5f4d7c"),
				CustomerID: uuid.FromStringOrNil("c45e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
				TMDelete:   nil,
			},
			expectRes: &smaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("c4923d44-2c01-11ef-b8c6-2b9e1a5f4d7c"),
				CustomerID: uuid.FromStringOrNil("c45e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
				TMDelete:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1AccountGet(ctx, tt.storageAccountID).Return(tt.responseStorageAccount, nil)

			res, err := h.StorageAccountGet(ctx, tt.agent, tt.storageAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StorageAccountGetByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseStorageAccounts []smaccount.Account
		expectFilters           map[smaccount.Field]any
		expectRes               *smaccount.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d52a90e0-2c01-11ef-9a00-7b5a2f8e1d3a"),
					CustomerID: uuid.FromStringOrNil("d55e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			responseStorageAccounts: []smaccount.Account{
				{
					ID:         uuid.FromStringOrNil("d5923d44-2c01-11ef-b8c6-2b9e1a5f4d7c"),
					CustomerID: uuid.FromStringOrNil("d55e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
				},
			},
			expectFilters: map[smaccount.Field]any{
				smaccount.FieldCustomerID: uuid.FromStringOrNil("d55e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
				smaccount.FieldDeleted:    false,
			},
			expectRes: &smaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("d5923d44-2c01-11ef-b8c6-2b9e1a5f4d7c"),
				CustomerID: uuid.FromStringOrNil("d55e7f12-2c01-11ef-a2b4-4f6d8c3e7a1b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1AccountList(ctx, "", uint64(1), tt.expectFilters).Return(tt.responseStorageAccounts, nil)

			res, err := h.StorageAccountGetByCustomerID(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
