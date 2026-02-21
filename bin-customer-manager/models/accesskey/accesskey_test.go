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
		ID:          id,
		CustomerID:  customerID,
		Name:        "Test Accesskey",
		Detail:      "Test accesskey details",
		TokenHash:   "abc123hash",
		TokenPrefix: "vb_test1234",
		RawToken:    "vb_test1234fulltoken",
		TMExpire:    &tmExpire,
		TMCreate:    &tmCreate,
		TMUpdate:    &tmUpdate,
		TMDelete:    nil,
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
	if a.TokenHash != "abc123hash" {
		t.Errorf("Accesskey.TokenHash = %v, expected %v", a.TokenHash, "abc123hash")
	}
	if a.TokenPrefix != "vb_test1234" {
		t.Errorf("Accesskey.TokenPrefix = %v, expected %v", a.TokenPrefix, "vb_test1234")
	}
	if a.RawToken != "vb_test1234fulltoken" {
		t.Errorf("Accesskey.RawToken = %v, expected %v", a.RawToken, "vb_test1234fulltoken")
	}
}
