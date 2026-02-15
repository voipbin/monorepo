package customer

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	billingAccountID := uuid.Must(uuid.NewV4())

	fs := FieldStruct{
		ID:               id,
		Name:             "test-name",
		Email:            "test@example.com",
		PhoneNumber:      "+1234567890",
		BillingAccountID: billingAccountID,
		Deleted:          false,
	}

	if fs.ID != id {
		t.Errorf("FieldStruct.ID = %v, expected %v", fs.ID, id)
	}
	if fs.Name != "test-name" {
		t.Errorf("FieldStruct.Name = %v, expected %v", fs.Name, "test-name")
	}
	if fs.Email != "test@example.com" {
		t.Errorf("FieldStruct.Email = %v, expected %v", fs.Email, "test@example.com")
	}
	if fs.PhoneNumber != "+1234567890" {
		t.Errorf("FieldStruct.PhoneNumber = %v, expected %v", fs.PhoneNumber, "+1234567890")
	}
	if fs.BillingAccountID != billingAccountID {
		t.Errorf("FieldStruct.BillingAccountID = %v, expected %v", fs.BillingAccountID, billingAccountID)
	}
	if fs.Deleted != false {
		t.Errorf("FieldStruct.Deleted = %v, expected %v", fs.Deleted, false)
	}
}
