package activeflowhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func Test_Execute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &activeflowHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	tests := []struct {
		name string

		id uuid.UUID

		responseActiveFlow *activeflow.ActiveFlow

		flow *flow.Flow

		refereceType activeflow.ReferenceType
		referenceID  uuid.UUID
		expectActive *activeflow.ActiveFlow
	}{
		{
			"normal",

			uuid.FromStringOrNil("bef23280-a7ab-11ec-8e79-1b236556e34d"),

			&activeflow.ActiveFlow{
				ID: uuid.FromStringOrNil("bef23280-a7ab-11ec-8e79-1b236556e34d"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("13c4e65e-a7ac-11ec-971e-0374e19101d3"),
						Type: action.TypeAnswer,
					},
				},
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
			},

			&flow.Flow{
				ID:      uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				Actions: []action.Action{},
			},

			activeflow.ReferenceTypeCall,
			uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
			&activeflow.ActiveFlow{
				ID:            uuid.FromStringOrNil("32808a7c-a7a1-11ec-8de8-2331c11da2e8"),
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("03e8a480-822f-11eb-b71f-8bbc09fa1e7a"),
				FlowID:        uuid.FromStringOrNil("dc8e048e-822e-11eb-8cb6-235002e45cf2"),
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
				ExecuteCount:    0,
				ForwardActionID: action.IDEmpty,
				Actions:         []action.Action{},
				ExecutedActions: []action.Action{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().ActiveFlowGet(ctx, tt.id).Return(tt.responseActiveFlow, nil).AnyTimes()
			mockDB.EXPECT().ActiveFlowSet(ctx, gomock.Any()).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseActiveFlow.CustomerID, activeflow.EventTypeActiveFlowUpdated, tt.responseActiveFlow)

			if err := h.Execute(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
