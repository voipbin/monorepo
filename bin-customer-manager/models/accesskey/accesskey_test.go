package accesskey

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestAccesskeyStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tmExpire := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	a := Accesskey{
		ID:         id,
		CustomerID: customerID,
		Name:       "Test Accesskey",
		Detail:     "Test accesskey details",
		Token:      "test-token-12345",
		TMExpire:   &tmExpire,
		TMCreate:   &tmCreate,
		TMUpdate:   &tmUpdate,
		TMDelete:   nil,
	}

	if a.ID != id {
		t.Errorf("Accesskey.ID = %v, expected %v", a.ID, id)
	}
	if a.CustomerID != customerID {
		t.Errorf("Accesskey.CustomerID = %v, expected %v", a.CustomerID, customerID)
	}
	if a.Name != "Test Accesskey" {
		t.Errorf("Accesskey.Name = %v, expected %v", a.Name, "Test Accesskey")
	}
	if a.Token != "test-token-12345" {
		t.Errorf("Accesskey.Token = %v, expected %v", a.Token, "test-token-12345")
	}
}
