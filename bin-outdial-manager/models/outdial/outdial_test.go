package outdial

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Outdial is published on the global topic exchange `bin-manager.event` and must carry an
// explicit subscription address (VOIP-1419). The assertion pins the POINTER type: the event data
// reaches notifyhandler as a pointer and the interface check matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Outdial)(nil)

// TestOutdialEventSubscriptionID asserts the subscription address is the outdial's OWN id — not
// any of the other uuid-typed fields a wrong implementation could plausibly return. Every uuid is
// distinct, so returning the wrong field fails loudly (mutation check).
func TestOutdialEventSubscriptionID(t *testing.T) {
	outdialID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	campaignID := uuid.Must(uuid.NewV4())

	data := &Outdial{
		Identity: commonidentity.Identity{
			ID:         outdialID,
			CustomerID: customerID,
		},
		CampaignID: campaignID,
		Name:       "test outdial",
	}

	res := data.EventSubscriptionID()
	if res != outdialID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", outdialID.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Outdial must not be addressed by its customer id. got: %s", res)
	}
	if res == campaignID.String() {
		t.Errorf("Outdial must not be addressed by its campaign id. got: %s", res)
	}
}

func TestOutdial(t *testing.T) {
	tests := []struct {
		name string

		campaignID uuid.UUID
		outdialName string
		detail     string
		data       string
	}{
		{
			name: "creates_outdial_with_all_fields",

			campaignID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			outdialName: "Test Outdial",
			detail:     "Test Detail",
			data:       `{"key": "value"}`,
		},
		{
			name: "creates_outdial_with_empty_fields",

			campaignID: uuid.Nil,
			outdialName: "",
			detail:     "",
			data:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Outdial{
				CampaignID: tt.campaignID,
				Name:       tt.outdialName,
				Detail:     tt.detail,
				Data:       tt.data,
			}

			if o.CampaignID != tt.campaignID {
				t.Errorf("Wrong CampaignID. expect: %s, got: %s", tt.campaignID, o.CampaignID)
			}
			if o.Name != tt.outdialName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.outdialName, o.Name)
			}
			if o.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, o.Detail)
			}
			if o.Data != tt.data {
				t.Errorf("Wrong Data. expect: %s, got: %s", tt.data, o.Data)
			}
		})
	}
}
