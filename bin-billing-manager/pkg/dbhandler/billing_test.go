package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/cachehandler"
)

func Test_BillingCreate(t *testing.T) {

	type test struct {
		name string

		billing *billing.Billing

		responseCurTime string
		expectRes       *billing.Billing
	}

	tests := []test{
		{
			name: "have all",

			billing: &billing.Billing{
				ID:               uuid.FromStringOrNil("012b0808-07ae-11ee-956e-a31693cf908b"),
				CustomerID:       uuid.FromStringOrNil("01845d18-07ae-11ee-b9ed-17e7ec2627df"),
				AccountID:        uuid.FromStringOrNil("01eaf474-07ae-11ee-b506-b75479a482cc"),
				Status:           billing.StatusProgressing,
				ReferenceType:    billing.ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("023c6296-07ae-11ee-8674-9f2a42cd114e"),
				CostPerUnit:      2.02,
				CostTotal:        0,
				BillingUnitCount: 30.12,
				TMBillingStart:   "2020-04-18 03:22:18.995000",
				TMBillingEnd:     "2020-04-18 03:22:19.995000",
			},

			responseCurTime: "2023-06-07 03:22:17.995000",
			expectRes: &billing.Billing{
				ID:               uuid.FromStringOrNil("012b0808-07ae-11ee-956e-a31693cf908b"),
				CustomerID:       uuid.FromStringOrNil("01845d18-07ae-11ee-b9ed-17e7ec2627df"),
				AccountID:        uuid.FromStringOrNil("01eaf474-07ae-11ee-b506-b75479a482cc"),
				Status:           billing.StatusProgressing,
				ReferenceType:    billing.ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("023c6296-07ae-11ee-8674-9f2a42cd114e"),
				CostPerUnit:      2.02,
				CostTotal:        0,
				BillingUnitCount: 30.12,
				TMBillingStart:   "2020-04-18 03:22:18.995000",
				TMBillingEnd:     "2020-04-18 03:22:19.995000",
				TMCreate:         "2023-06-07 03:22:17.995000",
				TMUpdate:         DefaultTimeStamp,
				TMDelete:         DefaultTimeStamp,
			},
		},
		{
			name: "empty",

			billing: &billing.Billing{
				ID: uuid.FromStringOrNil("a94849ce-07ae-11ee-a930-9b5e28e2ead9"),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &billing.Billing{
				ID:       uuid.FromStringOrNil("a94849ce-07ae-11ee-a930-9b5e28e2ead9"),
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BillingGet(ctx, tt.billing.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			res, err := h.BillingGet(context.Background(), tt.billing.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			mockCache.EXPECT().BillingGetByReferenceID(ctx, tt.billing.ReferenceID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			resByReference, err := h.BillingGetByReferenceID(ctx, tt.billing.ReferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, resByReference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, resByReference)
			}
		})
	}
}

func Test_BillingGets(t *testing.T) {

	type test struct {
		name     string
		billings []*billing.Billing

		filters map[string]string

		responseCurTime string
		expectRes       []*billing.Billing
	}

	tests := []test{
		{
			name: "normal",
			billings: []*billing.Billing{
				{
					ID:         uuid.FromStringOrNil("a97bade6-07ae-11ee-8916-4f8af853ac9b"),
					CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
				},
				{
					ID:         uuid.FromStringOrNil("a9ab7120-07ae-11ee-9a8d-6f3e025fae7a"),
					CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
				},
			},

			filters: map[string]string{
				"customer_id": "a9db1420-07ae-11ee-ab10-1ffa68fea7d8",
			},

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: []*billing.Billing{
				{
					ID:         uuid.FromStringOrNil("a97bade6-07ae-11ee-8916-4f8af853ac9b"),
					CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
					TMCreate:   "2023-06-08 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("a9ab7120-07ae-11ee-9a8d-6f3e025fae7a"),
					CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
					TMCreate:   "2023-06-08 03:22:17.995000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, c := range tt.billings {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().BillingSet(ctx, gomock.Any())
				if err := h.BillingCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.BillingGets(ctx, 1000, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_BillingSetStatusEnd(t *testing.T) {

	type test struct {
		name    string
		billing *billing.Billing

		id              uuid.UUID
		billingDuration float32
		timestamp       string

		responseCurTime string
		expectRes       *billing.Billing
	}

	tests := []test{
		{
			name: "normal",
			billing: &billing.Billing{
				ID:           uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
				Status:       billing.StatusProgressing,
				CostPerUnit:  10.12,
				CostTotal:    0,
				TMBillingEnd: "",
			},

			id:              uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
			billingDuration: 10.12,
			timestamp:       "2023-06-08 03:22:15.995000",

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &billing.Billing{
				ID:               uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
				Status:           billing.StatusEnd,
				CostPerUnit:      10.12,
				CostTotal:        102.4144,
				BillingUnitCount: 10.12,
				TMBillingEnd:     "2023-06-08 03:22:15.995000",

				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingSetStatusEnd(ctx, tt.id, tt.billingDuration, tt.timestamp); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BillingGet(ctx, tt.billing.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			res, err := h.BillingGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingSetStatus(t *testing.T) {

	type test struct {
		name    string
		billing *billing.Billing

		id     uuid.UUID
		status billing.Status

		responseCurTime string
		expectRes       *billing.Billing
	}

	tests := []test{
		{
			name: "normal",
			billing: &billing.Billing{
				ID:     uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
				Status: billing.StatusProgressing,
			},

			id:     uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
			status: billing.StatusFinished,

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &billing.Billing{
				ID:     uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
				Status: billing.StatusFinished,

				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingSetStatus(ctx, tt.id, tt.status); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BillingGet(ctx, tt.billing.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			res, err := h.BillingGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingDelete(t *testing.T) {

	type test struct {
		name    string
		billing *billing.Billing

		id uuid.UUID

		responseCurTime string
		expectRes       *billing.Billing
	}

	tests := []test{
		{
			name: "normal",
			billing: &billing.Billing{
				ID: uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),
			},

			id: uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),

			responseCurTime: "2023-06-08 03:22:17.995000",
			expectRes: &billing.Billing{
				ID: uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),

				TMCreate: "2023-06-08 03:22:17.995000",
				TMUpdate: "2023-06-08 03:22:17.995000",
				TMDelete: "2023-06-08 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BillingGet(ctx, tt.billing.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			res, err := h.BillingGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
