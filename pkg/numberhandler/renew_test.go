package numberhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
)

func Test_RenewNumbers_renewNumbersByTMRenew(t *testing.T) {

	type test struct {
		name string

		tmRenew string

		responseNumbers []*number.Number
	}

	tests := []test{
		{
			name: "normal",

			tmRenew: "2021-02-26 18:26:49.000",

			responseNumbers: []*number.Number{
				{
					ID: uuid.FromStringOrNil("8928588e-144f-11ee-b7b1-cf2766a4e52b"),
				},
				{
					ID: uuid.FromStringOrNil("8a163590-144f-11ee-914a-b7c9a2edb6d0"),
				},
				{
					ID: uuid.FromStringOrNil("8a3f8b5c-144f-11ee-8f6d-0ba403c732e1"),
				},
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

			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew, uint64(100), map[string]string{"deleted": "false"}).Return(tt.responseNumbers, nil)
			for _, n := range tt.responseNumbers {
				mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
				mockDB.EXPECT().NumberUpdateTMRenew(ctx, n.ID).Return(nil)
				mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)
				mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n)
			}

			res, err := h.RenewNumbers(ctx, 0, 0, tt.tmRenew)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseNumbers, res) {
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

			responseCurTime: "2021-02-26 18:26:49.000",
			responseNumbers: []*number.Number{
				{
					ID: uuid.FromStringOrNil("e51723ae-1e28-11ee-bdaa-83544a71ef96"),
				},
				{
					ID: uuid.FromStringOrNil("e54b6cfe-1e28-11ee-bb15-0f79849df0a6"),
				},
				{
					ID: uuid.FromStringOrNil("e571504a-1e28-11ee-b2c8-6fd277f0b78a"),
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
			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[string]string{"deleted": "false"}).Return(tt.responseNumbers, nil)
			for _, n := range tt.responseNumbers {
				mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
				mockDB.EXPECT().NumberUpdateTMRenew(ctx, n.ID).Return(nil)
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

			responseCurTime: "2021-02-26 18:26:49.000",
			responseNumbers: []*number.Number{
				{
					ID: uuid.FromStringOrNil("e51723ae-1e28-11ee-bdaa-83544a71ef96"),
				},
				{
					ID: uuid.FromStringOrNil("e54b6cfe-1e28-11ee-bb15-0f79849df0a6"),
				},
				{
					ID: uuid.FromStringOrNil("e571504a-1e28-11ee-b2c8-6fd277f0b78a"),
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
			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.responseCurTime, uint64(100), map[string]string{"deleted": "false"}).Return(tt.responseNumbers, nil)
			for _, n := range tt.responseNumbers {
				mockReq.EXPECT().CustomerV1CustomerIsValidBalance(ctx, n.CustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(true, nil)
				mockDB.EXPECT().NumberUpdateTMRenew(ctx, n.ID).Return(nil)
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
