package campaigncall

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {

	tests := []struct {
		name string

		data Campaigncall

		expectRes *WebhookMessage
	}{
		{
			name: "normal",

			data: Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14247a04-6ee0-11ee-be32-4794d521adda"),
					CustomerID: uuid.FromStringOrNil("14588a38-6ee0-11ee-93ce-1f8e4cd59d79"),
				},
				CampaignID:      uuid.FromStringOrNil("1482f8a4-6ee0-11ee-b4f2-d35541a3314d"),
				OutplanID:       uuid.FromStringOrNil("14b05290-6ee0-11ee-b509-9bf668e2e4b8"),
				OutdialID:       uuid.FromStringOrNil("14e049c8-6ee0-11ee-975b-777be1d5fc2f"),
				OutdialTargetID: uuid.FromStringOrNil("151307fa-6ee0-11ee-801d-e72c01cc7e31"),
				QueueID:         uuid.FromStringOrNil("15468760-6ee0-11ee-84f2-5b145e606cf1"),
				ActiveflowID:    uuid.FromStringOrNil("1576af62-6ee0-11ee-8f41-4b0c3037ac9a"),
				FlowID:          uuid.FromStringOrNil("15a41ef2-6ee0-11ee-ad81-ab9ecafe16d3"),
				ReferenceType:   ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("15db949a-6ee0-11ee-9630-0fc99b395d5d"),
				Status:          StatusProgressing,
				Result:          ResultSuccess,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         0,
				TMCreate:         "2020-10-10 03:30:17.000000",
				TMUpdate:         "2020-10-10 03:31:17.000000",
			},

			expectRes: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14247a04-6ee0-11ee-be32-4794d521adda"),
					CustomerID: uuid.FromStringOrNil("14588a38-6ee0-11ee-93ce-1f8e4cd59d79"),
				},
				CampaignID:      uuid.FromStringOrNil("1482f8a4-6ee0-11ee-b4f2-d35541a3314d"),
				OutplanID:       uuid.FromStringOrNil("14b05290-6ee0-11ee-b509-9bf668e2e4b8"),
				OutdialID:       uuid.FromStringOrNil("14e049c8-6ee0-11ee-975b-777be1d5fc2f"),
				OutdialTargetID: uuid.FromStringOrNil("151307fa-6ee0-11ee-801d-e72c01cc7e31"),
				QueueID:         uuid.FromStringOrNil("15468760-6ee0-11ee-84f2-5b145e606cf1"),
				ActiveflowID:    uuid.FromStringOrNil("1576af62-6ee0-11ee-8f41-4b0c3037ac9a"),
				FlowID:          uuid.FromStringOrNil("15a41ef2-6ee0-11ee-ad81-ab9ecafe16d3"),
				ReferenceType:   ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("15db949a-6ee0-11ee-9630-0fc99b395d5d"),
				Status:          StatusProgressing,
				Result:          ResultSuccess,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				DestinationIndex: 0,
				TryCount:         0,
				TMCreate:         "2020-10-10 03:30:17.000000",
				TMUpdate:         "2020-10-10 03:31:17.000000",
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
