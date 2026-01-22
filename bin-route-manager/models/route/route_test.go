package route

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestRouteStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	providerID := uuid.Must(uuid.NewV4())

	r := Route{
		ID:         id,
		CustomerID: customerID,
		Name:       "Default Route",
		Detail:     "Default routing for all destinations",
		ProviderID: providerID,
		Priority:   1,
		Target:     TargetAll,
	}

	if r.ID != id {
		t.Errorf("Route.ID = %v, expected %v", r.ID, id)
	}
	if r.CustomerID != customerID {
		t.Errorf("Route.CustomerID = %v, expected %v", r.CustomerID, customerID)
	}
	if r.Name != "Default Route" {
		t.Errorf("Route.Name = %v, expected %v", r.Name, "Default Route")
	}
	if r.Detail != "Default routing for all destinations" {
		t.Errorf("Route.Detail = %v, expected %v", r.Detail, "Default routing for all destinations")
	}
	if r.ProviderID != providerID {
		t.Errorf("Route.ProviderID = %v, expected %v", r.ProviderID, providerID)
	}
	if r.Priority != 1 {
		t.Errorf("Route.Priority = %v, expected %v", r.Priority, 1)
	}
	if r.Target != TargetAll {
		t.Errorf("Route.Target = %v, expected %v", r.Target, TargetAll)
	}
}

func TestTargetConstants(t *testing.T) {
	if TargetAll != "all" {
		t.Errorf("TargetAll = %v, expected %v", TargetAll, "all")
	}
}

func TestCustomerIDBasicRoute(t *testing.T) {
	expected := "00000000-0000-0000-0000-000000000001"
	if CustomerIDBasicRoute.String() != expected {
		t.Errorf("CustomerIDBasicRoute = %v, expected %v", CustomerIDBasicRoute.String(), expected)
	}
}
