package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("57214714-f168-11ee-9706-6f34dc976036"),

			responseActiveflow: &activeflow.Activeflow{
				ID:       uuid.FromStringOrNil("57214714-f168-11ee-9706-6f34dc976036"),
				Status:   activeflow.StatusEnded,
				TMDelete: dbhandler.DefaultTimeStamp,
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &activeflowHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().ActiveflowGet(ctx, tt.id).Return(tt.responseActiveflow, nil)

			mockDB.EXPECT().ActiveflowDelete(ctx, tt.responseActiveflow.ID).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.responseActiveflow.ID).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowDeleted, tt.responseActiveflow)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseActiveflow) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseActiveflow, res)
			}
		})
	}
}
