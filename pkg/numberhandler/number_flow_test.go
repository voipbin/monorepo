package numberhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
)

func Test_RemoveNumbersFlowID(t *testing.T) {

	type test struct {
		name               string
		numbersCallFlow    []*number.Number
		numbersMessageFlow []*number.Number
		flowID             uuid.UUID
	}

	tests := []test{
		{
			"normal call flow id",
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("e9e983b2-7d22-11eb-acd3-13c2efec905d"),
					CallFlowID: uuid.FromStringOrNil("dd92f3fa-7d22-11eb-be53-47ee94a9bce3"),
				},
			},
			[]*number.Number{},

			uuid.FromStringOrNil("dd92f3fa-7d22-11eb-be53-47ee94a9bce3"),
		},
		{
			"3 items call flow id",
			[]*number.Number{
				{
					ID:         uuid.FromStringOrNil("094aa406-7d24-11eb-81d5-2f5e99ab6fc1"),
					CallFlowID: uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
				},
				{
					ID:         uuid.FromStringOrNil("0993e8dc-7d24-11eb-8bee-dbca074d9894"),
					CallFlowID: uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
				},
				{
					ID:         uuid.FromStringOrNil("09ada2cc-7d24-11eb-8518-97f716018857"),
					CallFlowID: uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
				},
			},
			[]*number.Number{},

			uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
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

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().NumberGetsByCallFlowID(gomock.Any(), tt.flowID, gomock.Any(), gomock.Any()).Return(tt.numbersCallFlow, nil)
			for _, num := range tt.numbersCallFlow {
				mockDB.EXPECT().NumberUpdateCallFlowID(gomock.Any(), num.ID, uuid.Nil)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().NumberGetsByMessageFlowID(gomock.Any(), tt.flowID, gomock.Any(), gomock.Any()).Return(tt.numbersMessageFlow, nil)
			for _, num := range tt.numbersMessageFlow {
				mockDB.EXPECT().NumberUpdateMessageFlowID(gomock.Any(), num.ID, uuid.Nil)
			}

			if err := h.RemoveNumbersFlowID(ctx, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
