package campaign

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {

	tests := []struct {
		name string

		data Campaign

		expectRes *WebhookMessage
	}{
		{
			name: "normal",

			data: Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9e375ad0-8e62-11ee-8a45-530d00e19a2d"),
					CustomerID: uuid.FromStringOrNil("9e6bdbfc-8e62-11ee-b13f-dfdb25fe3da2"),
				},
				Type:         TypeCall,
				Execute:      ExecuteRun,
				Name:         "test name",
				Detail:       "test detail",
				Status:       StatusRun,
				ServiceLevel: 100,
				EndHandle:    EndHandleContinue,
				FlowID:       uuid.FromStringOrNil("9e9ae866-8e62-11ee-8a35-27115e9fbde4"),
				Actions: []fmaction.Action{
					{
						ID: uuid.FromStringOrNil("9ed28910-8e62-11ee-8ad4-7b973956a30e"),
					},
				},
				OutplanID:      uuid.FromStringOrNil("e17f7386-8e62-11ee-b43a-2f21a207ec0a"),
				OutdialID:      uuid.FromStringOrNil("e1ad39ba-8e62-11ee-87c5-2ba34c5eda2d"),
				QueueID:        uuid.FromStringOrNil("e1d9e2bc-8e62-11ee-a4f9-ef02c916b7d2"),
				NextCampaignID: uuid.FromStringOrNil("e202a06c-8e62-11ee-b00e-ab0605192f93"),
				TMCreate:       "2020-10-10T03:30:17.000000Z",
				TMUpdate:       "2020-10-10T03:31:17.000000Z",
				TMDelete:       "9999-01-01 00:00:000",
			},

			expectRes: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9e375ad0-8e62-11ee-8a45-530d00e19a2d"),
					CustomerID: uuid.FromStringOrNil("9e6bdbfc-8e62-11ee-b13f-dfdb25fe3da2"),
				},
				Type:         TypeCall,
				Name:         "test name",
				Detail:       "test detail",
				Status:       StatusRun,
				ServiceLevel: 100,
				EndHandle:    EndHandleContinue,
				Actions: []fmaction.Action{
					{
						ID: uuid.FromStringOrNil("9ed28910-8e62-11ee-8ad4-7b973956a30e"),
					},
				},
				OutplanID:      uuid.FromStringOrNil("e17f7386-8e62-11ee-b43a-2f21a207ec0a"),
				OutdialID:      uuid.FromStringOrNil("e1ad39ba-8e62-11ee-87c5-2ba34c5eda2d"),
				QueueID:        uuid.FromStringOrNil("e1d9e2bc-8e62-11ee-a4f9-ef02c916b7d2"),
				NextCampaignID: uuid.FromStringOrNil("e202a06c-8e62-11ee-b00e-ab0605192f93"),
				TMCreate:       "2020-10-10T03:30:17.000000Z",
				TMUpdate:       "2020-10-10T03:31:17.000000Z",
				TMDelete:       "9999-01-01 00:00:000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.data.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
