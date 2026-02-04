package accesskey

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestAccesskeyStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	a := Accesskey{
		ID:         id,
		CustomerID: customerID,
		Name:       "Test Accesskey",
		Detail:     "Test accesskey details",
		Token:      "test-token-12345",
		TMExpire:   "2025-01-01 00:00:00.000000",
		TMCreate:   "2024-01-01 00:00:00.000000",
		TMUpdate:   "2024-01-01 00:00:00.000000",
		TMDelete:   "9999-01-01T00:00:00.000000Z",
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
