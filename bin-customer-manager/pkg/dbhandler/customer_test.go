package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

func Test_CustomerCreate(t *testing.T) {

	curTime := time.Date(2024, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name     string
		customer *customer.Customer

		responseCurTime *time.Time
		expectRes       *customer.Customer
	}{
		{
			name: "all",

			customer: &customer.Customer{
				ID:               uuid.FromStringOrNil("0bc5b900-7c65-11ec-a205-3b81594c7376"),
				Name:             "test name",
				Detail:           "test detail",
				Email:            "test@test.com",
				PhoneNumber:      "+821100000001",
				Address:          "somewhere",
				WebhookMethod:    "POST",
				WebhookURI:       "test.com",
				BillingAccountID: uuid.FromStringOrNil("5d7c011c-0e83-11ee-afc0-57978d43b290"),
			},

			responseCurTime: &curTime,
			expectRes: &customer.Customer{
				ID:               uuid.FromStringOrNil("0bc5b900-7c65-11ec-a205-3b81594c7376"),
				Name:             "test name",
				Detail:           "test detail",
				Email:            "test@test.com",
				PhoneNumber:      "+821100000001",
				Address:          "somewhere",
				WebhookMethod:    "POST",
				WebhookURI:       "test.com",
				BillingAccountID: uuid.FromStringOrNil("5d7c011c-0e83-11ee-afc0-57978d43b290"),
				TMCreate:         &curTime,
				TMUpdate:         nil,
				TMDelete:         nil,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).AnyTimes()
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(ctx, tt.customer.ID).Return(nil, fmt.Errorf(""))
			res, err := h.CustomerGet(ctx, tt.customer.ID)
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
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name     string
		customer *customer.Customer

		responseCurTime *time.Time
		expectRes       *customer.Customer
	}{
		{
			name: "normal",
			customer: &customer.Customer{
				ID: uuid.FromStringOrNil("45adb3e8-7c65-11ec-8720-8f643ab80535"),
			},

			responseCurTime: &curTime,
			expectRes: &customer.Customer{
				ID:       uuid.FromStringOrNil("45adb3e8-7c65-11ec-8720-8f643ab80535"),
				Status:   customer.StatusDeleted,
				TMCreate: &curTime,
				TMUpdate: &curTime,
				TMDelete: &curTime,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerGet(ctx, tt.customer.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil).AnyTimes()
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.CustomerDelete(ctx, tt.customer.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerList(t *testing.T) {
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name      string
		customers []*customer.Customer
		size      uint64
		token     string
		filters   map[customer.Field]any

		responseCurTime *time.Time
		expectRes       []*customer.Customer
	}{
		{
			name: "normal",
			customers: []*customer.Customer{
				{
					ID: uuid.FromStringOrNil("500f6624-7c65-11ec-ba0f-8399fe28afb2"),
				},
				{
					ID: uuid.FromStringOrNil("5c4b732e-7c65-11ec-8f09-2720f96bb96d"),
				},
			},
			size:  2,
			token: "2020-04-18T03:22:17.995001Z",
			filters: map[customer.Field]any{
				customer.FieldDeleted: false,
			},

			responseCurTime: &curTime,
			expectRes: []*customer.Customer{
				{
					ID:       uuid.FromStringOrNil("500f6624-7c65-11ec-ba0f-8399fe28afb2"),
					TMCreate: &curTime,
					TMUpdate: nil,
					TMDelete: nil,
				},
				{
					ID:       uuid.FromStringOrNil("5c4b732e-7c65-11ec-8f09-2720f96bb96d"),
					TMCreate: &curTime,
					TMUpdate: nil,
					TMDelete: nil,
				},
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
			ctx := context.Background()

			for _, u := range tt.customers {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().CustomerSet(ctx, gomock.Any())
				if err := h.CustomerCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CustomerList(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if len(res) < len(tt.customers) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", len(tt.customers), len(res))
			}
		})
	}
}

func Test_CustomerUpdate(t *testing.T) {
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name     string
		customer *customer.Customer

		updateFields map[customer.Field]any

		responseCurTime *time.Time
		expectRes       *customer.Customer
	}{
		{
			name: "basic_info",
			customer: &customer.Customer{
				ID:            uuid.FromStringOrNil("a3697e6a-7c72-11ec-8fdf-dbda7d8fab3e"),
				Name:          "test4",
				Detail:        "detail4",
				Email:         "default@test.com",
				PhoneNumber:   "+821100000000",
				Address:       "somewhere",
				WebhookMethod: customer.WebhookMethodGet,
				WebhookURI:    "localhost.com",
			},

			updateFields: map[customer.Field]any{
				customer.FieldName:          "test4 new",
				customer.FieldDetail:        "detail4 new",
				customer.FieldEmail:         "test@test.com",
				customer.FieldPhoneNumber:   "+821100000001",
				customer.FieldAddress:       "middle of nowhere",
				customer.FieldWebhookMethod: customer.WebhookMethodPost,
				customer.FieldWebhookURI:    "test.com",
			},

			responseCurTime: &curTime,
			expectRes: &customer.Customer{
				ID:            uuid.FromStringOrNil("a3697e6a-7c72-11ec-8fdf-dbda7d8fab3e"),
				Name:          "test4 new",
				Detail:        "detail4 new",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "middle of nowhere",
				WebhookMethod: customer.WebhookMethodPost,
				WebhookURI:    "test.com",
				TMCreate:      &curTime,
				TMUpdate:      &curTime,
				TMDelete:      nil,
			},
		},
		{
			name: "empty",
			customer: &customer.Customer{
				ID:            uuid.FromStringOrNil("e778f8b0-7c72-11ec-8605-9f86a3f3debe"),
				Name:          "test4",
				Detail:        "detail4",
				Email:         "default@test.com",
				PhoneNumber:   "+821100000000",
				Address:       "somewhere",
				WebhookMethod: customer.WebhookMethodGet,
				WebhookURI:    "localhost.com",
			},

			updateFields: map[customer.Field]any{
				customer.FieldName:          "",
				customer.FieldDetail:        "",
				customer.FieldEmail:         "",
				customer.FieldPhoneNumber:   "",
				customer.FieldAddress:       "",
				customer.FieldWebhookMethod: customer.WebhookMethodNone,
				customer.FieldWebhookURI:    "",
			},

			responseCurTime: &curTime,
			expectRes: &customer.Customer{
				ID:            uuid.FromStringOrNil("e778f8b0-7c72-11ec-8605-9f86a3f3debe"),
				Name:          "",
				Detail:        "",
				WebhookMethod: "",
				WebhookURI:    "",
				TMCreate:      &curTime,
				TMUpdate:      &curTime,
				TMDelete:      nil,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			if err := h.CustomerUpdate(ctx, tt.customer.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(ctx, gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerUpdateBillingAccountID(t *testing.T) {
	curTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name     string
		customer *customer.Customer

		billingAccountID uuid.UUID

		responseCurTime *time.Time
		expectRes       *customer.Customer
	}{
		{
			"normal",
			&customer.Customer{
				ID:               uuid.FromStringOrNil("fb6946f8-0f8e-11ee-b6e8-0b5d7cba3ef2"),
				Name:             "test7",
				Detail:           "detail7",
				BillingAccountID: uuid.FromStringOrNil("fbd77aba-0f8e-11ee-a59f-cb37ae45541e"),
			},

			uuid.FromStringOrNil("fc1bc1fc-0f8e-11ee-970b-b75ca3799e1f"),

			&curTime,
			&customer.Customer{
				ID:               uuid.FromStringOrNil("fb6946f8-0f8e-11ee-b6e8-0b5d7cba3ef2"),
				Name:             "test7",
				Detail:           "detail7",
				BillingAccountID: uuid.FromStringOrNil("fc1bc1fc-0f8e-11ee-970b-b75ca3799e1f"),
				TMCreate:         &curTime,
				TMUpdate:         &curTime,
				TMDelete:         nil,
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
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			fields := map[customer.Field]any{
				customer.FieldBillingAccountID: tt.billingAccountID,
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerUpdate(ctx, tt.customer.ID, fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().CustomerGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerHardDelete(t *testing.T) {
	curTime := time.Date(2024, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name     string
		customer *customer.Customer

		responseCurTime *time.Time
	}{
		{
			name: "normal",
			customer: &customer.Customer{
				ID:    uuid.FromStringOrNil("aa111111-0000-0000-0000-000000000001"),
				Name:  "hard delete test",
				Email: "harddelete@voipbin.net",
			},

			responseCurTime: &curTime,
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
			ctx := context.Background()

			// create first
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// verify it exists
			mockCache.EXPECT().CustomerGet(ctx, tt.customer.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			res, err := h.CustomerGet(ctx, tt.customer.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Errorf("Wrong match. expect: customer, got: nil")
			}

			// hard delete
			if err := h.CustomerHardDelete(ctx, tt.customer.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// verify it's gone
			mockCache.EXPECT().CustomerGet(ctx, tt.customer.ID).Return(nil, fmt.Errorf(""))
			_, err = h.CustomerGet(ctx, tt.customer.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: error (not found), got: nil")
			}
		})
	}
}
