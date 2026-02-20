package numberhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
)

var testCurTime = time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

func Test_RenewNumbers_renewNumbersByTMRenew(t *testing.T) {

	type test struct {
		name string

		tmRenew string

		responseNumbers []*number.Number
	}

	tests := []test{
		{
			name: "normal",

			tmRenew: "2021-02-26T18:26:49.000Z",

			responseNumbers: []*number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8928588e-144f-11ee-b7b1-cf2766a4e52b"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8a163590-144f-11ee-914a-b7c9a2edb6d0"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8a3f8b5c-144f-11ee-8f6d-0ba403c732e1"),
					},
				},
			},
		},
		{
			name:            "empty result",
			tmRenew:         "2021-02-26T18:26:49.000Z",
			responseNumbers: []*number.Number{},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}

			ctx := context.Background()

			if len(tt.responseNumbers) > 0 {
				gomock.InOrder(
					mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return(tt.responseNumbers, nil),
					mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil),
				)
			} else {
				mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil)
			}
			for _, n := range tt.responseNumbers {
				mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
				mockUtil.EXPECT().TimeNow().Return(&testCurTime)
				mockDB.EXPECT().NumberUpdate(ctx, n.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)
				mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n)
			}

			res, err := h.RenewNumbers(ctx, 0, 0, tt.tmRenew)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(tt.responseNumbers) == 0 {
				if !reflect.DeepEqual([]*number.Number{}, res) {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", []*number.Number{}, res)
				}
			} else if !reflect.DeepEqual(tt.responseNumbers, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseNumbers, res)
			}
		})
	}
}

func Test_RenewNumbers_renewNumbersByDays(t *testing.T) {

	type test struct {
		name string

		days int

		responseCurTime string
		responseNumbers []*number.Number

		expectTimeAdd time.Duration
	}

	tests := []test{
		{
			name: "normal",

			days: 3,

			responseCurTime: "2021-02-26T18:26:49.000Z",
			responseNumbers: []*number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e51723ae-1e28-11ee-bdaa-83544a71ef96"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e54b6cfe-1e28-11ee-bb15-0f79849df0a6"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e571504a-1e28-11ee-b2c8-6fd277f0b78a"),
					},
				},
			},

			expectTimeAdd: -(time.Hour * 24 * 3),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTimeAdd(tt.expectTimeAdd).Return(tt.responseCurTime)
			gomock.InOrder(
				mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return(tt.responseNumbers, nil),
				mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil),
			)
			for _, n := range tt.responseNumbers {
				mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
				mockUtil.EXPECT().TimeNow().Return(&testCurTime)
				mockDB.EXPECT().NumberUpdate(ctx, n.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)
				mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n)
			}

			res, err := h.RenewNumbers(ctx, tt.days, 0, "")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseNumbers, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseNumbers, res)
			}
		})
	}
}

func Test_RenewNumbers_renewNumbersByHours(t *testing.T) {

	type test struct {
		name string

		hours int

		responseCurTime string
		responseNumbers []*number.Number

		expectTimeAdd time.Duration
	}

	tests := []test{
		{
			name: "normal",

			hours: 10,

			responseCurTime: "2021-02-26T18:26:49.000Z",
			responseNumbers: []*number.Number{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e51723ae-1e28-11ee-bdaa-83544a71ef96"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e54b6cfe-1e28-11ee-bb15-0f79849df0a6"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e571504a-1e28-11ee-b2c8-6fd277f0b78a"),
					},
				},
			},

			expectTimeAdd: -(time.Hour * 10),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := numberHandler{
				utilHandler:         mockUtil,
				reqHandler:          mockReq,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				numberHandlerTelnyx: mockTelnyx,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTimeAdd(tt.expectTimeAdd).Return(tt.responseCurTime)
			gomock.InOrder(
				mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return(tt.responseNumbers, nil),
				mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil),
			)
			for _, n := range tt.responseNumbers {
				mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
				mockUtil.EXPECT().TimeNow().Return(&testCurTime)
				mockDB.EXPECT().NumberUpdate(ctx, n.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)
				mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n)
			}

			res, err := h.RenewNumbers(ctx, 0, tt.hours, "")
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseNumbers, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseNumbers, res)
			}
		})
	}
}

