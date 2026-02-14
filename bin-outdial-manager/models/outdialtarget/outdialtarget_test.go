package outdialtarget

import (
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

func TestOutdialTarget(t *testing.T) {
	tmCreate := time.Now()

	tests := []struct {
		name   string
		target *OutdialTarget
	}{
		{
			name: "creates outdialtarget with all fields",
			target: &OutdialTarget{
				ID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				OutdialID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				Name:      "Test Target",
				Detail:    "Test Detail",
				Data:      `{"key": "value"}`,
				Status:    StatusIdle,
				Destination0: &commonaddress.Address{
					Type:   "phone",
					Target: "+12345678900",
				},
				Destination1: &commonaddress.Address{
					Type:   "phone",
					Target: "+12345678901",
				},
				TryCount0: 0,
				TryCount1: 0,
				TryCount2: 0,
				TryCount3: 0,
				TryCount4: 0,
				TMCreate:  &tmCreate,
			},
		},
		{
			name: "creates outdialtarget with minimal fields",
			target: &OutdialTarget{
				ID:           uuid.Nil,
				OutdialID:    uuid.Nil,
				Name:         "",
				Detail:       "",
				Data:         "",
				Status:       StatusIdle,
				Destination0: nil,
				Destination1: nil,
				Destination2: nil,
				Destination3: nil,
				Destination4: nil,
				TryCount0:    0,
				TryCount1:    0,
				TryCount2:    0,
				TryCount3:    0,
				TryCount4:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.target.ID != tt.target.ID {
				t.Errorf("ID mismatch")
			}
			if tt.target.OutdialID != tt.target.OutdialID {
				t.Errorf("OutdialID mismatch")
			}
			if tt.target.Name != tt.target.Name {
				t.Errorf("Name mismatch")
			}
			if tt.target.Status != tt.target.Status {
				t.Errorf("Status mismatch")
			}
		})
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		expect Status
	}{
		{
			name:   "status progressing",
			status: StatusProgressing,
			expect: StatusProgressing,
		},
		{
			name:   "status done",
			status: StatusDone,
			expect: StatusDone,
		},
		{
			name:   "status idle",
			status: StatusIdle,
			expect: StatusIdle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expect {
				t.Errorf("Expected status %v, got %v", tt.expect, tt.status)
			}
		})
	}
}

func TestOutdialTarget_Destinations(t *testing.T) {
	target := &OutdialTarget{
		ID:        uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
		OutdialID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		Destination0: &commonaddress.Address{
			Type:   "phone",
			Target: "+12345678900",
		},
		Destination1: &commonaddress.Address{
			Type:   "phone",
			Target: "+12345678901",
		},
		Destination2: &commonaddress.Address{
			Type:   "email",
			Target: "test@example.com",
		},
		TryCount0: 1,
		TryCount1: 2,
		TryCount2: 0,
		TryCount3: 0,
		TryCount4: 0,
	}

	if target.Destination0 == nil {
		t.Error("Expected Destination0 to be set")
	}
	if target.Destination0.Type != "phone" {
		t.Errorf("Expected Destination0 type 'phone', got %s", target.Destination0.Type)
	}
	if target.Destination0.Target != "+12345678900" {
		t.Errorf("Expected Destination0 target '+12345678900', got %s", target.Destination0.Target)
	}

	if target.TryCount0 != 1 {
		t.Errorf("Expected TryCount0 1, got %d", target.TryCount0)
	}
	if target.TryCount1 != 2 {
		t.Errorf("Expected TryCount1 2, got %d", target.TryCount1)
	}
}
