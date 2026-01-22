package account

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestAccountStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	a := Account{
		ID:             id,
		CustomerID:     customerID,
		TotalFileCount: 100,
		TotalFileSize:  1073741824, // 1GB
		TMCreate:       "2023-01-01 00:00:00",
		TMUpdate:       "2023-01-02 00:00:00",
		TMDelete:       "",
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
	if a.TMCreate != "2023-01-01 00:00:00" {
		t.Errorf("Account.TMCreate = %v, expected %v", a.TMCreate, "2023-01-01 00:00:00")
	}
	if a.TMUpdate != "2023-01-02 00:00:00" {
		t.Errorf("Account.TMUpdate = %v, expected %v", a.TMUpdate, "2023-01-02 00:00:00")
	}
}
