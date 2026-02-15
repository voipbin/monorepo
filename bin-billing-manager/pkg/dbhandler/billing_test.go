package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

func Test_BillingCreate(t *testing.T) {

	type test struct {
		name string

		billing *billing.Billing

		responseCurTime *time.Time
		expectRes       *billing.Billing
	}

	tmBillingStart := time.Date(2020, 4, 18, 3, 22, 18, 995000000, time.UTC)
	tmBillingEnd := time.Date(2020, 4, 18, 3, 22, 19, 995000000, time.UTC)
	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	tmCreate2 := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "have all",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("012b0808-07ae-11ee-956e-a31693cf908b"),
					CustomerID: uuid.FromStringOrNil("01845d18-07ae-11ee-b9ed-17e7ec2627df"),
				},
				AccountID:         uuid.FromStringOrNil("01eaf474-07ae-11ee-b506-b75479a482cc"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("023c6296-07ae-11ee-8674-9f2a42cd114e"),
				RateCreditPerUnit: 6000,
				TMBillingStart:    &tmBillingStart,
				TMBillingEnd:      &tmBillingEnd,
			},

			responseCurTime: &tmCreate,
			expectRes: &billing.Billing{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("012b0808-07ae-11ee-956e-a31693cf908b"),
					CustomerID: uuid.FromStringOrNil("01845d18-07ae-11ee-b9ed-17e7ec2627df"),
				},
				AccountID:         uuid.FromStringOrNil("01eaf474-07ae-11ee-b506-b75479a482cc"),
				TransactionType:   billing.TransactionTypeUsage,
				Status:            billing.StatusProgressing,
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("023c6296-07ae-11ee-8674-9f2a42cd114e"),
				RateCreditPerUnit: 6000,
				TMBillingStart:    &tmBillingStart,
				TMBillingEnd:      &tmBillingEnd,
				TMCreate:          &tmCreate,
				TMUpdate:          nil,
				TMDelete:          nil,
			},
		},
		{
			name: "empty",

			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a94849ce-07ae-11ee-a930-9b5e28e2ead9"),
				},
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a94849ce-07ae-11ee-a930-9b5e28e2ead9"),
			},

			responseCurTime: &tmCreate2,
			expectRes: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a94849ce-07ae-11ee-a930-9b5e28e2ead9"),
				},
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a94849ce-07ae-11ee-a930-9b5e28e2ead9"),
				TMCreate:      &tmCreate2,
				TMUpdate:      nil,
				TMDelete:      nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

func Test_BillingList(t *testing.T) {

	type test struct {
		name     string
		billings []*billing.Billing

		filters map[billing.Field]any

		responseCurTime *time.Time
		expectRes       []*billing.Billing
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			billings: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a97bade6-07ae-11ee-8916-4f8af853ac9b"),
						CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
					},
					ReferenceType: billing.ReferenceTypeSMS,
					ReferenceID:   uuid.FromStringOrNil("a97bade6-07ae-11ee-8916-4f8af853ac9b"),
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a9ab7120-07ae-11ee-9a8d-6f3e025fae7a"),
						CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
					},
					ReferenceType: billing.ReferenceTypeSMS,
					ReferenceID:   uuid.FromStringOrNil("a9ab7120-07ae-11ee-9a8d-6f3e025fae7a"),
				},
			},

			filters: map[billing.Field]any{
				billing.FieldCustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
			},

			responseCurTime: &tmCreate,
			expectRes: []*billing.Billing{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a97bade6-07ae-11ee-8916-4f8af853ac9b"),
						CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
					},
					ReferenceType: billing.ReferenceTypeSMS,
					ReferenceID:   uuid.FromStringOrNil("a97bade6-07ae-11ee-8916-4f8af853ac9b"),
					TMCreate:      &tmCreate,
					TMUpdate:      nil,
					TMDelete:      nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a9ab7120-07ae-11ee-9a8d-6f3e025fae7a"),
						CustomerID: uuid.FromStringOrNil("a9db1420-07ae-11ee-ab10-1ffa68fea7d8"),
					},
					ReferenceType: billing.ReferenceTypeSMS,
					ReferenceID:   uuid.FromStringOrNil("a9ab7120-07ae-11ee-9a8d-6f3e025fae7a"),
					TMCreate:      &tmCreate,
					TMUpdate:      nil,
					TMDelete:      nil,
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
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().BillingSet(ctx, gomock.Any())
				if err := h.BillingCreate(ctx, c); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.BillingList(ctx, 1000, utilhandler.TimeGetCurTime(), tt.filters)
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

		id                    uuid.UUID
		billableUnits         int
		usageDuration         int
		amountToken           int64
		amountCredit          int64
		balanceTokenSnapshot  int64
		balanceCreditSnapshot int64
		timestamp             *time.Time

		responseCurTime *time.Time
		expectRes       *billing.Billing
	}

	tmBillingEnd := time.Date(2023, 6, 8, 3, 22, 15, 995000000, time.UTC)
	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
				},
				ReferenceType:     billing.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
				Status:            billing.StatusProgressing,
				RateCreditPerUnit: 6000,
				TMBillingEnd:      nil,
			},

			id:                    uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
			billableUnits:         2,
			usageDuration:         65,
			amountToken:           0,
			amountCredit:          -12000,
			balanceTokenSnapshot:  0,
			balanceCreditSnapshot: 9988000,
			timestamp:             &tmBillingEnd,

			responseCurTime: &tmCreate,
			expectRes: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
				},
				ReferenceType:         billing.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("1441f0ac-07b1-11ee-a5d8-cb0119ac5064"),
				Status:                billing.StatusEnd,
				RateCreditPerUnit:     6000,
				BillableUnits:         2,
				UsageDuration:         65,
				AmountToken:           0,
				AmountCredit:          -12000,
				BalanceTokenSnapshot:  0,
				BalanceCreditSnapshot: 9988000,
				TMBillingEnd:          &tmBillingEnd,

				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingSetStatusEnd(ctx, tt.id, tt.billableUnits, tt.usageDuration, tt.amountToken, tt.amountCredit, tt.balanceTokenSnapshot, tt.balanceCreditSnapshot, tt.timestamp); err != nil {
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

		responseCurTime *time.Time
		expectRes       *billing.Billing
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
				},
				ReferenceType: billing.ReferenceTypeNumber,
				ReferenceID:   uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
				Status:        billing.StatusProgressing,
			},

			id:     uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
			status: billing.StatusFinished,

			responseCurTime: &tmCreate,
			expectRes: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
				},
				ReferenceType: billing.ReferenceTypeNumber,
				ReferenceID:   uuid.FromStringOrNil("1d30c8de-07b4-11ee-b80c-c7c0f2007941"),
				Status:        billing.StatusFinished,

				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

		responseCurTime *time.Time
		expectRes       *billing.Billing
	}

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),
				},
				ReferenceType: billing.ReferenceTypeNumberRenew,
				ReferenceID:   uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),
			},

			id: uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),

			responseCurTime: &tmCreate,
			expectRes: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),
				},
				ReferenceType: billing.ReferenceTypeNumberRenew,
				ReferenceID:   uuid.FromStringOrNil("699bdc72-07b4-11ee-83bc-db5341c91127"),

				TMCreate: &tmCreate,
				TMUpdate: &tmCreate,
				TMDelete: &tmCreate,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
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

