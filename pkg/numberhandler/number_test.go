package numberhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

func TestCreateNumbersTelnyx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandler(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name       string
		customerID uuid.UUID
		numbers    []string

		expectRes []*number.Number
	}

	tests := []test{
		{
			"normal us",
			uuid.FromStringOrNil("ed1601ee-7ff3-11ec-926b-4352f35f23ab"),
			[]string{"+821021656521"},
			[]*number.Number{
				{
					ID:           uuid.FromStringOrNil("84cdc0a8-79d8-11eb-9179-ffc8c4fc9756"),
					Number:       "+821021656521",
					ProviderName: number.ProviderNameTelnyx,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTelnyx.EXPECT().CreateOrderNumbers(tt.customerID, tt.numbers).Return(tt.expectRes, nil)

			res, err := h.CreateNumbers(tt.customerID, tt.numbers)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestCreateOrderNumberTelnyx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandler(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name       string
		customerID uuid.UUID
		number     string

		expectRes *number.Number
	}

	tests := []test{
		{
			"normal us",
			uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			"+821021656521",
			&number.Number{
				ID:           uuid.FromStringOrNil("61afc712-7b25-11eb-b31f-5357d050c809"),
				Number:       "+821021656521",
				ProviderName: number.ProviderNameTelnyx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tmpNumbers := []string{tt.number}
			tmpExpRes := []*number.Number{tt.expectRes}
			mockTelnyx.EXPECT().CreateOrderNumbers(tt.customerID, tmpNumbers).Return(tmpExpRes, nil)

			res, err := h.CreateNumber(tt.customerID, tt.number)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestGetNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandler(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name     string
		numberID uuid.UUID
		number   *number.Number
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("b737aade-7a34-11eb-90bb-978a74aed8f6"),
			&number.Number{
				ID:                  uuid.FromStringOrNil("b737aade-7a34-11eb-90bb-978a74aed8f6"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().NumberGet(gomock.Any(), tt.numberID).Return(tt.number, nil)
			res, err := h.GetNumber(ctx, tt.numberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.number, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.number, res)
			}
		})
	}
}

func TestGetNumbers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandler(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name       string
		customerID uuid.UUID
		pageSize   uint64
		pageToken  string

		numbers []*number.Number
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
			10,
			"2021-02-26 18:26:49.000",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("da535752-7a4d-11eb-aec4-5bac74c24370"),
					Number:              "+821021656521",
					CustomerID:          uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
					TMPurchase:          "2021-02-26 18:26:49.000",
					TMCreate:            "2021-02-26 18:26:49.000",
				},
			},
		},
		{
			"empty token",
			uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
			10,
			"",
			[]*number.Number{
				{
					ID:                  uuid.FromStringOrNil("b72d1844-7bdd-11eb-a2bb-4370f115b44c"),
					Number:              "+821021656521",
					CustomerID:          uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
					TMPurchase:          "2021-02-26 18:26:49.000",
					TMCreate:            "2021-02-26 18:26:49.000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.pageToken == "" {
				mockDB.EXPECT().NumberGets(gomock.Any(), tt.customerID, tt.pageSize, gomock.Any()).Return(tt.numbers, nil)
			} else {
				mockDB.EXPECT().NumberGets(gomock.Any(), tt.customerID, tt.pageSize, tt.pageToken).Return(tt.numbers, nil)
			}

			res, err := h.GetNumbers(ctx, tt.customerID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.numbers, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.numbers, res)
			}
		})
	}
}

func TestUpdateNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandler(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		cache:            mockCache,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name         string
		updateNumber *number.Number
		number       *number.Number
	}

	tests := []test{
		{
			"normal",
			&number.Number{
				ID:     uuid.FromStringOrNil("1e5f4238-7c58-11eb-a6aa-fb7278bbb0bc"),
				FlowID: uuid.FromStringOrNil("1f71c61e-7c58-11eb-8d07-6f618f90475f"),
			},
			&number.Number{
				ID:                  uuid.FromStringOrNil("1e5f4238-7c58-11eb-a6aa-fb7278bbb0bc"),
				FlowID:              uuid.FromStringOrNil("1f71c61e-7c58-11eb-8d07-6f618f90475f"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
				TMCreate:            "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().NumberGet(gomock.Any(), tt.updateNumber.ID).Return(tt.number, nil)
			mockDB.EXPECT().NumberUpdate(gomock.Any(), tt.updateNumber).Return(nil)
			res, err := h.UpdateNumber(ctx, tt.updateNumber)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.number, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.number, res)
			}
		})
	}
}
