package account

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestAccountStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	a := Account{
		ID:             id,
		CustomerID:     customerID,
		TotalFileCount: 100,
		TotalFileSize:  1073741824, // 1GB
		TMCreate:       &tmCreate,
		TMUpdate:       &tmUpdate,
		TMDelete:       nil,
	}

	if a.ID != id {
		t.Errorf("Account.ID = %v, expected %v", a.ID, id)
	}
	if a.CustomerID != customerID {
		t.Errorf("Account.CustomerID = %v, expected %v", a.CustomerID, customerID)
	}
	if a.TotalFileCount != 100 {
		t.Errorf("Account.TotalFileCount = %v, expected %v", a.TotalFileCount, 100)
	}
	if a.TotalFileSize != 1073741824 {
		t.Errorf("Account.TotalFileSize = %v, expected %v", a.TotalFileSize, 1073741824)
	}
	if !a.TMCreate.Equal(tmCreate) {
		t.Errorf("Account.TMCreate = %v, expected %v", a.TMCreate, tmCreate)
	}
	if !a.TMUpdate.Equal(tmUpdate) {
		t.Errorf("Account.TMUpdate = %v, expected %v", a.TMUpdate, tmUpdate)
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
				"total_file_count": int64(100),
				"total_file_size":  int64(1024),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result map[Field]any) {
				if val, ok := result[FieldTotalFileCount]; !ok || val != int64(100) {
					t.Errorf("FieldTotalFileCount = %v, expected 100", val)
				}
				if val, ok := result[FieldTotalFileSize]; !ok || val != int64(1024) {
					t.Errorf("FieldTotalFileSize = %v, expected 1024", val)
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
	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	a := &Account{
		ID:             id,
		CustomerID:     customerID,
		TotalFileCount: 100,
		TotalFileSize:  1073741824,
		TMCreate:       &tmCreate,
		TMUpdate:       &tmUpdate,
		TMDelete:       nil,
	}

	msg := a.ConvertWebhookMessage()

	if msg.ID != id {
		t.Errorf("WebhookMessage.ID = %v, expected %v", msg.ID, id)
	}
	if msg.CustomerID != customerID {
		t.Errorf("WebhookMessage.CustomerID = %v, expected %v", msg.CustomerID, customerID)
	}
	if msg.TotalFileCount != 100 {
		t.Errorf("WebhookMessage.TotalFileCount = %v, expected %v", msg.TotalFileCount, 100)
	}
	if msg.TotalFileSize != 1073741824 {
		t.Errorf("WebhookMessage.TotalFileSize = %v, expected %v", msg.TotalFileSize, 1073741824)
	}
	if !msg.TMCreate.Equal(tmCreate) {
		t.Errorf("WebhookMessage.TMCreate = %v, expected %v", msg.TMCreate, tmCreate)
	}
	if !msg.TMUpdate.Equal(tmUpdate) {
		t.Errorf("WebhookMessage.TMUpdate = %v, expected %v", msg.TMUpdate, tmUpdate)
	}
	if msg.TMDelete != nil {
		t.Errorf("WebhookMessage.TMDelete = %v, expected nil", msg.TMDelete)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	a := &Account{
		ID:             id,
		CustomerID:     customerID,
		TotalFileCount: 50,
		TotalFileSize:  524288,
		TMCreate:       &tmCreate,
	}

	data, err := a.CreateWebhookEvent()
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
	if msg.TotalFileCount != 50 {
		t.Errorf("Unmarshaled TotalFileCount = %v, expected %v", msg.TotalFileCount, 50)
	}
}
