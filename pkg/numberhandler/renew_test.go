package numberhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
)

func Test_RenewNumbers(t *testing.T) {

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

			mockDB.EXPECT().NumberGetsByTMRenew(ctx, tt.tmRenew).Return(tt.responseNumbers, nil)
			for _, n := range tt.responseNumbers {
				mockDB.EXPECT().NumberUpdateTMRenew(ctx, n.ID).Return(nil)
				mockDB.EXPECT().NumberGet(ctx, n.ID).Return(n, nil)
				mockNotify.EXPECT().PublishEvent(ctx, number.EventTypeNumberRenewed, n)
			}

			res, err := h.RenewNumbers(ctx, tt.tmRenew)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseNumbers, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseNumbers, res)
			}
		})
	}
}