func Test_RenewNumbers_renewNumbersByTMRenew_insufficientBalance(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaa00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}
	deletedN := &number.Number{
		Identity: commonidentity.Identity{
			ID:         n.ID,
			CustomerID: n.CustomerID,
		},
		Status: number.StatusDeleted,
	}

	// First query returns one number
	firstQuery := mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n}, nil)

	// Balance check returns invalid
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil)

	// Delete chain: Get -> provider release (ProviderNameNone = no-op) -> dbDelete -> Get -> PublishWebhookEvent
	getForDelete := mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)
	mockDB.EXPECT().NumberDelete(ctx, n.ID).Return(nil)
	mockDB.EXPECT().NumberGet(ctx, n.ID).Return(deletedN, nil).After(getForDelete)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, n.CustomerID, number.EventTypeNumberDeleted, deletedN)

	// Pagination: second query returns empty
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil).After(firstQuery)

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expected := []*number.Number{}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Expected empty result, got: %v", res)
	}
}

func Test_RenewNumbers_renewNumbersByTMRenew_balanceCheckError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("bbb00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}

	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n}, nil)

	// Balance check returns error — number skipped, processed stays 0, loop breaks
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, fmt.Errorf("billing service unavailable"))

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expected := []*number.Number{}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Expected empty result, got: %v", res)
	}
}

func Test_RenewNumbers_renewNumbersByTMRenew_dbUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("ddd00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}

	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n}, nil)

	// Balance is valid
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
	mockUtil.EXPECT().TimeNow().Return(&testCurTime)

	// DB update fails — processed stays 0, loop breaks
	mockDB.EXPECT().NumberUpdate(ctx, n.ID, gomock.Any()).Return(fmt.Errorf("database error"))

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expected := []*number.Number{}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Expected empty result, got: %v", res)
	}
}

func Test_RenewNumbers_renewNumbersByTMRenew_mixed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := numberHandler{
		utilHandler:         mockUtil,
		reqHandler:          mockReq,
		db:                  mockDB,
		notifyHandler:       mockNotify,
		numberHandlerTelnyx: mockTelnyx,
	}

	ctx := context.Background()
	tmRenew := "2021-02-26T18:26:49.000Z"

	n1 := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eee00000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000001"),
		},
	}
	n2 := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eee00000-0000-0000-0000-000000000002"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000002"),
		},
	}
	n2Deleted := &number.Number{
		Identity: commonidentity.Identity{
			ID:         n2.ID,
			CustomerID: n2.CustomerID,
		},
		Status: number.StatusDeleted,
	}
	n3 := &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("eee00000-0000-0000-0000-000000000003"),
			CustomerID: uuid.FromStringOrNil("ccc00000-0000-0000-0000-000000000003"),
		},
	}

	// First query returns 3 numbers
	firstQuery := mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{n1, n2, n3}, nil)

	// n1: valid balance -> renewed
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n1.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
	mockUtil.EXPECT().TimeNow().Return(&testCurTime)
	mockDB.EXPECT().NumberUpdate(ctx, n1.ID, gomock.Any()).Return(nil)
	mockDB.EXPECT().NumberGet(ctx, n1.ID).Return(n1, nil)
	mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n1)

	// n2: insufficient balance -> deleted
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n2.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil)
	getN2ForDelete := mockDB.EXPECT().NumberGet(ctx, n2.ID).Return(n2, nil)
	mockDB.EXPECT().NumberDelete(ctx, n2.ID).Return(nil)
	mockDB.EXPECT().NumberGet(ctx, n2.ID).Return(n2Deleted, nil).After(getN2ForDelete)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, n2.CustomerID, number.EventTypeNumberDeleted, n2Deleted)

	// n3: valid balance -> renewed
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, n3.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
	mockUtil.EXPECT().TimeNow().Return(&testCurTime)
	mockDB.EXPECT().NumberUpdate(ctx, n3.ID, gomock.Any()).Return(nil)
	mockDB.EXPECT().NumberGet(ctx, n3.ID).Return(n3, nil)
	mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n3)

	// Pagination: second query returns empty
	mockDB.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), map[number.Field]any{number.FieldDeleted: false}).Return([]*number.Number{}, nil).After(firstQuery)

	res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expected := []*number.Number{n1, n3}
	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Wrong result.\nexpect: %v\ngot: %v", expected, res)
	}
}
