package numberhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
)

func TestCreateOrderNumberTelnyx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		notifyHandler:    mockNotify,
		numHandlerTelnyx: mockTelnyx,
	}

	tests := []struct {
		name string

		customerID uuid.UUID
		flowID     uuid.UUID
		number     string
		numberName string
		detail     string

		expectRes *number.Number
	}{
		{
			"normal us",

			uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			uuid.FromStringOrNil("3ba45c68-8821-11ec-bc88-2367c938e4d5"),
			"+821021656521",
			"test name",
			"test detail",

			&number.Number{
				ID:           uuid.FromStringOrNil("61afc712-7b25-11eb-b31f-5357d050c809"),
				Number:       "+821021656521",
				ProviderName: number.ProviderNameTelnyx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockTelnyx.EXPECT().CreateNumber(tt.customerID, tt.number, tt.flowID, tt.numberName, tt.detail).Return(tt.expectRes, nil)
			mockDB.EXPECT().NumberCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), number.EventTypeNumberCreated, tt.expectRes)

			res, err := h.CreateNumber(ctx, tt.customerID, tt.number, tt.flowID, tt.numberName, tt.detail)
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
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
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
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
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

func TestUpdateBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

	h := numberHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		notifyHandler:    mockNotify,
		numHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name string

		id         uuid.UUID
		numberName string
		detail     string

		number *number.Number
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("1f4ec1a4-8806-11ec-9012-7f3e92770a1f"),
			"update name",
			"update detail",

			&number.Number{
				ID:                  uuid.FromStringOrNil("1e5f4238-7c58-11eb-a6aa-fb7278bbb0bc"),
				FlowID:              uuid.FromStringOrNil("1f71c61e-7c58-11eb-8d07-6f618f90475f"),
				Number:              "+821021656521",
				CustomerID:          uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
				Name:                "update name",
				Detail:              "update detail",
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

			mockDB.EXPECT().NumberGet(gomock.Any(), tt.id).Return(tt.number, nil).AnyTimes()
			mockDB.EXPECT().NumberUpdateBasicInfo(gomock.Any(), tt.id, tt.numberName, tt.detail).Return(nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), number.EventTypeNumberUpdated, tt.number)
			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.number, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.number, res)
			}
		})
	}
}
