package flow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
)

func TestConvertWebhookMessage(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	onCompleteFlowID := uuid.Must(uuid.NewV4())
	tmCreate := time.Now()
	tmUpdate := time.Now()

	f := &Flow{
		Type:             TypeFlow,
		Name:             "test-flow",
		Detail:           "test detail",
		Actions:          []action.Action{{ID: uuid.Must(uuid.NewV4()), Type: action.TypeAnswer}},
		OnCompleteFlowID: onCompleteFlowID,
		TMCreate:         &tmCreate,
		TMUpdate:         &tmUpdate,
	}
	f.ID = id
	f.CustomerID = customerID

	wm := f.ConvertWebhookMessage()

	if wm.ID != f.ID {
		t.Errorf("WebhookMessage.ID = %v, expected %v", wm.ID, f.ID)
	}
	if wm.CustomerID != f.CustomerID {
		t.Errorf("WebhookMessage.CustomerID = %v, expected %v", wm.CustomerID, f.CustomerID)
	}
	if wm.Type != f.Type {
		t.Errorf("WebhookMessage.Type = %v, expected %v", wm.Type, f.Type)
	}
	if wm.Name != f.Name {
		t.Errorf("WebhookMessage.Name = %v, expected %v", wm.Name, f.Name)
	}
	if wm.Detail != f.Detail {
		t.Errorf("WebhookMessage.Detail = %v, expected %v", wm.Detail, f.Detail)
	}
	if wm.OnCompleteFlowID != f.OnCompleteFlowID {
		t.Errorf("WebhookMessage.OnCompleteFlowID = %v, expected %v", wm.OnCompleteFlowID, f.OnCompleteFlowID)
	}
	if len(wm.Actions) != len(f.Actions) {
		t.Errorf("WebhookMessage.Actions length = %v, expected %v", len(wm.Actions), len(f.Actions))
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	f := &Flow{
		Type:   TypeFlow,
		Name:   "test-flow",
		Detail: "test detail",
	}
	f.ID = id
	f.CustomerID = customerID

	data, err := f.CreateWebhookEvent()
	if err != nil {
		t.Errorf("CreateWebhookEvent() error = %v, expected nil", err)
	}

	if len(data) == 0 {
		t.Error("CreateWebhookEvent() returned empty data")
	}

	// Verify it's valid JSON
	var wm WebhookMessage
	if err := json.Unmarshal(data, &wm); err != nil {
		t.Errorf("CreateWebhookEvent() returned invalid JSON: %v", err)
	}

	if wm.ID != f.ID {
		t.Errorf("Unmarshaled WebhookMessage.ID = %v, expected %v", wm.ID, f.ID)
	}
}
