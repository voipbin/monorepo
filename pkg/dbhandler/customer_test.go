package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/cachehandler"
)

func Test_CustomerCreate(t *testing.T) {

	tests := []struct {
		name      string
		customer  *customer.Customer
		expectRes *customer.Customer
	}{
		{
			"normal",
			&customer.Customer{
				ID:            uuid.FromStringOrNil("0bc5b900-7c65-11ec-a205-3b81594c7376"),
				Username:      "test",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test name",
				Detail:        "test detail",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
				LineSecret:    "5b8a5776-e613-11ec-b218-ffe0383dd0f2",
				LineToken:     "5fa39322-e613-11ec-a6e0-fb39c4836722",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("6a6443a4-7c70-11ec-9635-abbaf773da29"),
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&customer.Customer{
				ID:            uuid.FromStringOrNil("0bc5b900-7c65-11ec-a205-3b81594c7376"),
				Username:      "test",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test name",
				Detail:        "test detail",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
				LineSecret:    "5b8a5776-e613-11ec-b218-ffe0383dd0f2",
				LineToken:     "5fa39322-e613-11ec-a6e0-fb39c4836722",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("6a6443a4-7c70-11ec-9635-abbaf773da29"),
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).AnyTimes()
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(gomock.Any(), tt.customer.ID).Return(nil, fmt.Errorf(""))
			res, err := h.CustomerGet(context.Background(), tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name      string
		customer  *customer.Customer
		expectRes *customer.Customer
	}{
		{
			"test normal",
			&customer.Customer{
				ID:           uuid.FromStringOrNil("45adb3e8-7c65-11ec-8720-8f643ab80535"),
				Username:     "test",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
			&customer.Customer{
				ID:            uuid.FromStringOrNil("45adb3e8-7c65-11ec-8720-8f643ab80535"),
				Username:      "test",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().CustomerGet(gomock.Any(), tt.customer.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.CustomerDelete(ctx, tt.customer.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CustomerGet(context.Background(), tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectRes.TMDelete = res.TMDelete
			if res.TMDelete == "" || !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func TestCustomerGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name      string
		customers []*customer.Customer
		size      uint64
		expectRes []*customer.Customer
	}{
		{
			"normal",
			[]*customer.Customer{
				{
					ID:           uuid.FromStringOrNil("500f6624-7c65-11ec-ba0f-8399fe28afb2"),
					Username:     "test2",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TMCreate:     "2020-04-18T03:22:17.995000",
				},
				{
					ID:           uuid.FromStringOrNil("5c4b732e-7c65-11ec-8f09-2720f96bb96d"),
					Username:     "test3",
					PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					TMCreate:     "2020-04-18T03:22:17.995000",
				},
			},
			2,
			[]*customer.Customer{
				{
					ID:            uuid.FromStringOrNil("500f6624-7c65-11ec-ba0f-8399fe28afb2"),
					Username:      "test2",
					PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					PermissionIDs: []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("5c4b732e-7c65-11ec-8f09-2720f96bb96d"),
					Username:      "test3",
					PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
					PermissionIDs: []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// clean test database users
			_ = cleanTestDBCustomers()

			h := NewHandler(dbTest, mockCache)

			for _, u := range tt.customers {
				mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any())
				if err := h.CustomerCreate(context.Background(), u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CustomerGets(ctx, tt.size, GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerGetByUsername(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name      string
		customer  *customer.Customer
		expectRes *customer.Customer
	}{
		{
			"test normal",
			&customer.Customer{
				ID:           uuid.FromStringOrNil("6923c328-7c72-11ec-9624-efe2285c5992"),
				Username:     "test7",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18T03:22:17.995000",
			},
			&customer.Customer{
				ID:            uuid.FromStringOrNil("6923c328-7c72-11ec-9624-efe2285c5992"),
				Username:      "test7",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any())
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CustomerGetByUsername(context.Background(), tt.customer.Username)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerSetBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name     string
		customer *customer.Customer

		userName      string
		detail        string
		webhookMethod customer.WebhookMethod
		webhookURI    string

		expectRes *customer.Customer
	}{
		{
			"normal",
			&customer.Customer{
				ID:           uuid.FromStringOrNil("a3697e6a-7c72-11ec-8fdf-dbda7d8fab3e"),
				Username:     "abc0df18-7c72-11ec-8b18-5f22d10c7abd",
				Name:         "test4",
				Detail:       "detail4",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18 03:22:17.995000",
			},
			"test4 new",
			"detail4 new",
			"",
			"",

			&customer.Customer{
				ID:            uuid.FromStringOrNil("a3697e6a-7c72-11ec-8fdf-dbda7d8fab3e"),
				Username:      "abc0df18-7c72-11ec-8b18-5f22d10c7abd",
				Name:          "test4 new",
				Detail:        "detail4 new",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},
		},
		{
			"have webhook",
			&customer.Customer{
				ID:           uuid.FromStringOrNil("e778f8b0-7c72-11ec-8605-9f86a3f3debe"),
				Username:     "e778f8b0-7c72-11ec-8605-9f86a3f3debe",
				Name:         "name",
				Detail:       "defail",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				TMCreate:     "2020-04-18 03:22:17.995000",
			},
			"name new",
			"detail new",
			"POST",
			"test.com",

			&customer.Customer{
				ID:            uuid.FromStringOrNil("e778f8b0-7c72-11ec-8605-9f86a3f3debe"),
				Username:      "e778f8b0-7c72-11ec-8605-9f86a3f3debe",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "name new",
				Detail:        "detail new",
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerSetBasicInfo(ctx, tt.customer.ID, tt.userName, tt.detail, tt.webhookMethod, tt.webhookURI); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerSetPermissionIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name     string
		customer *customer.Customer

		permissionIDs []uuid.UUID

		expectRes *customer.Customer
	}{
		{
			"normal",
			&customer.Customer{
				ID:            uuid.FromStringOrNil("300d0788-7c73-11ec-930a-4f107f248651"),
				Username:      "300d0788-7c73-11ec-930a-4f107f248651",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test5",
				Detail:        "detail5",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},

			[]uuid.UUID{
				uuid.FromStringOrNil("8b61102a-7c73-11ec-83dd-73e46fe6b1da"),
			},

			&customer.Customer{
				ID:           uuid.FromStringOrNil("300d0788-7c73-11ec-930a-4f107f248651"),
				Username:     "300d0788-7c73-11ec-930a-4f107f248651",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:         "test5",
				Detail:       "detail5",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("8b61102a-7c73-11ec-83dd-73e46fe6b1da"),
				},
				TMCreate: "2020-04-18 03:22:17.995000",
			},
		},
		{
			"set to no permssion",
			&customer.Customer{
				ID:           uuid.FromStringOrNil("bc1c9af4-7c73-11ec-81c8-538f743bc72f"),
				Username:     "300d0788-7c73-11ec-930a-4f107f248651",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:         "test5",
				Detail:       "detail5",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("bc4b0916-7c73-11ec-a413-6341b2b8f254"),
				},
				TMCreate: "2020-04-18 03:22:17.995000",
			},

			[]uuid.UUID{},

			&customer.Customer{
				ID:            uuid.FromStringOrNil("bc1c9af4-7c73-11ec-81c8-538f743bc72f"),
				Username:      "300d0788-7c73-11ec-930a-4f107f248651",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test5",
				Detail:        "detail5",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},
		},
		{
			"set 2 items",
			&customer.Customer{
				ID:           uuid.FromStringOrNil("db4489dc-7c73-11ec-841c-ffb70daba8fb"),
				Username:     "db4489dc-7c73-11ec-841c-ffb70daba8fb",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:         "test5",
				Detail:       "detail5",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("bc4b0916-7c73-11ec-a413-6341b2b8f254"),
				},
				TMCreate: "2020-04-18 03:22:17.995000",
			},

			[]uuid.UUID{
				uuid.FromStringOrNil("db6ea6ea-7c73-11ec-add0-d7584d01f278"),
				uuid.FromStringOrNil("dba3c276-7c73-11ec-8157-bb712a7969d0"),
			},

			&customer.Customer{
				ID:           uuid.FromStringOrNil("db4489dc-7c73-11ec-841c-ffb70daba8fb"),
				Username:     "db4489dc-7c73-11ec-841c-ffb70daba8fb",
				PasswordHash: "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:         "test5",
				Detail:       "detail5",
				PermissionIDs: []uuid.UUID{
					uuid.FromStringOrNil("db6ea6ea-7c73-11ec-add0-d7584d01f278"),
					uuid.FromStringOrNil("dba3c276-7c73-11ec-8157-bb712a7969d0"),
				},
				TMCreate: "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerSetPermissionIDs(ctx, tt.customer.ID, tt.permissionIDs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCustomerSetPasswordHash(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name     string
		customer *customer.Customer

		passwordHash string

		expectRes *customer.Customer
	}{
		{
			"normal",
			&customer.Customer{
				ID:            uuid.FromStringOrNil("2d08b194-7c74-11ec-b055-e757dd189346"),
				Username:      "2d08b194-7c74-11ec-b055-e757dd189346",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test6",
				Detail:        "detail6",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},

			"ttttttttttiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",

			&customer.Customer{
				ID:            uuid.FromStringOrNil("2d08b194-7c74-11ec-b055-e757dd189346"),
				Username:      "2d08b194-7c74-11ec-b055-e757dd189346",
				PasswordHash:  "ttttttttttiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test6",
				Detail:        "detail6",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerSetPasswordHash(ctx, tt.customer.ID, tt.passwordHash); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSetLineInfo(t *testing.T) {
	tests := []struct {
		name     string
		customer *customer.Customer

		customerID uuid.UUID
		lineSecret string
		lineToken  string

		expectRes *customer.Customer
	}{
		{
			"normal",
			&customer.Customer{
				ID:            uuid.FromStringOrNil("9df40078-e616-11ec-97c4-aba60c3dbfba"),
				Username:      "9df40078-e616-11ec-97c4-aba60c3dbfba",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test6",
				Detail:        "detail6",
				LineSecret:    "a58ea6e4-e616-11ec-9fc6-b30ddcefdfae",
				LineToken:     "a5b2a300-e616-11ec-84e3-2368f0e80458",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},

			uuid.FromStringOrNil("9df40078-e616-11ec-97c4-aba60c3dbfba"),
			"a5d629ba-e616-11ec-a781-d75d16026f0a",
			"a5f9bb78-e616-11ec-af5c-738f2bd772f2",

			&customer.Customer{
				ID:            uuid.FromStringOrNil("9df40078-e616-11ec-97c4-aba60c3dbfba"),
				Username:      "9df40078-e616-11ec-97c4-aba60c3dbfba",
				PasswordHash:  "sifD7dbCmUiBA4XqRMpZce8Bvuz8U5Wil7fwCcH8fhezEPwSNopzO",
				Name:          "test6",
				Detail:        "detail6",
				LineSecret:    "a5d629ba-e616-11ec-a781-d75d16026f0a",
				LineToken:     "a5f9bb78-e616-11ec-af5c-738f2bd772f2",
				PermissionIDs: []uuid.UUID{},
				TMCreate:      "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerSetLineInfo(ctx, tt.customer.ID, tt.lineSecret, tt.lineToken); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
