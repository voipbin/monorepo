package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smaccount "monorepo/bin-storage-manager/models/account"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_storageAccountGet(t *testing.T) {

	tests := []struct {
		name string

		agent            *amagent.Agent
		storageAccountID uuid.UUID

		responseStorageAccount *smaccount.Account
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b7ef6ea6-1bd7-11ef-88a6-ff71fa05d8bd"),
					CustomerID: uuid.FromStringOrNil("b83e3c98-1bd7-11ef-8f14-9f07e5f6c56b"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("b87c0fc8-1bd7-11ef-84a7-3b073405f0cd"),
			&smaccount.Account{
				ID:         uuid.FromStringOrNil("b87c0fc8-1bd7-11ef-84a7-3b073405f0cd"),
				CustomerID: uuid.FromStringOrNil("b83e3c98-1bd7-11ef-8f14-9f07e5f6c56b"),
				TMDelete:   defaultTimestamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().StorageV1AccountGet(ctx, tt.storageAccountID).Return(tt.responseStorageAccount, nil)

			res, err := h.storageAccountGet(ctx, tt.agent, tt.storageAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseStorageAccount) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseStorageAccount, res)
			}
		})
	}
}

func Test_StorageAccountDelete(t *testing.T) {

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
					ID:         uuid.FromStringOrNil("1a49c8f8-1bd8-11ef-b861-bf0a568022b9"),
					CustomerID: uuid.FromStringOrNil("1a73a632-1bd8-11ef-8c46-4fdca968dac2"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			storageAccountID: uuid.FromStringOrNil("1aa43522-1bd8-11ef-870e-4f7d5cfff4f5"),

			responseStorageAccount: &smaccount.Account{
				ID:         uuid.FromStringOrNil("1aa43522-1bd8-11ef-870e-4f7d5cfff4f5"),
				CustomerID: uuid.FromStringOrNil("1a73a632-1bd8-11ef-8c46-4fdca968dac2"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &smaccount.WebhookMessage{
				ID:         uuid.FromStringOrNil("1aa43522-1bd8-11ef-870e-4f7d5cfff4f5"),
				CustomerID: uuid.FromStringOrNil("1a73a632-1bd8-11ef-8c46-4fdca968dac2"),
				TMDelete:   defaultTimestamp,
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
			mockReq.EXPECT().StorageV1AccountDelete(ctx, tt.storageAccountID, 60000).Return(tt.responseStorageAccount, nil)

			res, err := h.StorageAccountDelete(ctx, tt.agent, tt.storageAccountID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_StorageAccountGets(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseStorageAcounts []smaccount.Account
		expectFilters          map[string]string
		expectRes              []*smaccount.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6998ca62-1bd8-11ef-bfe1-f3c47f813931"),
					CustomerID: uuid.FromStringOrNil("69dc78e8-1bd8-11ef-9710-ffa2bc5ebf93"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",

			responseStorageAcounts: []smaccount.Account{
				{
					ID: uuid.FromStringOrNil("6a1a3db8-1bd8-11ef-bffb-8bab4b517f52"),
				},
				{
					ID: uuid.FromStringOrNil("6a5476cc-1bd8-11ef-9863-3b26eb47b0e0"),
				},
			},
			expectFilters: map[string]string{
				"deleted": "false",
			},
			expectRes: []*smaccount.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6a1a3db8-1bd8-11ef-bffb-8bab4b517f52"),
				},
				{
					ID: uuid.FromStringOrNil("6a5476cc-1bd8-11ef-9863-3b26eb47b0e0"),
				},
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

			mockReq.EXPECT().StorageV1AccountGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseStorageAcounts, nil)

			res, err := h.StorageAccountGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_StorageAccountCreate(t *testing.T) {

	tests := []struct {
		name string

		agent      *amagent.Agent
		customerID uuid.UUID

		responseStorageAccount *smaccount.Account
		expectRes              *smaccount.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f2295586-1bd8-11ef-8610-73602171ce63"),
					CustomerID: uuid.FromStringOrNil("f24de752-1bd8-11ef-b438-4361eeff2690"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			customerID: uuid.FromStringOrNil("e9942acc-1bd8-11ef-9c19-33ff4d2cf1ae"),

			responseStorageAccount: &smaccount.Account{
				ID: uuid.FromStringOrNil("f27afcba-1bd8-11ef-a4b8-6f4d6a5ab550"),
			},
			expectRes: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("f27afcba-1bd8-11ef-a4b8-6f4d6a5ab550"),
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

			mockReq.EXPECT().StorageV1AccountCreate(ctx, tt.agent.CustomerID).Return(tt.responseStorageAccount, nil)

			res, err := h.StorageAccountCreate(ctx, tt.agent, tt.customerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
