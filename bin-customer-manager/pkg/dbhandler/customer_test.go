package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/cachehandler"
)

func Test_CustomerCreate(t *testing.T) {

	tests := []struct {
		name     string
		customer *customer.Customer

		responseCurTime string
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

			responseCurTime: "2020-04-18 03:22:17.995000",
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
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         DefaultTimeStamp,
				TMDelete:         DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

	tests := []struct {
		name     string
		customer *customer.Customer

		responseCurTime string
		expectRes       *customer.Customer
	}{
		{
			name: "normal",
			customer: &customer.Customer{
				ID: uuid.FromStringOrNil("45adb3e8-7c65-11ec-8720-8f643ab80535"),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &customer.Customer{
				ID:       uuid.FromStringOrNil("45adb3e8-7c65-11ec-8720-8f643ab80535"),
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: "2020-04-18 03:22:17.995000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerGet(ctx, tt.customer.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil).AnyTimes()
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

func Test_CustomerGets(t *testing.T) {

	tests := []struct {
		name      string
		customers []*customer.Customer
		size      uint64
		filters   map[string]string

		responseCurTime string
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
			size: 2,
			filters: map[string]string{
				"deleted": "false",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: []*customer.Customer{
				{
					ID:       uuid.FromStringOrNil("500f6624-7c65-11ec-ba0f-8399fe28afb2"),
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("5c4b732e-7c65-11ec-8f09-2720f96bb96d"),
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
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

			// clean test database users
			_ = cleanTestDBCustomers()

			for _, u := range tt.customers {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().CustomerSet(ctx, gomock.Any())
				if err := h.CustomerCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.CustomerGets(ctx, tt.size, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerSetBasicInfo(t *testing.T) {

	tests := []struct {
		name     string
		customer *customer.Customer

		customerName  string
		detail        string
		email         string
		phoneNumber   string
		address       string
		webhookMethod customer.WebhookMethod
		webhookURI    string

		responseCurTime string
		expectRes       *customer.Customer
	}{
		{
			name: "all",
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

			customerName:  "test4 new",
			detail:        "detail4 new",
			email:         "test@test.com",
			phoneNumber:   "+821100000001",
			address:       "middle of nowhere",
			webhookMethod: customer.WebhookMethodPost,
			webhookURI:    "test.com",

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &customer.Customer{
				ID:            uuid.FromStringOrNil("a3697e6a-7c72-11ec-8fdf-dbda7d8fab3e"),
				Name:          "test4 new",
				Detail:        "detail4 new",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "middle of nowhere",
				WebhookMethod: customer.WebhookMethodPost,
				WebhookURI:    "test.com",
				TMCreate:      "2020-04-18 03:22:17.995000",
				TMUpdate:      "2020-04-18 03:22:17.995000",
				TMDelete:      DefaultTimeStamp,
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

			customerName:  "",
			detail:        "",
			email:         "",
			phoneNumber:   "",
			address:       "",
			webhookMethod: customer.WebhookMethodNone,
			webhookURI:    "",

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &customer.Customer{
				ID:            uuid.FromStringOrNil("e778f8b0-7c72-11ec-8605-9f86a3f3debe"),
				Name:          "",
				Detail:        "",
				WebhookMethod: "",
				WebhookURI:    "",
				TMCreate:      "2020-04-18 03:22:17.995000",
				TMUpdate:      "2020-04-18 03:22:17.995000",
				TMDelete:      DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			if err := h.CustomerCreate(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(ctx, gomock.Any()).Return(nil)
			if err := h.CustomerSetBasicInfo(
				ctx,
				tt.customer.ID,
				tt.customerName,
				tt.detail,
				tt.email,
				tt.phoneNumber,
				tt.address,
				tt.webhookMethod,
				tt.webhookURI,
			); err != nil {
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

func Test_CustomerSetBillingAccountID(t *testing.T) {

	tests := []struct {
		name     string
		customer *customer.Customer

		billingAccountID uuid.UUID

		responseCurTime string
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

			"2020-04-18 03:22:17.995000",
			&customer.Customer{
				ID:               uuid.FromStringOrNil("fb6946f8-0f8e-11ee-b6e8-0b5d7cba3ef2"),
				Name:             "test7",
				Detail:           "detail7",
				BillingAccountID: uuid.FromStringOrNil("fc1bc1fc-0f8e-11ee-970b-b75ca3799e1f"),
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         "2020-04-18 03:22:17.995000",
				TMDelete:         DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerCreate(context.Background(), tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.CustomerSetBillingAccountID(ctx, tt.customer.ID, tt.billingAccountID); err != nil {
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
