package activeflow

import (
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/stack"
)

func Test_ConvertWebhookMessage(t *testing.T) {

	tmCreate := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		activeflow *Activeflow

		expectedRes *WebhookMessage
	}{
		{
			name: "string equal match",

			activeflow: &Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3e6879cc-bc6b-11ee-8ba5-1fd7ab9b740f"),
					CustomerID: uuid.FromStringOrNil("7a112bae-bc6b-11ee-912a-bb8b9a34d084"),
				},
				FlowID:           uuid.FromStringOrNil("7a717810-bc6b-11ee-ba93-17ff10a17809"),
				Status:           StatusRunning,
				ReferenceType:    ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("7a9a4056-bc6b-11ee-8dc5-37ff818a164d"),
				OnCompleteFlowID: uuid.FromStringOrNil("4c015af2-ce18-11f0-89f9-23476a716300"),
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
				TMCreate: &tmCreate,
				TMUpdate: &tmUpdate,
				TMDelete: nil,
			},

			expectedRes: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3e6879cc-bc6b-11ee-8ba5-1fd7ab9b740f"),
					CustomerID: uuid.FromStringOrNil("7a112bae-bc6b-11ee-912a-bb8b9a34d084"),
				},
				FlowID:           uuid.FromStringOrNil("7a717810-bc6b-11ee-ba93-17ff10a17809"),
				Status:           StatusRunning,
				ReferenceType:    ReferenceTypeCall,
				ReferenceID:      uuid.FromStringOrNil("7a9a4056-bc6b-11ee-8dc5-37ff818a164d"),
				OnCompleteFlowID: uuid.FromStringOrNil("4c015af2-ce18-11f0-89f9-23476a716300"),
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
				TMCreate: &tmCreate,
				TMUpdate: &tmUpdate,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := tt.activeflow.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name       string
		activeflow *Activeflow
		wantErr    bool
	}{
		{
			name: "valid_activeflow",
			activeflow: &Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3e6879cc-bc6b-11ee-8ba5-1fd7ab9b740f"),
					CustomerID: uuid.FromStringOrNil("7a112bae-bc6b-11ee-912a-bb8b9a34d084"),
				},
				FlowID:        uuid.FromStringOrNil("7a717810-bc6b-11ee-ba93-17ff10a17809"),
				Status:        StatusRunning,
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("7a9a4056-bc6b-11ee-8dc5-37ff818a164d"),
				TMCreate:      &tmCreate,
				TMUpdate:      &tmUpdate,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.activeflow.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(data) == 0 {
				t.Error("CreateWebhookEvent() returned empty data")
			}
		})
	}
}
