package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/cachehandler"
)

func Test_NumberCreate(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	type test struct {
		name   string
		number *number.Number

		responseCurTime *time.Time
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"test normal",
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
					CustomerID: uuid.FromStringOrNil("31a1ca10-7ff3-11ec-80f5-83db3c8e951b"),
				},
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			&curTime,
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
					CustomerID: uuid.FromStringOrNil("31a1ca10-7ff3-11ec-80f5-83db3c8e951b"),
				},
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          &curTime,
				TMRenew:             &curTime,
				TMCreate:            &curTime,
				TMUpdate:            nil,
				TMDelete:            nil,
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
			mockCache.EXPECT().NumberSet(ctx, gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(ctx, tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(ctx, gomock.Any())
			res, err := h.NumberGet(ctx, tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectNumber, res)
			}
		})
	}
}

func Test_NumberList(t *testing.T) {

	curTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	type test struct {
		name    string
		numbers []*number.Number

		filters map[number.Field]any

		responseCurTime *time.Time

		expectRes []*number.Number
	}

	tests := []test{
		{
			"normal",
			[]*number.Number{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ca3108a0-eca6-11ee-b06b-4763d2112b09"),
						CustomerID: uuid.FromStringOrNil("ca8a717e-eca6-11ee-8067-1785c729a82f"),
					},
					Number: "+1234567890",
				},
			},

			map[number.Field]any{
				number.FieldCustomerID: uuid.FromStringOrNil("ca8a717e-eca6-11ee-8067-1785c729a82f"),
				number.FieldDeleted:    false,
			},

			&curTime,

			[]*number.Number{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ca3108a0-eca6-11ee-b06b-4763d2112b09"),
						CustomerID: uuid.FromStringOrNil("ca8a717e-eca6-11ee-8067-1785c729a82f"),
					},
					Number:     "+1234567890",
					TMPurchase: &curTime,
					TMRenew:    &curTime,
					TMCreate:   &curTime,
					TMUpdate:   nil,
					TMDelete:   nil,
				},
			},
		},
		{
			"empty",
			[]*number.Number{},

			map[number.Field]any{
				number.FieldDeleted:    false,
				number.FieldCustomerID: uuid.FromStringOrNil("f0395124-eca6-11ee-919f-9b86454807ab"),
			},

			&curTime,
			[]*number.Number{},
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

			// creates numbers for test
			for i := 0; i < len(tt.numbers); i++ {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				_ = h.NumberCreate(context.Background(), tt.numbers[i])
			}

			res, err := h.NumberList(context.Background(), 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil {
				t.Errorf("Expected non-nil slice, got nil")
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberDelete(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	type test struct {
		name            string
		number          *number.Number
		responseCurTime *time.Time
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
					CustomerID: uuid.FromStringOrNil("6884d8f6-7ff3-11ec-8b5c-d3aa777ad672"),
				},
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			&curTime,
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
					CustomerID: uuid.FromStringOrNil("6884d8f6-7ff3-11ec-8b5c-d3aa777ad672"),
				},
				Number:              "+821021656521",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusDeleted,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          &curTime,
				TMRenew:             &curTime,
				TMCreate:            &curTime,
				TMUpdate:            &curTime,
				TMDelete:            &curTime,
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
				cache:       mockCache,
				db:          dbTest,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			if err := h.NumberDelete(ctx, tt.number.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func Test_NumberUpdate(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	type test struct {
		name   string
		num    *number.Number

		updateFields map[number.Field]any

		responseCurTime *time.Time
		expectNumber    *number.Number
	}

	tests := []test{
		{
			name: "normal",
			num: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
					CustomerID: uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
				},
				CallFlowID:          uuid.FromStringOrNil("5293ec2e-881a-11ec-a3bd-bbda5d0724de"),
				MessageFlowID:       uuid.FromStringOrNil("734faf76-20a2-11ee-914a-7bd4f24e615d"),
				Number:              "+821021656521",
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			updateFields: map[number.Field]any{
				number.FieldCallFlowID:    uuid.FromStringOrNil("23ff7078-20a2-11ee-934c-8371c4d02f71"),
				number.FieldMessageFlowID: uuid.FromStringOrNil("24477634-20a2-11ee-b62f-4f341e77043b"),
				number.FieldName:          "update name",
				number.FieldDetail:        "update detail",
			},

			responseCurTime: &curTime,
			expectNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
					CustomerID: uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
				},
				CallFlowID:          uuid.FromStringOrNil("23ff7078-20a2-11ee-934c-8371c4d02f71"),
				MessageFlowID:       uuid.FromStringOrNil("24477634-20a2-11ee-b62f-4f341e77043b"),
				Number:              "+821021656521",
				Name:                "update name",
				Detail:              "update detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          &curTime,
				TMRenew:             &curTime,
				TMCreate:            &curTime,
				TMUpdate:            &curTime,
				TMDelete:            nil,
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
				cache:       mockCache,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.num.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.num); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdate(ctx, tt.num.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.NumberGet(context.Background(), tt.num.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func Test_NumberUpdateFlowID(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	type test struct {
		name   string
		num    *number.Number

		updateFields map[number.Field]any

		responseCurTime *time.Time
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4a7c7d2a-a85f-11ec-8d15-9730036800e5"),
					CustomerID: uuid.FromStringOrNil("4aa698da-a85f-11ec-a93a-5fbf7b8302db"),
				},
				Number:              "+821021656521",
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			map[number.Field]any{
				number.FieldCallFlowID:    uuid.FromStringOrNil("4acedf84-a85f-11ec-bdc6-27902c5c6987"),
				number.FieldMessageFlowID: uuid.FromStringOrNil("4af49f4e-a85f-11ec-ad06-676681d45adb"),
			},

			&curTime,
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4a7c7d2a-a85f-11ec-8d15-9730036800e5"),
					CustomerID: uuid.FromStringOrNil("4aa698da-a85f-11ec-a93a-5fbf7b8302db"),
				},
				Number:              "+821021656521",
				CallFlowID:          uuid.FromStringOrNil("4acedf84-a85f-11ec-bdc6-27902c5c6987"),
				MessageFlowID:       uuid.FromStringOrNil("4af49f4e-a85f-11ec-ad06-676681d45adb"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          &curTime,
				TMRenew:             &curTime,
				TMCreate:            &curTime,
				TMUpdate:            &curTime,
				TMDelete:            nil,
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
				cache:       mockCache,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.num.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.num); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdate(ctx, tt.num.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.NumberGet(context.Background(), tt.num.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func Test_NumberUpdateCallFlowID(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	type test struct {
		name   string
		num    *number.Number

		updateFields map[number.Field]any

		responseCurTime *time.Time
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"update callflow",
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("37357400-8817-11ec-9616-0f38be341833"),
					CustomerID: uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
				},
				Number:              "+821021656521",
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			map[number.Field]any{
				number.FieldCallFlowID: uuid.FromStringOrNil("535c0ca4-8801-11ec-accb-7bd692b1c078"),
			},

			&curTime,
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("37357400-8817-11ec-9616-0f38be341833"),
					CustomerID: uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
				},
				Number:              "+821021656521",
				CallFlowID:          uuid.FromStringOrNil("535c0ca4-8801-11ec-accb-7bd692b1c078"),
				MessageFlowID:       uuid.Nil,
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          &curTime,
				TMRenew:             &curTime,
				TMCreate:            &curTime,
				TMUpdate:            &curTime,
				TMDelete:            nil,
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
				cache:       mockCache,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.num.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.num); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdate(ctx, tt.num.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.NumberGet(context.Background(), tt.num.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func Test_NumberUpdateMessageFlowID(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	type test struct {
		name   string
		num    *number.Number

		updateFields map[number.Field]any

		responseCurTime *time.Time
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b37a0bae-a85e-11ec-b666-7fce3ed0d0d5"),
					CustomerID: uuid.FromStringOrNil("b3e74570-a85e-11ec-a53b-331bdfd1d2f3"),
				},
				Number:              "+821021656521",
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			map[number.Field]any{
				number.FieldMessageFlowID: uuid.FromStringOrNil("b416b062-a85e-11ec-a230-7f3aae198503"),
			},

			&curTime,
			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b37a0bae-a85e-11ec-b666-7fce3ed0d0d5"),
					CustomerID: uuid.FromStringOrNil("b3e74570-a85e-11ec-a53b-331bdfd1d2f3"),
				},
				Number:              "+821021656521",
				MessageFlowID:       uuid.FromStringOrNil("b416b062-a85e-11ec-a230-7f3aae198503"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          &curTime,
				TMRenew:             &curTime,
				TMCreate:            &curTime,
				TMUpdate:            &curTime,
				TMDelete:            nil,
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
				cache:       mockCache,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.num.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.num); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdate(ctx, tt.num.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.NumberGet(context.Background(), tt.num.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func Test_NumberSetTMRenew(t *testing.T) {

	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)
	curTimeRenew := time.Date(2021, 2, 27, 18, 26, 49, 0, time.UTC)

	type test struct {
		name string
		num  *number.Number

		responseCurTime      *time.Time
		responseCurTimeRenew *time.Time
		expectRes            *number.Number
	}

	tests := []test{
		{
			name: "normal",
			num: &number.Number{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),
				},
			},

			responseCurTime:      &curTime,
			responseCurTimeRenew: &curTimeRenew,
			expectRes: &number.Number{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),
				},
				TMPurchase: &curTime,
				TMRenew:    &curTimeRenew,
				TMCreate:   &curTime,
				TMUpdate:   &curTimeRenew,
				TMDelete:   nil,
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
				cache:       mockCache,
				db:          dbTest,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.num); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTimeRenew)
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())

			updateFields := map[number.Field]any{
				number.FieldTMRenew: tt.responseCurTimeRenew,
			}
			if err := h.NumberUpdate(ctx, tt.num.ID, updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.num.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.num.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberListByTMRenew(t *testing.T) {

	curTime1 := time.Date(2020, 4, 10, 18, 26, 49, 0, time.UTC)
	curTime2 := time.Date(2020, 4, 11, 18, 26, 49, 0, time.UTC)
	curTime3 := time.Date(2020, 4, 12, 18, 26, 49, 0, time.UTC)
	curTime4 := time.Date(2020, 4, 13, 18, 26, 49, 0, time.UTC)

	type test struct {
		name    string
		numbers []number.Number

		id      uuid.UUID
		tmRenew string
		size    uint64
		filters map[number.Field]any

		responseCurTimes []*time.Time
		expectRes        []*number.Number
	}

	tests := []test{
		{
			name: "normal",
			numbers: []number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("9356093a-144c-11ee-b0ca-fbaf4f96747c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("93884aa8-144c-11ee-a261-eb324d4a94ab"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("93b536da-144c-11ee-8e04-5f11847ed981"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("93ec6588-144c-11ee-ae23-cb2a64c0f80a"),
					},
				},
			},

			id:      uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),
			tmRenew: "2020-04-12T20:26:49.000Z",
			size:    100,
			filters: map[number.Field]any{
				number.FieldDeleted: false,
			},

			responseCurTimes: []*time.Time{
				&curTime1,
				&curTime2,
				&curTime3,
				&curTime4,
			},
			expectRes: []*number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("93b536da-144c-11ee-8e04-5f11847ed981"),
					},
					TMPurchase: &curTime3,
					TMRenew:    &curTime3,
					TMCreate:   &curTime3,
					TMUpdate:   nil,
					TMDelete:   nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("93884aa8-144c-11ee-a261-eb324d4a94ab"),
					},
					TMPurchase: &curTime2,
					TMRenew:    &curTime2,
					TMCreate:   &curTime2,
					TMUpdate:   nil,
					TMDelete:   nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("9356093a-144c-11ee-b0ca-fbaf4f96747c"),
					},
					TMPurchase: &curTime1,
					TMRenew:    &curTime1,
					TMCreate:   &curTime1,
					TMUpdate:   nil,
					TMDelete:   nil,
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
				cache:       mockCache,
				db:          dbTest,
			}
			ctx := context.Background()

			for i, n := range tt.numbers {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTimes[i])
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				if err := h.NumberCreate(ctx, &n); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.NumberGetsByTMRenew(ctx, tt.tmRenew, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberGetExistingNumbers_Empty(t *testing.T) {
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

	res, err := h.NumberGetExistingNumbers(ctx, []string{"+19999999999", "+18888888888"})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res == nil {
		t.Errorf("Expected non-nil empty slice, got nil")
	}

	if len(res) != 0 {
		t.Errorf("Expected empty slice, got %d items", len(res))
	}
}
