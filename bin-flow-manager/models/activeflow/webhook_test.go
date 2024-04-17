package activeflow

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/stack"
)

func Test_ConvertWebhookMessage(t *testing.T) {

	tests := []struct {
		name string

		activeflow *Activeflow

		expectRes *WebhookMessage
	}{
		{
			name: "string equal match",

			activeflow: &Activeflow{
				ID:            uuid.FromStringOrNil("3e6879cc-bc6b-11ee-8ba5-1fd7ab9b740f"),
				CustomerID:    uuid.FromStringOrNil("7a112bae-bc6b-11ee-912a-bb8b9a34d084"),
				FlowID:        uuid.FromStringOrNil("7a717810-bc6b-11ee-ba93-17ff10a17809"),
				Status:        StatusRunning,
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("7a9a4056-bc6b-11ee-8dc5-37ff818a164d"),
				StackMap: map[uuid.UUID]*stack.Stack{
					uuid.FromStringOrNil("9e90cf3e-bc6b-11ee-907f-a71ca36fa0ca"): {
						ID: uuid.FromStringOrNil("9e90cf3e-bc6b-11ee-907f-a71ca36fa0ca"),
						Actions: []action.Action{
							{
								ID: uuid.FromStringOrNil("9eb9b3a4-bc6b-11ee-9e6b-3f556df3c586"),
							},
						},
					},
				},
				CurrentStackID: uuid.FromStringOrNil("9e90cf3e-bc6b-11ee-907f-a71ca36fa0ca"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("9eb9b3a4-bc6b-11ee-9e6b-3f556df3c586"),
					Type: action.TypeBeep,
				},
				ForwardStackID:  uuid.FromStringOrNil("9ee81b2c-bc6b-11ee-9e92-2753cddf0d26"),
				ForwardActionID: uuid.FromStringOrNil("9f1531e8-bc6b-11ee-8da3-fff5045b5ac8"),
				ExecuteCount:    1,
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9f3e1e14-bc6b-11ee-8c34-27a8a450e79f"),
						Type: action.TypeAnswer,
					},
				},
				TMCreate: "2022-04-18 03:22:17.995000",
				TMUpdate: "2022-04-18 03:22:17.995000",
				TMDelete: "9999-01-01 00:00:000",
			},

			expectRes: &WebhookMessage{
				ID:            uuid.FromStringOrNil("3e6879cc-bc6b-11ee-8ba5-1fd7ab9b740f"),
				CustomerID:    uuid.FromStringOrNil("7a112bae-bc6b-11ee-912a-bb8b9a34d084"),
				FlowID:        uuid.FromStringOrNil("7a717810-bc6b-11ee-ba93-17ff10a17809"),
				Status:        StatusRunning,
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("7a9a4056-bc6b-11ee-8dc5-37ff818a164d"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("9eb9b3a4-bc6b-11ee-9e6b-3f556df3c586"),
					Type: action.TypeBeep,
				},
				ForwardActionID: uuid.FromStringOrNil("9f1531e8-bc6b-11ee-8da3-fff5045b5ac8"),
				ExecutedActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9f3e1e14-bc6b-11ee-8c34-27a8a450e79f"),
						Type: action.TypeAnswer,
					},
				},
				TMCreate: "2022-04-18 03:22:17.995000",
				TMUpdate: "2022-04-18 03:22:17.995000",
				TMDelete: "9999-01-01 00:00:000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := tt.activeflow.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
