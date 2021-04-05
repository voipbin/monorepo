package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
)

func TestNumberCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		number       *number.Number
		expectNumber *number.Number
	}

	tests := []test{
		{
			"test normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
			&number.Number{
				ID:                  uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(context.Background(), tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectNumber, res)
			}
		})
	}
}

func TestNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	type test struct {
		name string

		userID      uint64
		expectCount int
		numbers     []*number.Number
	}

	tests := []test{
		{
			"normal",
			1,
			1,
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("10f04e98-95bd-11eb-a2c3-1ba7aeb1cd61"),
					UserID:     1,
					Number:     "+1234567890",
					TMPurchase: "2021-01-01 00:00:00.000",
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   defaultTimeStamp,
					TMDelete:   defaultTimeStamp,
				},
			},
		},
		{
			"empty",
			2,
			0,
			[]*number.Number{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// creates numbers for test
			for i := 0; i < len(tt.numbers); i++ {
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
				h.NumberCreate(context.Background(), tt.numbers[i])

			}

			res, err := h.NumberGets(context.Background(), tt.userID, 10, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectCount {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectCount, len(res))
			}
		})
	}
}

func TestNumberGetsByFlowID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	type test struct {
		name string

		flowID  uuid.UUID
		numbers []*number.Number

		expectNum int
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("66beabfe-7d20-11eb-9b69-375c485b40fa"),
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("5d73b940-7d20-11eb-8335-97856a00f2c6"),
					UserID:     1,
					FlowID:     uuid.FromStringOrNil("66beabfe-7d20-11eb-9b69-375c485b40fa"),
					TMPurchase: "2021-01-01 00:00:00.000",
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   defaultTimeStamp,
					TMDelete:   defaultTimeStamp,
				},
			},
			1,
		},
		{
			"3 flows, but grep 2",
			uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("109347b6-7d21-11eb-bdd4-c7226a0e1c81"),
					UserID:     1,
					FlowID:     uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
					TMPurchase: "2021-01-01 00:00:00.000",
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   defaultTimeStamp,
					TMDelete:   defaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("10b60706-7d21-11eb-90ae-2305526adf47"),
					UserID:     1,
					FlowID:     uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
					TMPurchase: "2021-01-01 00:00:00.000",
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   defaultTimeStamp,
					TMDelete:   defaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("10cf5ee0-7d21-11eb-9733-b73b63288625"),
					UserID:     1,
					FlowID:     uuid.FromStringOrNil("10eff100-7d21-11eb-b275-6ff5cde65beb"),
					TMPurchase: "2021-01-01 00:00:00.000",
					TMCreate:   "2021-01-01 00:00:00.000",
					TMUpdate:   defaultTimeStamp,
					TMDelete:   defaultTimeStamp,
				},
			},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// create numbers
			for _, n := range tt.numbers {
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
				h.NumberCreate(ctx, n)
			}

			res, err := h.NumberGetsByFlowID(ctx, tt.flowID, 100, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectNum {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectNum, len(res))
			}
		})
	}
}

func TestNumberDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		number       *number.Number
		expectNumber *number.Number
	}

	tests := []test{
		{
			"test normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},

			&number.Number{
				ID:                  uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusDeleted,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberDelete(ctx, tt.number.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == "" || res.TMUpdate == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
			res.TMCreate = ""
			res.TMDelete = ""
			res.TMUpdate = ""

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func TestNumberUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		number       *number.Number
		updateNumber *number.Number
		expectNumber *number.Number
	}

	tests := []test{
		{
			"test normal",
			&number.Number{
				ID:                  uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
			&number.Number{
				ID:     uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				FlowID: uuid.FromStringOrNil("9496e31a-7c54-11eb-915d-3f8ab244a929"),
			},
			&number.Number{
				ID:                  uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				Number:              "+821021656521",
				FlowID:              uuid.FromStringOrNil("9496e31a-7c54-11eb-915d-3f8ab244a929"),
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberUpdate(ctx, tt.updateNumber); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMUpdate == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
			res.TMCreate = ""
			res.TMDelete = ""
			res.TMUpdate = ""

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func TestNumberGetFromDBByNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	type test struct {
		name         string
		num          string
		numbers      []*number.Number
		expectNumber *number.Number
	}

	tests := []test{
		{
			"test normal",
			"+821021656521",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("41401778-95c6-11eb-ba94-3f9e9f4fcab2"),
					Number:              "+821021656521",
					UserID:              1,
					ProviderName:        "telnyx",
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          true,
					EmergencyEnabled:    false,
					TMPurchase:          "2021-02-26 18:26:49.000",
					TMCreate:            "2021-02-26 18:26:49.000",
					TMUpdate:            defaultTimeStamp,
					TMDelete:            defaultTimeStamp,
				},
			},
			&number.Number{
				ID:                  uuid.FromStringOrNil("41401778-95c6-11eb-ba94-3f9e9f4fcab2"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            defaultTimeStamp,
				TMDelete:            defaultTimeStamp,
			},
		},
		{
			"deleted number",
			"+821021656522",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("a97fca22-95c6-11eb-bac1-1bda92edcfd9"),
					Number:              "+821021656522",
					UserID:              1,
					ProviderName:        "telnyx",
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          true,
					EmergencyEnabled:    false,
					TMPurchase:          "2021-02-26 18:26:49.000",
					TMCreate:            "2021-02-26 18:26:49.000",
					TMUpdate:            defaultTimeStamp,
					TMDelete:            "2021-02-26 18:26:49.000",
				},
				{
					ID:                  uuid.FromStringOrNil("0d590ee6-95c7-11eb-a038-db90335f3a7d"),
					Number:              "+821021656522",
					UserID:              1,
					ProviderName:        "telnyx",
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          true,
					EmergencyEnabled:    false,
					TMPurchase:          "2021-02-26 18:26:49.000",
					TMCreate:            "2021-02-26 18:26:49.000",
					TMUpdate:            defaultTimeStamp,
					TMDelete:            defaultTimeStamp,
				}},
			&number.Number{
				ID:                  uuid.FromStringOrNil("0d590ee6-95c7-11eb-a038-db90335f3a7d"),
				Number:              "+821021656522",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            defaultTimeStamp,
				TMDelete:            defaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, num := range tt.numbers {
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
				if err := h.NumberCreate(ctx, num); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.NumberGetFromDBByNumber(ctx, tt.num)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectNumber) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectNumber, res)
			}
		})
	}
}