func Test_BillingGetByReferenceTypeAndID(t *testing.T) {

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		billing *billing.Billing

		referenceType billing.ReferenceType
		referenceID   uuid.UUID

		expectRes *billing.Billing
	}{
		{
			name: "found",
			billing: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1a2b3c4-d5e6-11ee-abcd-1234567890ab"),
				},
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("f2a2b3c4-d5e6-11ee-abcd-1234567890ab"),
			},

			referenceType: billing.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("f2a2b3c4-d5e6-11ee-abcd-1234567890ab"),

			expectRes: &billing.Billing{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1a2b3c4-d5e6-11ee-abcd-1234567890ab"),
				},
				ReferenceType: billing.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("f2a2b3c4-d5e6-11ee-abcd-1234567890ab"),
				TMCreate:      &tmCreate,
				TMUpdate:      nil,
				TMDelete:      nil,
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

			mockUtil.EXPECT().TimeNow().Return(&tmCreate)
			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			if err := h.BillingCreate(ctx, tt.billing); err != nil {
				t.Fatalf("Could not create billing: %v", err)
			}

			mockCache.EXPECT().BillingSet(ctx, gomock.Any())
			res, err := h.BillingGetByReferenceTypeAndID(ctx, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_BillingGetByReferenceTypeAndID_not_found(t *testing.T) {
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

	_, err := h.BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeCall, uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"))
	if err != ErrNotFound {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_BillingGetByReferenceTypeAndID_deleted_not_returned(t *testing.T) {
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

	tmCreate := time.Date(2023, 6, 8, 3, 22, 17, 995000000, time.UTC)

	b := &billing.Billing{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("e1a2b3c4-d5e6-11ee-abcd-ffffffffffff"),
		},
		ReferenceType: billing.ReferenceTypeSMS,
		ReferenceID:   uuid.FromStringOrNil("e2a2b3c4-d5e6-11ee-abcd-ffffffffffff"),
	}

	// create
	mockUtil.EXPECT().TimeNow().Return(&tmCreate)
	mockCache.EXPECT().BillingSet(ctx, gomock.Any())
	if err := h.BillingCreate(ctx, b); err != nil {
		t.Fatalf("Could not create billing: %v", err)
	}

	// delete it
	mockUtil.EXPECT().TimeNow().Return(&tmCreate)
	mockCache.EXPECT().BillingSet(ctx, gomock.Any())
	if err := h.BillingDelete(ctx, b.ID); err != nil {
		t.Fatalf("Could not delete billing: %v", err)
	}

	// should not be found by reference type+id (deleted records excluded)
	_, err := h.BillingGetByReferenceTypeAndID(ctx, billing.ReferenceTypeSMS, b.ReferenceID)
	if err != ErrNotFound {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

// Note: AccountSubtractBalanceWithCheck uses SELECT ... FOR UPDATE which is
// MySQL-specific and not supported by SQLite. This function is tested through
// the accounthandler mock-based tests instead.
