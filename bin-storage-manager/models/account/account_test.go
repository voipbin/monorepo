package account

import (
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
