package numberhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
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

		responseUUID   uuid.UUID
		responseTelnyx *providernumber.ProviderNumber
		responseNumber *number.Number

		expectNumber *number.Number
		expectTags   []string
	}{
		{
			name: "normal us",

			customerID:    uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			callFlowID:    uuid.FromStringOrNil("1b38eca6-a864-11ec-a2a1-6f2bb4ef8c7e"),
			messageFlowID: uuid.FromStringOrNil("3ba45c68-8821-11ec-bc88-2367c938e4d5"),
			number:        "+821021656521",
			numberName:    "test name",
			detail:        "test detail",

			responseUUID: uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
			responseTelnyx: &providernumber.ProviderNumber{
				ID:               "7dfbe2b4-1f4e-11ee-8502-23ddd1432a09",
				Status:           number.StatusActive,
				T38Enabled:       true,
				EmergencyEnabled: false,
			},
			responseNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
					CustomerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
				},
				Number:       "+821021656521",
				ProviderName: number.ProviderNameTelnyx,
			},

			expectNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
					CustomerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
				},
				Number:              "+821021656521",
				CallFlowID:          uuid.FromStringOrNil("1b38eca6-a864-11ec-a2a1-6f2bb4ef8c7e"),
				MessageFlowID:       uuid.FromStringOrNil("3ba45c68-8821-11ec-bc88-2367c938e4d5"),
				Name:                "test name",
				Detail:              "test detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "7dfbe2b4-1f4e-11ee-8502-23ddd1432a09",
				Status:              number.StatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
			},
			expectTags: []string{
				"CustomerID_f8509f38-7ff3-11ec-ac84-e3401d882a9f",
				"NumberID_96c97670-7315-11ed-8501-739535181602",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, tt.customerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)

			mockTelnyx.EXPECT().NumberPurchase(tt.number).Return(tt.responseTelnyx, nil)
			mockDB.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: tt.number}).Return([]*number.Number{}, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().NumberCreate(ctx, tt.expectNumber).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, gomock.Any()).Return(tt.responseNumber, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseNumber.CustomerID, number.EventTypeNumberCreated, tt.responseNumber)

			mockTelnyx.EXPECT().NumberUpdateTags(ctx, tt.responseNumber, tt.expectTags).Return(nil)

			res, err := h.Create(ctx, tt.customerID, tt.number, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}

