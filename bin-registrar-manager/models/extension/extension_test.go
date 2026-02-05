package extension

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
)

func TestExtensionStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	ext := Extension{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Name:       "Test Extension",
		Detail:     "Test extension description",
		EndpointID: "endpoint_123",
		AORID:      "aor_123",
		AuthID:     "auth_123",
		Extension:  "1001",
		DomainName: "ext.example.com",
		Realm:      "example.com",
		Username:   "ext_user",
		Password:   "secret123",
		TMCreate:   func() *time.Time { t := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
		TMUpdate:   func() *time.Time { t := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC); return &t }(),
		TMDelete:   nil,
	}

	if ext.ID != id {
		t.Errorf("Extension.ID = %v, expected %v", ext.ID, id)
	}
	if ext.CustomerID != customerID {
		t.Errorf("Extension.CustomerID = %v, expected %v", ext.CustomerID, customerID)
	}
	if ext.Name != "Test Extension" {
		t.Errorf("Extension.Name = %v, expected %v", ext.Name, "Test Extension")
	}
	if ext.Detail != "Test extension description" {
		t.Errorf("Extension.Detail = %v, expected %v", ext.Detail, "Test extension description")
	}
	if ext.EndpointID != "endpoint_123" {
		t.Errorf("Extension.EndpointID = %v, expected %v", ext.EndpointID, "endpoint_123")
	}
	if ext.AORID != "aor_123" {
		t.Errorf("Extension.AORID = %v, expected %v", ext.AORID, "aor_123")
	}
	if ext.AuthID != "auth_123" {
		t.Errorf("Extension.AuthID = %v, expected %v", ext.AuthID, "auth_123")
	}
	if ext.Extension != "1001" {
		t.Errorf("Extension.Extension = %v, expected %v", ext.Extension, "1001")
	}
	if ext.DomainName != "ext.example.com" {
		t.Errorf("Extension.DomainName = %v, expected %v", ext.DomainName, "ext.example.com")
	}
	if ext.Realm != "example.com" {
		t.Errorf("Extension.Realm = %v, expected %v", ext.Realm, "example.com")
	}
	if ext.Username != "ext_user" {
		t.Errorf("Extension.Username = %v, expected %v", ext.Username, "ext_user")
	}
	if ext.Password != "secret123" {
		t.Errorf("Extension.Password = %v, expected %v", ext.Password, "secret123")
	}
}

func TestExtension_GenerateSIPAuth(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	ext := Extension{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Realm:    "test.realm.com",
		Username: "testuser",
		Password: "testpass",
	}

	result := ext.GenerateSIPAuth()

	if result.ID != id {
		t.Errorf("GenerateSIPAuth().ID = %v, expected %v", result.ID, id)
	}
	if result.ReferenceType != sipauth.ReferenceTypeExtension {
		t.Errorf("GenerateSIPAuth().ReferenceType = %v, expected %v", result.ReferenceType, sipauth.ReferenceTypeExtension)
	}
	if len(result.AuthTypes) != 1 {
		t.Errorf("GenerateSIPAuth().AuthTypes length = %v, expected %v", len(result.AuthTypes), 1)
	}
	if result.AuthTypes[0] != sipauth.AuthTypeBasic {
		t.Errorf("GenerateSIPAuth().AuthTypes[0] = %v, expected %v", result.AuthTypes[0], sipauth.AuthTypeBasic)
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
	if len(result.AllowedIPs) != 0 {
		t.Errorf("GenerateSIPAuth().AllowedIPs length = %v, expected %v", len(result.AllowedIPs), 0)
	}
}
