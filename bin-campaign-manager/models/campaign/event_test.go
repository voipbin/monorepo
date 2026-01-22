package campaign

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_campaign_created", EventTypeCampaignCreated, "campaign_created"},
		{"event_type_campaign_updated", EventTypeCampaignUpdated, "campaign_updated"},
		{"event_type_campaign_deleted", EventTypeCampaignDeleted, "campaign_deleted"},
		{"event_type_campaign_status_run", EventTypeCampaignStatusRun, "campaign_status_run"},
		{"event_type_campaign_status_stopping", EventTypeCampaignStatusStopping, "campaign_status_stopping"},
		{"event_type_campaign_status_stop", EventTypeCampaignStatusStop, "campaign_status_stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
