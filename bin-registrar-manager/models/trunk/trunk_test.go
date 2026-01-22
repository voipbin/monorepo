package trunk

import (
	"testing"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
)

func TestTrunkStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tr := Trunk{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:       "Test Trunk",
		Detail:     "Test trunk description",
		DomainName: "trunk.example.com",
		AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
		Realm:      "example.com",
		Username:   "trunk_user",
		Password:   "secret123",
		AllowedIPs: []string{"192.168.1.1", "10.0.0.1"},
		TMCreate:   "2023-01-01 00:00:00",
		TMUpdate:   "2023-01-02 00:00:00",
		TMDelete:   "",
	}

	if tr.ID != id {
		t.Errorf("Trunk.ID = %v, expected %v", tr.ID, id)
	}
	if tr.CustomerID != customerID {
		t.Errorf("Trunk.CustomerID = %v, expected %v", tr.CustomerID, customerID)
	}
	if tr.Name != "Test Trunk" {
		t.Errorf("Trunk.Name = %v, expected %v", tr.Name, "Test Trunk")
	}
	if tr.Detail != "Test trunk description" {
		t.Errorf("Trunk.Detail = %v, expected %v", tr.Detail, "Test trunk description")
	}
	if tr.DomainName != "trunk.example.com" {
		t.Errorf("Trunk.DomainName = %v, expected %v", tr.DomainName, "trunk.example.com")
	}
	if len(tr.AuthTypes) != 2 {
		t.Errorf("Trunk.AuthTypes length = %v, expected %v", len(tr.AuthTypes), 2)
	}
	if tr.Realm != "example.com" {
		t.Errorf("Trunk.Realm = %v, expected %v", tr.Realm, "example.com")
	}
	if tr.Username != "trunk_user" {
		t.Errorf("Trunk.Username = %v, expected %v", tr.Username, "trunk_user")
	}
	if tr.Password != "secret123" {
		t.Errorf("Trunk.Password = %v, expected %v", tr.Password, "secret123")
	}
	if len(tr.AllowedIPs) != 2 {
		t.Errorf("Trunk.AllowedIPs length = %v, expected %v", len(tr.AllowedIPs), 2)
	}
}

func TestTrunk_GenerateSIPAuth(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tr := Trunk{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
		Realm:      "test.realm.com",
		Username:   "testuser",
		Password:   "testpass",
		AllowedIPs: []string{"192.168.1.1"},
	}

	result := tr.GenerateSIPAuth()

	if result.ID != id {
		t.Errorf("GenerateSIPAuth().ID = %v, expected %v", result.ID, id)
	}
	if result.ReferenceType != sipauth.ReferenceTypeTrunk {
		t.Errorf("GenerateSIPAuth().ReferenceType = %v, expected %v", result.ReferenceType, sipauth.ReferenceTypeTrunk)
	}
	if len(result.AuthTypes) != 1 {
		t.Errorf("GenerateSIPAuth().AuthTypes length = %v, expected %v", len(result.AuthTypes), 1)
	}
	if result.Realm != "test.realm.com" {
		t.Errorf("GenerateSIPAuth().Realm = %v, expected %v", result.Realm, "test.realm.com")
	}
	if result.Username != "testuser" {
		t.Errorf("GenerateSIPAuth().Username = %v, expected %v", result.Username, "testuser")
	}
	if result.Password != "testpass" {
		t.Errorf("GenerateSIPAuth().Password = %v, expected %v", result.Password, "testpass")
	}
	if len(result.AllowedIPs) != 1 {
		t.Errorf("GenerateSIPAuth().AllowedIPs length = %v, expected %v", len(result.AllowedIPs), 1)
	}
}
