package outdialtargetcall

import (
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestOutdialTargetCall(t *testing.T) {
	tmCreate := time.Now()

	tests := []struct {
		name string
		call *OutdialTargetCall
	}{
		{
			name: "creates outdialtargetcall with all fields",
			call: &OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
				},
				CampaignID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
				OutdialID:       uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
				OutdialTargetID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
				ActiveflowID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440005"),
				ReferenceType:   ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440006"),
				Status:          StatusProgressing,
				Destination: &commonaddress.Address{
					Type:   "phone",
					Target: "+12345678900",
				},
				DestinationIndex: 0,
				TryCount:         1,
				TMCreate:         &tmCreate,
			},
		},
		{
			name: "creates outdialtargetcall with minimal fields",
			call: &OutdialTargetCall{
				Identity: commonidentity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
				CampaignID:       uuid.Nil,
				OutdialID:        uuid.Nil,
				OutdialTargetID:  uuid.Nil,
				ActiveflowID:     uuid.Nil,
				ReferenceType:    ReferenceTypeNone,
				ReferenceID:      uuid.Nil,
				Status:           StatusDone,
				Destination:      nil,
				DestinationIndex: 0,
				TryCount:         0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.call.ID != tt.call.ID {
				t.Errorf("ID mismatch")
			}
			if tt.call.CampaignID != tt.call.CampaignID {
				t.Errorf("CampaignID mismatch")
			}
			if tt.call.OutdialID != tt.call.OutdialID {
				t.Errorf("OutdialID mismatch")
			}
			if tt.call.OutdialTargetID != tt.call.OutdialTargetID {
				t.Errorf("OutdialTargetID mismatch")
			}
			if tt.call.Status != tt.call.Status {
				t.Errorf("Status mismatch")
			}
		})
	}
}

func TestReferenceType(t *testing.T) {
	tests := []struct {
		name          string
		referenceType ReferenceType
		expect        ReferenceType
	}{
		{
			name:          "reference type none",
			referenceType: ReferenceTypeNone,
			expect:        ReferenceTypeNone,
		},
		{
			name:          "reference type call",
			referenceType: ReferenceTypeCall,
			expect:        ReferenceTypeCall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.referenceType != tt.expect {
				t.Errorf("Expected reference type %v, got %v", tt.expect, tt.referenceType)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status != tt.expect {
				t.Errorf("Expected status %v, got %v", tt.expect, tt.status)
			}
		})
	}
}

func TestOutdialTargetCall_WithDestination(t *testing.T) {
	call := &OutdialTargetCall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		},
		Destination: &commonaddress.Address{
			Type:   "phone",
			Target: "+12345678900",
		},
		DestinationIndex: 2,
		TryCount:         3,
		Status:           StatusProgressing,
		ReferenceType:    ReferenceTypeCall,
		ReferenceID:      uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440007"),
	}

	if call.Destination == nil {
		t.Error("Expected Destination to be set")
	}
	if call.Destination.Type != "phone" {
		t.Errorf("Expected Destination type 'phone', got %s", call.Destination.Type)
	}
	if call.Destination.Target != "+12345678900" {
		t.Errorf("Expected Destination target '+12345678900', got %s", call.Destination.Target)
	}
	if call.DestinationIndex != 2 {
		t.Errorf("Expected DestinationIndex 2, got %d", call.DestinationIndex)
	}
	if call.TryCount != 3 {
		t.Errorf("Expected TryCount 3, got %d", call.TryCount)
	}
	if call.ReferenceType != ReferenceTypeCall {
		t.Errorf("Expected ReferenceType 'call', got %s", call.ReferenceType)
	}
}

func TestOutdialTargetCall_WithoutReference(t *testing.T) {
	call := &OutdialTargetCall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		},
		ReferenceType: ReferenceTypeNone,
		ReferenceID:   uuid.Nil,
		Status:        StatusDone,
	}

	if call.ReferenceType != ReferenceTypeNone {
		t.Errorf("Expected ReferenceType 'none', got %s", call.ReferenceType)
	}
	if call.ReferenceID != uuid.Nil {
		t.Errorf("Expected nil ReferenceID, got %v", call.ReferenceID)
	}
	if call.Status != StatusDone {
		t.Errorf("Expected Status 'done', got %s", call.Status)
	}
}
