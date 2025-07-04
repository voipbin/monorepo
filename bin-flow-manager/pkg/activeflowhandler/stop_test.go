package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_Stop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseActiveflow *activeflow.Activeflow
		responseDBCurTime  string

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("6d8a9464-c8d9-11ed-abfb-d3b58a5adf22"),

			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6d8a9464-c8d9-11ed-abfb-d3b58a5adf22"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
			},

			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldStatus: activeflow.StatusEnded,
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
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.id, tt.expectUpdateFields).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseActiveflow, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseActiveflow, nil)
			}
		})
	}
}
