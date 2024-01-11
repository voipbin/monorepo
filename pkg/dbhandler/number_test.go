package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
)

func Test_NumberCreate(t *testing.T) {

	type test struct {
		name   string
		number *number.Number

		responseCurTime string
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"test normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("31a1ca10-7ff3-11ec-80f5-83db3c8e951b"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			"2021-02-26 18:26:49.000",
			&number.Number{
				ID:                  uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("31a1ca10-7ff3-11ec-80f5-83db3c8e951b"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            DefaultTimeStamp,
				TMDelete:            DefaultTimeStamp,
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

func Test_NumberGets(t *testing.T) {

	type test struct {
		name    string
		numbers []*number.Number

		customerID uuid.UUID
		filters    map[string]string

		responseCurTime string

		expectRes []*number.Number
	}

	tests := []test{
		{
			"normal",
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("10f04e98-95bd-11eb-a2c3-1ba7aeb1cd61"),
					CustomerID: uuid.FromStringOrNil("3b0bfcce-7ff3-11ec-b5cd-f3669cd35916"),
					Number:     "+1234567890",
				},
			},

			uuid.FromStringOrNil("3b0bfcce-7ff3-11ec-b5cd-f3669cd35916"),
			map[string]string{
				"deleted": "false",
			},

			"2021-01-01 00:00:00.000",

			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("10f04e98-95bd-11eb-a2c3-1ba7aeb1cd61"),
					CustomerID: uuid.FromStringOrNil("3b0bfcce-7ff3-11ec-b5cd-f3669cd35916"),
					Number:     "+1234567890",
					TMPurchase: "2021-01-01 00:00:00.000",
					TMRenew:    "2021-01-01 00:00:00.000",
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*number.Number{},

			uuid.FromStringOrNil("4c1150be-7ff3-11ec-adb5-771b9c899a73"),
			map[string]string{
				"deleted": "false",
			},

			"2021-01-01 00:00:00.000",
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
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				_ = h.NumberCreate(context.Background(), tt.numbers[i])
			}

			res, err := h.NumberGets(context.Background(), tt.customerID, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberGetsByCallFlowID(t *testing.T) {

	type test struct {
		name    string
		numbers []*number.Number

		flowID  uuid.UUID
		filters map[string]string

		responseCurTime string

		expectRes int
	}

	tests := []test{
		{
			"call flow id",
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("5d73b940-7d20-11eb-8335-97856a00f2c6"),
					CustomerID: uuid.FromStringOrNil("57115e32-7ff3-11ec-850e-afd53272231d"),
					CallFlowID: uuid.FromStringOrNil("66beabfe-7d20-11eb-9b69-375c485b40fa"),
					TMPurchase: "2021-01-01 00:00:00.000",
				},
			},

			uuid.FromStringOrNil("66beabfe-7d20-11eb-9b69-375c485b40fa"),
			map[string]string{},

			"2021-01-01 00:00:00.000",
			1,
		},
		{
			"3 flows, but grep 2",
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("109347b6-7d21-11eb-bdd4-c7226a0e1c81"),
					CustomerID: uuid.FromStringOrNil("5c6ea31c-7ff3-11ec-a028-b345c3f8ab55"),
					CallFlowID: uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
					TMPurchase: "2021-01-01 00:00:00.000",
				},
				{
					ID:         uuid.FromStringOrNil("10b60706-7d21-11eb-90ae-2305526adf47"),
					CustomerID: uuid.FromStringOrNil("5c6ea31c-7ff3-11ec-a028-b345c3f8ab55"),
					CallFlowID: uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
					TMPurchase: "2021-01-01 00:00:00.000",
				},
				{
					ID:         uuid.FromStringOrNil("10cf5ee0-7d21-11eb-9733-b73b63288625"),
					CustomerID: uuid.FromStringOrNil("5c6ea31c-7ff3-11ec-a028-b345c3f8ab55"),
					CallFlowID: uuid.FromStringOrNil("10eff100-7d21-11eb-b275-6ff5cde65beb"),
					TMPurchase: "2021-01-01 00:00:00.000",
				},
			},

			uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
			map[string]string{},

			"2021-01-01 00:00:00.000",
			2,
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

			// create numbers
			for _, n := range tt.numbers {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				_ = h.NumberCreate(ctx, n)
			}

			res, err := h.NumberGetsByCallFlowID(ctx, tt.flowID, 100, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectRes {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectRes, len(res))
			}
		})
	}
}

