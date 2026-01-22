package outdial

import (
	"testing"

	"github.com/gofrs/uuid"
)

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
