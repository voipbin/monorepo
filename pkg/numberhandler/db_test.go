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
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/util"
)

func Test_Create_OrderNumberTelnyx(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		number        string
		numberName    string
		detail        string

		responseUUID uuid.UUID

		expectRes *number.Number
	}{
		{
			"normal us",

			uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			uuid.FromStringOrNil("1b38eca6-a864-11ec-a2a1-6f2bb4ef8c7e"),
			uuid.FromStringOrNil("3ba45c68-8821-11ec-bc88-2367c938e4d5"),
			"+821021656521",
			"test name",
			"test detail",

			uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),

			&number.Number{
				ID:           uuid.FromStringOrNil("61afc712-7b25-11eb-b31f-5357d050c809"),
				Number:       "+821021656521",
				ProviderName: number.ProviderNameTelnyx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := util.NewMockUtil(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				util:                mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.customerID).Return(true, nil)

			mockTelnyx.EXPECT().CreateNumber(tt.customerID, tt.number, tt.callFlowID, tt.numberName, tt.detail).Return(tt.expectRes, nil)
			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().NumberCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishEvent(gomock.Any(), number.EventTypeNumberCreated, tt.expectRes)

			res, err := h.Create(ctx, tt.customerID, tt.number, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseNumber *number.Number

		expectRes *number.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("3ed894d6-7316-11ed-896c-2bd2d37eb485"),

			&number.Number{
				ID:           uuid.FromStringOrNil("3ed894d6-7316-11ed-896c-2bd2d37eb485"),
				ProviderName: number.ProviderNameTelnyx,
			},

			&number.Number{
				ID:           uuid.FromStringOrNil("3ed894d6-7316-11ed-896c-2bd2d37eb485"),
				ProviderName: number.ProviderNameTelnyx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := util.NewMockUtil(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				util:                mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}

			ctx := context.Background()

			mockDB.EXPECT().NumberGet(ctx, tt.id).Return(tt.responseNumber, nil)

			switch tt.responseNumber.ProviderName {
			case number.ProviderNameTelnyx:
				mockTelnyx.EXPECT().ReleaseNumber(ctx, tt.responseNumber).Return(nil)
			}

			mockDB.EXPECT().NumberDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, tt.id).Return(tt.responseNumber, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseNumber.CustomerID, number.EventTypeNumberDeleted, tt.responseNumber)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		numberID       uuid.UUID
		responseNumber *number.Number
	}{
		{
			"normal",
			uuid.FromStringOrNil("b737aade-7a34-11eb-90bb-978a74aed8f6"),
			&number.Number{
				ID: uuid.FromStringOrNil("b737aade-7a34-11eb-90bb-978a74aed8f6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				reqHandler:          mockReq,
				db:                  mockDB,
				numberHandlerTelnyx: mockTelnyx,
			}

			ctx := context.Background()

			mockDB.EXPECT().NumberGet(ctx, tt.numberID).Return(tt.responseNumber, nil)
			res, err := h.Get(ctx, tt.numberID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}

func Test_GetByNumber(t *testing.T) {

	tests := []struct {
		name string

		number         string
		responseNumber *number.Number
	}{
		{
			"normal",

			"+821100000001",
			&number.Number{
				ID: uuid.FromStringOrNil("e48134a4-7318-11ed-8617-0be7df22c985"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				reqHandler:          mockReq,
				db:                  mockDB,
				numberHandlerTelnyx: mockTelnyx,
			}

			ctx := context.Background()

			mockDB.EXPECT().NumberGetByNumber(ctx, tt.number).Return(tt.responseNumber, nil)
			res, err := h.GetByNumber(ctx, tt.number)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}

func Test_GetNumbers(t *testing.T) {

	type test struct {
		name string

		customerID uuid.UUID
		pageSize   uint64
		pageToken  string

		responseNumbers []*number.Number
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := util.NewMockUtil(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				util:                mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				numberHandlerTelnyx: mockTelnyx,
			}

			ctx := context.Background()

			if tt.pageToken == "" {
				mockUtil.EXPECT().GetCurTime().Return(util.GetCurTime())
				mockDB.EXPECT().NumberGets(gomock.Any(), tt.customerID, tt.pageSize, gomock.Any()).Return(tt.responseNumbers, nil)
			} else {
				mockDB.EXPECT().NumberGets(gomock.Any(), tt.customerID, tt.pageSize, tt.pageToken).Return(tt.responseNumbers, nil)
			}

			res, err := h.GetsByCustomerID(ctx, tt.customerID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumbers, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumbers, res)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		numberName string
		detail     string

		responseNumber *number.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("1f4ec1a4-8806-11ec-9012-7f3e92770a1f"),
			"update name",
			"update detail",

			&number.Number{
				ID:                  uuid.FromStringOrNil("1e5f4238-7c58-11eb-a6aa-fb7278bbb0bc"),
				CallFlowID:          uuid.FromStringOrNil("1f71c61e-7c58-11eb-8d07-6f618f90475f"),
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockDB.EXPECT().NumberGet(gomock.Any(), tt.id).Return(tt.responseNumber, nil).AnyTimes()
			mockDB.EXPECT().NumberUpdateBasicInfo(gomock.Any(), tt.id, tt.numberName, tt.detail).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseNumber.CustomerID, number.EventTypeNumberUpdated, tt.responseNumber)
			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}
