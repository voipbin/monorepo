package speaking

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	now := time.Now()
	s := &Speaking{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		ReferenceType: streaming.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("r0000000-0000-0000-0000-000000000001"),
		Language:      "en-US",
		Provider:      "elevenlabs",
		VoiceID:       "voice123",
		Direction:     streaming.DirectionIncoming,
		Status:        StatusActive,
		PodID:         "tts-pod-abc-123",
		TMCreate:      &now,
		TMUpdate:      &now,
		TMDelete:      nil,
	}

	msg := s.ConvertWebhookMessage()

	if msg.ID != s.ID {
		t.Errorf("expected ID %s, got %s", s.ID, msg.ID)
	}
	if msg.CustomerID != s.CustomerID {
		t.Errorf("expected CustomerID %s, got %s", s.CustomerID, msg.CustomerID)
	}
	if msg.ReferenceType != s.ReferenceType {
		t.Errorf("expected ReferenceType %s, got %s", s.ReferenceType, msg.ReferenceType)
	}
	if msg.ReferenceID != s.ReferenceID {
		t.Errorf("expected ReferenceID %s, got %s", s.ReferenceID, msg.ReferenceID)
	}
	if msg.Language != s.Language {
		t.Errorf("expected Language %s, got %s", s.Language, msg.Language)
	}
	if msg.Provider != s.Provider {
		t.Errorf("expected Provider %s, got %s", s.Provider, msg.Provider)
	}
	if msg.VoiceID != s.VoiceID {
		t.Errorf("expected VoiceID %s, got %s", s.VoiceID, msg.VoiceID)
	}
	if msg.Direction != s.Direction {
		t.Errorf("expected Direction %s, got %s", s.Direction, msg.Direction)
	}
	if msg.Status != s.Status {
		t.Errorf("expected Status %s, got %s", s.Status, msg.Status)
	}
	if msg.TMCreate != s.TMCreate {
		t.Errorf("expected TMCreate %v, got %v", s.TMCreate, msg.TMCreate)
	}
	if msg.TMUpdate != s.TMUpdate {
		t.Errorf("expected TMUpdate %v, got %v", s.TMUpdate, msg.TMUpdate)
	}
	if msg.TMDelete != s.TMDelete {
		t.Errorf("expected TMDelete %v, got %v", s.TMDelete, msg.TMDelete)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	now := time.Now()
	s := &Speaking{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		ReferenceType: streaming.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("r0000000-0000-0000-0000-000000000001"),
		Language:      "en-US",
		Provider:      "elevenlabs",
		VoiceID:       "voice123",
		Direction:     streaming.DirectionIncoming,
		Status:        StatusActive,
		PodID:         "tts-pod-abc-123",
		TMCreate:      &now,
	}

	data, err := s.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	// pod_id must NOT appear in the external JSON
	if _, exists := parsed["pod_id"]; exists {
		t.Error("expected pod_id to be absent from webhook JSON, but it was present")
	}

	// verify expected fields are present
	if parsed["id"] != s.ID.String() {
		t.Errorf("expected id %s in JSON, got %v", s.ID, parsed["id"])
	}
	if parsed["language"] != "en-US" {
		t.Errorf("expected language en-US in JSON, got %v", parsed["language"])
	}
	if parsed["provider"] != "elevenlabs" {
		t.Errorf("expected provider elevenlabs in JSON, got %v", parsed["provider"])
	}
	if parsed["voice_id"] != "voice123" {
		t.Errorf("expected voice_id voice123 in JSON, got %v", parsed["voice_id"])
	}
}

func Test_CreateWebhookEvent_EmptyPodID(t *testing.T) {
	s := &Speaking{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001"),
		},
		Status: StatusActive,
		PodID:  "",
	}

	data, err := s.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	if _, exists := parsed["pod_id"]; exists {
		t.Error("expected pod_id to be absent from webhook JSON even when empty")
	}
}
