package accesskey

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	fs := FieldStruct{
		ID:         id,
		CustomerID: customerID,
		Name:       "test-name",
		Token:      "test-token",
		Deleted:    false,
	}

	if fs.ID != id {
		t.Errorf("FieldStruct.ID = %v, expected %v", fs.ID, id)
	}
	if fs.CustomerID != customerID {
		t.Errorf("FieldStruct.CustomerID = %v, expected %v", fs.CustomerID, customerID)
	}
	if fs.Name != "test-name" {
		t.Errorf("FieldStruct.Name = %v, expected %v", fs.Name, "test-name")
	}
	if fs.Token != "test-token" {
		t.Errorf("FieldStruct.Token = %v, expected %v", fs.Token, "test-token")
	}
	if fs.Deleted != false {
		t.Errorf("FieldStruct.Deleted = %v, expected %v", fs.Deleted, false)
	}
}
