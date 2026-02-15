package recording

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestRecordingStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())

	r := Recording{
		ActiveflowID:  activeflowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusRecording,
		Format:        FormatWAV,
		OnEndFlowID:   onEndFlowID,
		RecordingName: "test-recording",
		Filenames:     []string{"file1.wav", "file2.wav"},
		AsteriskID:    "asterisk-1",
		ChannelIDs:    []string{"channel-1", "channel-2"},
	}
	r.ID = id

	if r.ID != id {
		t.Errorf("Recording.ID = %v, expected %v", r.ID, id)
	}
	if r.ActiveflowID != activeflowID {
		t.Errorf("Recording.ActiveflowID = %v, expected %v", r.ActiveflowID, activeflowID)
	}
	if r.ReferenceType != ReferenceTypeCall {
		t.Errorf("Recording.ReferenceType = %v, expected %v", r.ReferenceType, ReferenceTypeCall)
	}
	if r.ReferenceID != referenceID {
		t.Errorf("Recording.ReferenceID = %v, expected %v", r.ReferenceID, referenceID)
	}
	if r.Status != StatusRecording {
		t.Errorf("Recording.Status = %v, expected %v", r.Status, StatusRecording)
	}
	if r.Format != FormatWAV {
		t.Errorf("Recording.Format = %v, expected %v", r.Format, FormatWAV)
	}
	if r.OnEndFlowID != onEndFlowID {
		t.Errorf("Recording.OnEndFlowID = %v, expected %v", r.OnEndFlowID, onEndFlowID)
	}
	if r.RecordingName != "test-recording" {
		t.Errorf("Recording.RecordingName = %v, expected %v", r.RecordingName, "test-recording")
	}
	if len(r.Filenames) != 2 {
		t.Errorf("Recording.Filenames length = %v, expected %v", len(r.Filenames), 2)
	}
	if r.AsteriskID != "asterisk-1" {
		t.Errorf("Recording.AsteriskID = %v, expected %v", r.AsteriskID, "asterisk-1")
	}
	if len(r.ChannelIDs) != 2 {
		t.Errorf("Recording.ChannelIDs length = %v, expected %v", len(r.ChannelIDs), 2)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_confbridge", ReferenceTypeConfbridge, "confbridge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_initiating", StatusInitiating, "initiating"},
		{"status_recording", StatusRecording, "recording"},
		{"status_stopping", StatusStopping, "stopping"},
		{"status_ended", StatusEnded, "ended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestFormatConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Format
		expected string
	}{
		{"format_wav", FormatWAV, "wav"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestConvertWebhookMessage(t *testing.T) {
	now := time.Now()
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	onEndFlowID := uuid.Must(uuid.NewV4())

	r := &Recording{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   ownerID,
		},
		ActiveflowID:  activeflowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusRecording,
		Format:        FormatWAV,
		OnEndFlowID:   onEndFlowID,
		TMStart:       &now,
		TMCreate:      &now,
		TMUpdate:      &now,
	}

	webhook := r.ConvertWebhookMessage()

	if webhook.ID != id {
		t.Errorf("ConvertWebhookMessage ID = %v, expected %v", webhook.ID, id)
	}
	if webhook.CustomerID != customerID {
		t.Errorf("ConvertWebhookMessage CustomerID = %v, expected %v", webhook.CustomerID, customerID)
	}
	if webhook.ActiveflowID != activeflowID {
		t.Errorf("ConvertWebhookMessage ActiveflowID = %v, expected %v", webhook.ActiveflowID, activeflowID)
	}
	if webhook.ReferenceType != ReferenceTypeCall {
		t.Errorf("ConvertWebhookMessage ReferenceType = %v, expected %v", webhook.ReferenceType, ReferenceTypeCall)
	}
	if webhook.ReferenceID != referenceID {
		t.Errorf("ConvertWebhookMessage ReferenceID = %v, expected %v", webhook.ReferenceID, referenceID)
	}
	if webhook.Status != StatusRecording {
		t.Errorf("ConvertWebhookMessage Status = %v, expected %v", webhook.Status, StatusRecording)
	}
	if webhook.Format != FormatWAV {
		t.Errorf("ConvertWebhookMessage Format = %v, expected %v", webhook.Format, FormatWAV)
	}
	if webhook.OnEndFlowID != onEndFlowID {
		t.Errorf("ConvertWebhookMessage OnEndFlowID = %v, expected %v", webhook.OnEndFlowID, onEndFlowID)
	}
	if webhook.TMStart == nil || !webhook.TMStart.Equal(now) {
		t.Errorf("ConvertWebhookMessage TMStart mismatch")
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	r := &Recording{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{},
		ActiveflowID:  activeflowID,
		ReferenceType: ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        StatusRecording,
		Format:        FormatWAV,
	}

	data, err := r.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent returned error: %v", err)
	}

	if len(data) == 0 {
		t.Error("CreateWebhookEvent returned empty data")
	}

	// Verify it's valid JSON
	var webhook WebhookMessage
	if err := json.Unmarshal(data, &webhook); err != nil {
		t.Errorf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	if webhook.ID != id {
		t.Errorf("Unmarshalled webhook ID = %v, expected %v", webhook.ID, id)
	}
	if webhook.Status != StatusRecording {
		t.Errorf("Unmarshalled webhook Status = %v, expected %v", webhook.Status, StatusRecording)
	}
	if webhook.Format != FormatWAV {
		t.Errorf("Unmarshalled webhook Format = %v, expected %v", webhook.Format, FormatWAV)
	}
}