func Test_NumberGetsByMessageFlowID(t *testing.T) {

	type test struct {
		name    string
		numbers []*number.Number

		flowID  uuid.UUID
		filters map[string]string

		expectNum int
	}

	tests := []test{
		{
			"normal",
			[]*number.Number{
				{
					ID:            uuid.FromStringOrNil("974bb36e-a85f-11ec-afd8-13a6af6c6b79"),
					CustomerID:    uuid.FromStringOrNil("9775e2f6-a85f-11ec-a63e-ff69aed73a7f"),
					MessageFlowID: uuid.FromStringOrNil("9714e212-a85f-11ec-a571-d79e5520a61c"),
					TMPurchase:    "2021-01-01 00:00:00.000",
					TMCreate:      "2021-01-01 00:00:00.000",
					TMUpdate:      DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("9714e212-a85f-11ec-a571-d79e5520a61c"),
			map[string]string{},
			1,
		},
		{
			"3 flows, but grep 2",
			[]*number.Number{
				{
					ID:            uuid.FromStringOrNil("c5b4a4b6-a861-11ec-84c2-4fedd4b408ea"),
					CustomerID:    uuid.FromStringOrNil("5c6ea31c-7ff3-11ec-a028-b345c3f8ab55"),
					MessageFlowID: uuid.FromStringOrNil("97a1d14a-a85f-11ec-bd41-53acbe702228"),
					TMPurchase:    "2021-01-01 00:00:00.000",
					TMCreate:      "2021-01-01 00:00:00.000",
					TMUpdate:      DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
				{
					ID:            uuid.FromStringOrNil("c5e7c274-a861-11ec-b567-9bf433831b7f"),
					CustomerID:    uuid.FromStringOrNil("5c6ea31c-7ff3-11ec-a028-b345c3f8ab55"),
					MessageFlowID: uuid.FromStringOrNil("97a1d14a-a85f-11ec-bd41-53acbe702228"),
					TMPurchase:    "2021-01-01 00:00:00.000",
					TMCreate:      "2021-01-01 00:00:00.000",
					TMUpdate:      DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
				{
					ID:            uuid.FromStringOrNil("c611edce-a861-11ec-9dd3-abcaf22fe95a"),
					CustomerID:    uuid.FromStringOrNil("5c6ea31c-7ff3-11ec-a028-b345c3f8ab55"),
					MessageFlowID: uuid.FromStringOrNil("97d0b186-a85f-11ec-8a8a-cb8bcfc342ba"),
					TMPurchase:    "2021-01-01 00:00:00.000",
					TMCreate:      "2021-01-01 00:00:00.000",
					TMUpdate:      DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("97a1d14a-a85f-11ec-bd41-53acbe702228"),
			map[string]string{},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			// create numbers
			for _, n := range tt.numbers {
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				_ = h.NumberCreate(ctx, n)
			}

			res, err := h.NumberGetsByMessageFlowID(ctx, tt.flowID, 100, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectNum {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectNum, len(res))
			}
		})
	}
}

func Test_NumberDelete(t *testing.T) {

	type test struct {
		name            string
		number          *number.Number
		responseCurTime string
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("6884d8f6-7ff3-11ec-8b5c-d3aa777ad672"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			"2021-02-26 18:26:49.000",
			&number.Number{
				ID:                  uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("6884d8f6-7ff3-11ec-8b5c-d3aa777ad672"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusDeleted,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            "2021-02-26 18:26:49.000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()

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

func Test_NumberUpdateInfo(t *testing.T) {

	type test struct {
		name   string
		number *number.Number

		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		numberName    string
		detail        string

		responseCurTime string
		expectNumber    *number.Number
	}

	tests := []test{
		{
			name: "normal",
			number: &number.Number{
				ID:                  uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				CustomerID:          uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
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

			callFlowID:    uuid.FromStringOrNil("23ff7078-20a2-11ee-934c-8371c4d02f71"),
			messageFlowID: uuid.FromStringOrNil("24477634-20a2-11ee-b62f-4f341e77043b"),
			numberName:    "update name",
			detail:        "update detail",

			responseCurTime: "2021-02-26 18:26:49.000",
			expectNumber: &number.Number{
				ID:                  uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				CustomerID:          uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
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
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdateInfo(ctx, tt.number.ID, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

func Test_NumberUpdateFlowID(t *testing.T) {

	type test struct {
		name   string
		number *number.Number

		callFlowID    uuid.UUID
		messageFlowID uuid.UUID

		responseCurTime string
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("4a7c7d2a-a85f-11ec-8d15-9730036800e5"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("4aa698da-a85f-11ec-a93a-5fbf7b8302db"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			uuid.FromStringOrNil("4acedf84-a85f-11ec-bdc6-27902c5c6987"),
			uuid.FromStringOrNil("4af49f4e-a85f-11ec-ad06-676681d45adb"),

			"2021-02-26 18:26:49.000",
			&number.Number{
				ID:                  uuid.FromStringOrNil("4a7c7d2a-a85f-11ec-8d15-9730036800e5"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("4aa698da-a85f-11ec-a93a-5fbf7b8302db"),
				CallFlowID:          uuid.FromStringOrNil("4acedf84-a85f-11ec-bdc6-27902c5c6987"),
				MessageFlowID:       uuid.FromStringOrNil("4af49f4e-a85f-11ec-ad06-676681d45adb"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdateFlowID(ctx, tt.number.ID, tt.callFlowID, tt.messageFlowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

func Test_NumberUpdateCallFlowID(t *testing.T) {

	type test struct {
		name   string
		number *number.Number

		callFlowID uuid.UUID

		responseCurTime string
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"update callflow",
			&number.Number{
				ID:                  uuid.FromStringOrNil("37357400-8817-11ec-9616-0f38be341833"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			uuid.FromStringOrNil("535c0ca4-8801-11ec-accb-7bd692b1c078"),

			"2021-02-26 18:26:49.000",
			&number.Number{
				ID:                  uuid.FromStringOrNil("37357400-8817-11ec-9616-0f38be341833"),
				Number:              "+821021656521",
				CallFlowID:          uuid.FromStringOrNil("535c0ca4-8801-11ec-accb-7bd692b1c078"),
				MessageFlowID:       uuid.Nil,
				CustomerID:          uuid.FromStringOrNil("78da4358-7ff3-11ec-b15a-2754681def5e"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdateCallFlowID(ctx, tt.number.ID, tt.callFlowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

func Test_NumberUpdateMessageFlowID(t *testing.T) {

	type test struct {
		name   string
		number *number.Number

		flowID uuid.UUID

		responseCurTime string
		expectNumber    *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("b37a0bae-a85e-11ec-b666-7fce3ed0d0d5"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("b3e74570-a85e-11ec-a53b-331bdfd1d2f3"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},

			uuid.FromStringOrNil("b416b062-a85e-11ec-a230-7f3aae198503"),

			"2021-02-26 18:26:49.000",
			&number.Number{
				ID:                  uuid.FromStringOrNil("b37a0bae-a85e-11ec-b666-7fce3ed0d0d5"),
				Number:              "+821021656521",
				MessageFlowID:       uuid.FromStringOrNil("b416b062-a85e-11ec-a230-7f3aae198503"),
				CustomerID:          uuid.FromStringOrNil("b3e74570-a85e-11ec-a53b-331bdfd1d2f3"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime).AnyTimes()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any()).AnyTimes()
			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.NumberUpdateMessageFlowID(ctx, tt.number.ID, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

func Test_NumberGetFromDBByNumber(t *testing.T) {

	type test struct {
		name    string
		num     string
		numbers []*number.Number

		responseCurTime string
		expectRes       *number.Number
	}

	tests := []test{
		{
			"test normal",
			"+821100000010",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("41401778-95c6-11eb-ba94-3f9e9f4fcab2"),
					Number:              "+821100000010",
					CustomerID:          uuid.FromStringOrNil("82914798-7ff3-11ec-b5d5-1fc07ae57c63"),
					ProviderName:        "telnyx",
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          true,
					EmergencyEnabled:    false,
				},
			},

			"2021-02-26 18:26:49.000",
			&number.Number{
				ID:                  uuid.FromStringOrNil("41401778-95c6-11eb-ba94-3f9e9f4fcab2"),
				Number:              "+821100000010",
				CustomerID:          uuid.FromStringOrNil("82914798-7ff3-11ec-b5d5-1fc07ae57c63"),
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMRenew:             "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            DefaultTimeStamp,
				TMDelete:            DefaultTimeStamp,
			},
		}}

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

			for _, num := range tt.numbers {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				if err := h.NumberCreate(ctx, num); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.numberGetFromDBByNumber(ctx, tt.num)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberSetTMRenew(t *testing.T) {

	type test struct {
		name   string
		number *number.Number

		id uuid.UUID

		responseCurTime      string
		responseCurTimeRenew string
		expectRes            *number.Number
	}

	tests := []test{
		{
			name: "normal",
			number: &number.Number{
				ID: uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),
			},

			id: uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),

			responseCurTime:      "2021-02-26 18:26:49.000",
			responseCurTimeRenew: "2021-02-27 18:26:49.000",
			expectRes: &number.Number{
				ID:         uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),
				TMPurchase: "2021-02-26 18:26:49.000",
				TMRenew:    "2021-02-27 18:26:49.000",
				TMCreate:   "2021-02-26 18:26:49.000",
				TMUpdate:   "2021-02-27 18:26:49.000",
				TMDelete:   DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTimeRenew)
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			if err := h.NumberUpdateTMRenew(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_NumberGetsByTMRenew(t *testing.T) {

	type test struct {
		name    string
		numbers []number.Number

		id      uuid.UUID
		tmRenew string
		size    uint64
		filters map[string]string

		responseCurTimes []string
		expectRes        []*number.Number
	}

	tests := []test{
		{
			name: "normal",
			numbers: []number.Number{
				{
					ID: uuid.FromStringOrNil("9356093a-144c-11ee-b0ca-fbaf4f96747c"),
				},
				{
					ID: uuid.FromStringOrNil("93884aa8-144c-11ee-a261-eb324d4a94ab"),
				},
				{
					ID: uuid.FromStringOrNil("93b536da-144c-11ee-8e04-5f11847ed981"),
				},
				{
					ID: uuid.FromStringOrNil("93ec6588-144c-11ee-ae23-cb2a64c0f80a"),
				},
			},

			id:      uuid.FromStringOrNil("51535516-144b-11ee-8f01-3f32d4b89553"),
			tmRenew: "2020-04-12 20:26:49.000",
			size:    100,
			filters: map[string]string{
				"deleted": "false",
			},

			responseCurTimes: []string{
				"2020-04-10 18:26:49.000",
				"2020-04-11 18:26:49.000",
				"2020-04-12 18:26:49.000",
				"2020-04-13 18:26:49.000",
			},
			expectRes: []*number.Number{
				{
					ID:         uuid.FromStringOrNil("93b536da-144c-11ee-8e04-5f11847ed981"),
					TMPurchase: "2020-04-12 18:26:49.000",
					TMRenew:    "2020-04-12 18:26:49.000",
					TMCreate:   "2020-04-12 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("93884aa8-144c-11ee-a261-eb324d4a94ab"),
					TMPurchase: "2020-04-11 18:26:49.000",
					TMRenew:    "2020-04-11 18:26:49.000",
					TMCreate:   "2020-04-11 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("9356093a-144c-11ee-b0ca-fbaf4f96747c"),
					TMPurchase: "2020-04-10 18:26:49.000",
					TMRenew:    "2020-04-10 18:26:49.000",
					TMCreate:   "2020-04-10 18:26:49.000",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
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
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTimes[i])
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
