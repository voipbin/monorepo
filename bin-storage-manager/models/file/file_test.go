package file

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestFileStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	f := File{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerID: ownerID,
		},
		AccountID:     accountID,
		ReferenceType: ReferenceTypeRecording,
		ReferenceID:   referenceID,
		Name:          "test_file.wav",
		Detail:        "Test recording file",
		BucketName:    "voipbin-media",
		Filename:      "recording.wav",
		Filepath:      "recording/2023/01/",
		Filesize:      1048576,
		URIBucket:     "gs://voipbin-media/recording/2023/01/recording.wav",
		URIDownload:   "https://storage.googleapis.com/...",
	}

	if f.ID != id {
		t.Errorf("File.ID = %v, expected %v", f.ID, id)
	}
	if f.CustomerID != customerID {
		t.Errorf("File.CustomerID = %v, expected %v", f.CustomerID, customerID)
	}
	if f.OwnerID != ownerID {
		t.Errorf("File.OwnerID = %v, expected %v", f.OwnerID, ownerID)
	}
	if f.AccountID != accountID {
		t.Errorf("File.AccountID = %v, expected %v", f.AccountID, accountID)
	}
	if f.ReferenceType != ReferenceTypeRecording {
		t.Errorf("File.ReferenceType = %v, expected %v", f.ReferenceType, ReferenceTypeRecording)
	}
	if f.ReferenceID != referenceID {
		t.Errorf("File.ReferenceID = %v, expected %v", f.ReferenceID, referenceID)
	}
	if f.Name != "test_file.wav" {
		t.Errorf("File.Name = %v, expected %v", f.Name, "test_file.wav")
	}
	if f.Detail != "Test recording file" {
		t.Errorf("File.Detail = %v, expected %v", f.Detail, "Test recording file")
	}
	if f.BucketName != "voipbin-media" {
		t.Errorf("File.BucketName = %v, expected %v", f.BucketName, "voipbin-media")
	}
	if f.Filename != "recording.wav" {
		t.Errorf("File.Filename = %v, expected %v", f.Filename, "recording.wav")
	}
	if f.Filepath != "recording/2023/01/" {
		t.Errorf("File.Filepath = %v, expected %v", f.Filepath, "recording/2023/01/")
	}
	if f.Filesize != 1048576 {
		t.Errorf("File.Filesize = %v, expected %v", f.Filesize, 1048576)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_none", ReferenceTypeNone, ""},
		{"reference_type_normal", ReferenceTypeNormal, "normal"},
		{"reference_type_recording", ReferenceTypeRecording, "recording"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantErr   bool
		checkFunc func(t *testing.T, result map[Field]any)
	}{
		{
			name: "valid_conversion",
			input: map[string]any{
				"name":     "test.wav",
				"filesize": int64(1024),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result map[Field]any) {
				if val, ok := result[FieldName]; !ok || val != "test.wav" {
					t.Errorf("FieldName = %v, expected test.wav", val)
				}
				if val, ok := result[FieldFilesize]; !ok || val != int64(1024) {
					t.Errorf("FieldFilesize = %v, expected 1024", val)
				}
			},
		},
		{
			name:    "empty_map",
			input:   map[string]any{},
			wantErr: false,
			checkFunc: func(t *testing.T, result map[Field]any) {
				if len(result) != 0 {
					t.Errorf("Expected empty map, got %d elements", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringMapToFieldMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringMapToFieldMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestConvertWebhookMessage(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	f := &File{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerID: ownerID,
		},
		ReferenceType: ReferenceTypeRecording,
		ReferenceID:   referenceID,
		Name:          "test_file.wav",
		Detail:        "Test recording",
		Filename:      "recording.wav",
		Filesize:      1048576,
		URIDownload:   "https://example.com/download",
		TMCreate:      &tmCreate,
		TMUpdate:      &tmUpdate,
	}

	msg := f.ConvertWebhookMessage()

	if msg.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", msg.ID, id)
	}
	if msg.CustomerID != customerID {
		t.Errorf("WebhookMessage.CustomerID = %v, expected %v", msg.CustomerID, customerID)
	}
	if msg.OwnerID != ownerID {
		t.Errorf("WebhookMessage.OwnerID = %v, expected %v", msg.OwnerID, ownerID)
	}
	if msg.ReferenceType != ReferenceTypeRecording {
		t.Errorf("WebhookMessage.ReferenceType = %v, expected %v", msg.ReferenceType, ReferenceTypeRecording)
	}
	if msg.ReferenceID != referenceID {
		t.Errorf("WebhookMessage.ReferenceID = %v, expected %v", msg.ReferenceID, referenceID)
	}
	if msg.Name != "test_file.wav" {
		t.Errorf("WebhookMessage.Name = %v, expected test_file.wav", msg.Name)
	}
	if msg.Detail != "Test recording" {
		t.Errorf("WebhookMessage.Detail = %v, expected Test recording", msg.Detail)
	}
	if msg.Filename != "recording.wav" {
		t.Errorf("WebhookMessage.Filename = %v, expected recording.wav", msg.Filename)
	}
	if msg.Filesize != 1048576 {
		t.Errorf("WebhookMessage.Filesize = %v, expected 1048576", msg.Filesize)
	}
	if msg.URIDownload != "https://example.com/download" {
		t.Errorf("WebhookMessage.URIDownload = %v, expected https://example.com/download", msg.URIDownload)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	f := &File{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerID: ownerID,
		},
		ReferenceType: ReferenceTypeNormal,
		ReferenceID:   referenceID,
		Name:          "document.pdf",
		Filename:      "doc.pdf",
		Filesize:      2048,
		URIDownload:   "https://example.com/doc.pdf",
		TMCreate:      &tmCreate,
	}

	data, err := f.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent() error = %v", err)
	}

	var msg WebhookMessage
	err = json.Unmarshal(data, &msg)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if msg.ID != id {
		t.Errorf("Unmarshaled ID = %v, expected %v", msg.ID, id)
	}
	if msg.CustomerID != customerID {
		t.Errorf("Unmarshaled CustomerID = %v, expected %v", msg.CustomerID, customerID)
	}
	if msg.Name != "document.pdf" {
		t.Errorf("Unmarshaled Name = %v, expected document.pdf", msg.Name)
	}
	if msg.Filesize != 2048 {
		t.Errorf("Unmarshaled Filesize = %v, expected 2048", msg.Filesize)
	}
}
