package target

import (
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
)

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestTargetStruct(t *testing.T) {
	tmUpdate := timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	target := Target{
		Destination: commonaddress.Address{
			Type:   commonaddress.TypeTel,
			Target: "+15551234567",
		},
		Status:   StatusSent,
		Parts:    2,
		TMUpdate: tmUpdate,
	}

	if target.Destination.Type != commonaddress.TypeTel {
		t.Errorf("Target.Destination.Type = %v, expected %v", target.Destination.Type, commonaddress.TypeTel)
	}
	if target.Destination.Target != "+15551234567" {
		t.Errorf("Target.Destination.Target = %v, expected %v", target.Destination.Target, "+15551234567")
	}
	if target.Status != StatusSent {
		t.Errorf("Target.Status = %v, expected %v", target.Status, StatusSent)
	}
	if target.Parts != 2 {
		t.Errorf("Target.Parts = %v, expected %v", target.Parts, 2)
	}
	if target.TMUpdate == nil || !target.TMUpdate.Equal(*tmUpdate) {
		t.Errorf("Target.TMUpdate = %v, expected %v", target.TMUpdate, tmUpdate)
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_received", StatusReceived, "received"},
		{"status_queued", StatusQueued, "queued"},
		{"status_gw_timeout", StatusGWTimeout, "gw_timeout"},
		{"status_sent", StatusSent, "sent"},
		{"status_dlr_timeout", StatusDLRTimeout, "dlr_timeout"},
		{"status_failed", StatusFailed, "failed"},
		{"status_delivered", StatusDelivered, "delivered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestTargetWithDifferentStatuses(t *testing.T) {
	statuses := []Status{
		StatusReceived,
		StatusQueued,
		StatusGWTimeout,
		StatusSent,
		StatusDLRTimeout,
		StatusFailed,
		StatusDelivered,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			target := Target{
				Status: status,
			}
			if target.Status != status {
				t.Errorf("Target.Status = %v, expected %v", target.Status, status)
			}
		})
	}
}
