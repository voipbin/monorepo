package numberhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

func TestRemoveNumbersFlowID(t *testing.T) {
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
		name    string
		numbers []*number.Number
		flowID  uuid.UUID
	}

	tests := []test{
		{
			"normal",
			[]*number.Number{
				{
					ID:     uuid.FromStringOrNil("e9e983b2-7d22-11eb-acd3-13c2efec905d"),
					FlowID: uuid.FromStringOrNil("dd92f3fa-7d22-11eb-be53-47ee94a9bce3"),
				},
			},
			uuid.FromStringOrNil("dd92f3fa-7d22-11eb-be53-47ee94a9bce3"),
		},
		{
			"3 items",
			[]*number.Number{
				{
					ID:     uuid.FromStringOrNil("094aa406-7d24-11eb-81d5-2f5e99ab6fc1"),
					FlowID: uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
				},
				{
					ID:     uuid.FromStringOrNil("0993e8dc-7d24-11eb-8bee-dbca074d9894"),
					FlowID: uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
				},
				{
					ID:     uuid.FromStringOrNil("09ada2cc-7d24-11eb-8518-97f716018857"),
					FlowID: uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
				},
			},
			uuid.FromStringOrNil("0974bd22-7d24-11eb-8517-8f90f5f6be56"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().NumberGetsByFlowID(gomock.Any(), tt.flowID, gomock.Any(), gomock.Any()).Return(tt.numbers, nil)
			for _, num := range tt.numbers {
				num.FlowID = uuid.Nil
				mockDB.EXPECT().NumberUpdate(gomock.Any(), num)
			}

			if err := h.RemoveNumbersFlowID(ctx, tt.flowID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
