package outplan

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestOutplanStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	o := Outplan{
		Name:         "Test Outplan",
		Detail:       "Test outplan details",
		DialTimeout:  30000,
		TryInterval:  60000,
		MaxTryCount0: 3,
		MaxTryCount1: 2,
		MaxTryCount2: 2,
		MaxTryCount3: 1,
		MaxTryCount4: 1,
		TMCreate:     "2024-01-01 00:00:00.000000",
		TMUpdate:     "2024-01-01 00:00:00.000000",
		TMDelete:     "9999-01-01 00:00:00.000000",
	}
	o.ID = id

	if o.ID != id {
		t.Errorf("Outplan.ID = %v, expected %v", o.ID, id)
	}
	if o.Name != "Test Outplan" {
		t.Errorf("Outplan.Name = %v, expected %v", o.Name, "Test Outplan")
	}
	if o.DialTimeout != 30000 {
		t.Errorf("Outplan.DialTimeout = %v, expected %v", o.DialTimeout, 30000)
	}
	if o.MaxTryCount0 != 3 {
		t.Errorf("Outplan.MaxTryCount0 = %v, expected %v", o.MaxTryCount0, 3)
	}
}

func TestMaxTryCountLenConstant(t *testing.T) {
	if MaxTryCountLen != 5 {
		t.Errorf("MaxTryCountLen = %v, expected %v", MaxTryCountLen, 5)
	}
}
