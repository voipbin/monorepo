package sipauth

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestSIPAuthStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	sa := SIPAuth{
		ID:            id,
		ReferenceType: ReferenceTypeTrunk,
		AuthTypes:     []AuthType{AuthTypeBasic, AuthTypeIP},
		Realm:         "example.com",
		Username:      "testuser",
		Password:      "testpass",
		AllowedIPs:    []string{"192.168.1.1", "10.0.0.1"},
		TMCreate:      "2023-01-01 00:00:00",
		TMUpdate:      "2023-01-02 00:00:00",
	}

	if sa.ID != id {
		t.Errorf("SIPAuth.ID = %v, expected %v", sa.ID, id)
	}
	if sa.ReferenceType != ReferenceTypeTrunk {
		t.Errorf("SIPAuth.ReferenceType = %v, expected %v", sa.ReferenceType, ReferenceTypeTrunk)
	}
	if len(sa.AuthTypes) != 2 {
		t.Errorf("SIPAuth.AuthTypes length = %v, expected %v", len(sa.AuthTypes), 2)
	}
	if sa.Realm != "example.com" {
		t.Errorf("SIPAuth.Realm = %v, expected %v", sa.Realm, "example.com")
	}
	if sa.Username != "testuser" {
		t.Errorf("SIPAuth.Username = %v, expected %v", sa.Username, "testuser")
	}
	if sa.Password != "testpass" {
		t.Errorf("SIPAuth.Password = %v, expected %v", sa.Password, "testpass")
	}
	if len(sa.AllowedIPs) != 2 {
		t.Errorf("SIPAuth.AllowedIPs length = %v, expected %v", len(sa.AllowedIPs), 2)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_trunk", ReferenceTypeTrunk, "trunk"},
		{"reference_type_extension", ReferenceTypeExtension, "extension"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestAuthTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant AuthType
		expected string
	}{
		{"auth_type_basic", AuthTypeBasic, "basic"},
		{"auth_type_ip", AuthTypeIP, "ip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
