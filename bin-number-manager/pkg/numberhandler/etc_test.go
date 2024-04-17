package numberhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_generateTags(t *testing.T) {

	tests := []struct {
		name string

		number *number.Number

		expectRes []string
	}{
		{
			name: "normal",

			number: &number.Number{
				ID:         uuid.FromStringOrNil("7808a85c-f684-11ee-b64f-d3a82dd85406"),
				CustomerID: uuid.FromStringOrNil("8010617a-f684-11ee-b310-53812305ffd5"),
			},
			expectRes: []string{
				"CustomerID_8010617a-f684-11ee-b310-53812305ffd5",
				"NumberID_7808a85c-f684-11ee-b64f-d3a82dd85406",
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

			res := h.generateTags(ctx, tt.number)

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