func Test_Register(t *testing.T) {

	tests := []struct {
		name string

		customerID          uuid.UUID
		callFlowID          uuid.UUID
		messageFlowID       uuid.UUID
		number              string
		numberName          string
		detail              string
		providerName        number.ProviderName
		providerReferenceID string
		status              number.Status
		t38Enabled          bool
		emergencyEnabled    bool

		responseUUID uuid.UUID
		responseGets []*number.Number

		expectCreateNumber *number.Number
		responseNumber     *number.Number

		expectRes *number.Number
	}{
		{
			name: "normal",

			customerID:          uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			callFlowID:          uuid.FromStringOrNil("1b38eca6-a864-11ec-a2a1-6f2bb4ef8c7e"),
			messageFlowID:       uuid.FromStringOrNil("3ba45c68-8821-11ec-bc88-2367c938e4d5"),
			number:              "+821021656521",
			numberName:          "test name",
			detail:              "test detail",
			providerName:        number.ProviderNameNone,
			providerReferenceID: "",
			status:              number.StatusActive,
			t38Enabled:          false,
			emergencyEnabled:    false,

			responseUUID: uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
			responseGets: []*number.Number{},

			expectCreateNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
					CustomerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
				},
				Number:           "+821021656521",
				CallFlowID:       uuid.FromStringOrNil("1b38eca6-a864-11ec-a2a1-6f2bb4ef8c7e"),
				MessageFlowID:    uuid.FromStringOrNil("3ba45c68-8821-11ec-bc88-2367c938e4d5"),
				Name:             "test name",
				Detail:           "test detail",
				ProviderName:     number.ProviderNameNone,
				Status:           number.StatusActive,
				T38Enabled:       false,
				EmergencyEnabled: false,
			},
			responseNumber: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
					CustomerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
				},
				Number:       "+821021656521",
				ProviderName: number.ProviderNameNone,
			},
			expectRes: &number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("96c97670-7315-11ed-8501-739535181602"),
					CustomerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
				},
				Number:       "+821021656521",
				ProviderName: number.ProviderNameNone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := numberHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: tt.number}).Return(tt.responseGets, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().NumberCreate(ctx, tt.expectCreateNumber).Return(nil)
			mockDB.EXPECT().NumberGet(ctx, gomock.Any()).Return(tt.responseNumber, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseNumber.CustomerID, number.EventTypeNumberCreated, tt.responseNumber)

			res, err := h.Register(ctx, tt.customerID, tt.number, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail, tt.providerName, tt.providerReferenceID, tt.status, tt.t38Enabled, tt.emergencyEnabled)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_Register_error(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		callFlowID    uuid.UUID
		messageFlowID uuid.UUID
		number        string
		numberName    string
		detail        string

		mockGetsResult []*number.Number
		mockGetsErr    error
	}{
		{
			name: "already exists",

			customerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			number:     "+821021656521",

			mockGetsResult: []*number.Number{
				{
					Number: "+821021656521",
				},
			},
		},
		{
			name: "db error",

			customerID: uuid.FromStringOrNil("f8509f38-7ff3-11ec-ac84-e3401d882a9f"),
			number:     "+821021656521",

			mockGetsErr: fmt.Errorf("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := numberHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: tt.number}).Return(tt.mockGetsResult, tt.mockGetsErr)

			_, err := h.Register(ctx, tt.customerID, tt.number, tt.callFlowID, tt.messageFlowID, tt.numberName, tt.detail, number.ProviderNameNone, "", number.StatusActive, false, false)
			if err == nil {
				t.Errorf("Wrong match. expect error, got nil")
			}
		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[number.Field]any

		responseNumber *number.Number
	}{
		{
			"normal",

			uuid.FromStringOrNil("1f4ec1a4-8806-11ec-9012-7f3e92770a1f"),
			map[number.Field]any{
				number.FieldCallFlowID:    uuid.FromStringOrNil("a3b2e2be-20a2-11ee-9802-374336cc5af7"),
				number.FieldMessageFlowID: uuid.FromStringOrNil("a3ed47f6-20a2-11ee-baa5-0b63182446cf"),
				number.FieldName:          "update name",
				number.FieldDetail:        "update detail",
			},

			&number.Number{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1e5f4238-7c58-11eb-a6aa-fb7278bbb0bc"),
					CustomerID: uuid.FromStringOrNil("0598bd6a-7ff4-11ec-aba4-a7de6d96d9b3"),
				},
				CallFlowID:          uuid.FromStringOrNil("1f71c61e-7c58-11eb-8d07-6f618f90475f"),
				Number:              "+821021656521",
				Name:                "update name",
				Detail:              "update detail",
				ProviderName:        number.ProviderNameTelnyx,
				ProviderReferenceID: "1580568175064384684",
				Status:              number.StatusActive,
				T38Enabled:          false,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26T18:26:49.000Z",
				TMCreate:            "2021-02-26T18:26:49.000Z",
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

			mockDB.EXPECT().NumberUpdate(gomock.Any(), tt.id, tt.fields).Return(nil)
			mockDB.EXPECT().NumberGet(gomock.Any(), tt.id).Return(tt.responseNumber, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseNumber.CustomerID, number.EventTypeNumberUpdated, tt.responseNumber)
			res, err := h.Update(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumber, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumber, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	type test struct {
		name string

		pageSize  uint64
		pageToken string
		filters   map[number.Field]any

		responseNumbers []*number.Number
	}

	tests := []test{
		{
			"normal",

			10,
			"2021-02-26T18:26:49.000Z",
			map[number.Field]any{
				number.FieldCustomerID: uuid.FromStringOrNil("0b22cb36-eca8-11ee-a178-2f4c3561dcfd"),
				number.FieldDeleted:    false,
			},

			[]*number.Number{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("da535752-7a4d-11eb-aec4-5bac74c24370"),
						CustomerID: uuid.FromStringOrNil("0b22cb36-eca8-11ee-a178-2f4c3561dcfd"),
					},
					Number:              "+821021656521",
					ProviderName:        number.ProviderNameTelnyx,
					ProviderReferenceID: "1580568175064384684",
					Status:              number.StatusActive,
					T38Enabled:          false,
					EmergencyEnabled:    false,
					TMPurchase:          "2021-02-26T18:26:49.000Z",
					TMCreate:            "2021-02-26T18:26:49.000Z",
				},
			},
		},
		{
			"empty result",

			10,
			"2021-02-26T18:26:49.000Z",
			map[number.Field]any{
				number.FieldCustomerID: uuid.FromStringOrNil("17ea600e-eca8-11ee-b3c1-576ea96bdbfb"),
				number.FieldDeleted:    false,
			},

			[]*number.Number{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockDB.EXPECT().NumberList(ctx, tt.pageSize, tt.pageToken, tt.filters).Return(tt.responseNumbers, nil)

			res, err := h.List(ctx, tt.pageSize, tt.pageToken, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseNumbers, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseNumbers, res)
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b737aade-7a34-11eb-90bb-978a74aed8f6"),
				},
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3ed894d6-7316-11ed-896c-2bd2d37eb485"),
				},
				ProviderName: number.ProviderNameTelnyx,
			},

			&number.Number{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3ed894d6-7316-11ed-896c-2bd2d37eb485"),
				},
				ProviderName: number.ProviderNameTelnyx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}

			ctx := context.Background()

			mockDB.EXPECT().NumberGet(ctx, tt.id).Return(tt.responseNumber, nil)

			switch tt.responseNumber.ProviderName {
			case number.ProviderNameTelnyx:
				mockTelnyx.EXPECT().NumberRelease(ctx, tt.responseNumber).Return(nil)
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
