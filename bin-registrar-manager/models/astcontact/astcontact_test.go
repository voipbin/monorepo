package astcontact

import (
	"testing"
)

func TestAstContactStruct(t *testing.T) {
	contact := AstContact{
		ID:                  "contact_123",
		URI:                 "sip:user@192.168.1.100:5060",
		ExpirationTime:      1704067200,
		QualifyFrequency:    60,
		OutboundProxy:       "sip:proxy.example.com",
		Path:                "<sip:path@example.com>",
		UserAgent:           "Obi/5.0",
		QualifyTimeout:      3.0,
		RegServer:           "asterisk.example.com",
		AuthenticateQualify: "no",
		ViaAddr:             "192.168.1.100",
		ViaPort:             5060,
		CallID:              "abc123@192.168.1.100",
		Endpoint:            "endpoint_123",
		PruneOnBoot:         "yes",
	}

	if contact.ID != "contact_123" {
		t.Errorf("AstContact.ID = %v, expected %v", contact.ID, "contact_123")
	}
	if contact.URI != "sip:user@192.168.1.100:5060" {
		t.Errorf("AstContact.URI = %v, expected %v", contact.URI, "sip:user@192.168.1.100:5060")
	}
	if contact.ExpirationTime != 1704067200 {
		t.Errorf("AstContact.ExpirationTime = %v, expected %v", contact.ExpirationTime, 1704067200)
	}
	if contact.QualifyFrequency != 60 {
		t.Errorf("AstContact.QualifyFrequency = %v, expected %v", contact.QualifyFrequency, 60)
	}
	if contact.OutboundProxy != "sip:proxy.example.com" {
		t.Errorf("AstContact.OutboundProxy = %v, expected %v", contact.OutboundProxy, "sip:proxy.example.com")
	}
	if contact.Path != "<sip:path@example.com>" {
		t.Errorf("AstContact.Path = %v, expected %v", contact.Path, "<sip:path@example.com>")
	}
	if contact.UserAgent != "Obi/5.0" {
		t.Errorf("AstContact.UserAgent = %v, expected %v", contact.UserAgent, "Obi/5.0")
	}
	if contact.QualifyTimeout != 3.0 {
		t.Errorf("AstContact.QualifyTimeout = %v, expected %v", contact.QualifyTimeout, 3.0)
	}
	if contact.RegServer != "asterisk.example.com" {
		t.Errorf("AstContact.RegServer = %v, expected %v", contact.RegServer, "asterisk.example.com")
	}
	if contact.AuthenticateQualify != "no" {
		t.Errorf("AstContact.AuthenticateQualify = %v, expected %v", contact.AuthenticateQualify, "no")
	}
	if contact.ViaAddr != "192.168.1.100" {
		t.Errorf("AstContact.ViaAddr = %v, expected %v", contact.ViaAddr, "192.168.1.100")
	}
	if contact.ViaPort != 5060 {
		t.Errorf("AstContact.ViaPort = %v, expected %v", contact.ViaPort, 5060)
	}
	if contact.CallID != "abc123@192.168.1.100" {
		t.Errorf("AstContact.CallID = %v, expected %v", contact.CallID, "abc123@192.168.1.100")
	}
	if contact.Endpoint != "endpoint_123" {
		t.Errorf("AstContact.Endpoint = %v, expected %v", contact.Endpoint, "endpoint_123")
	}
	if contact.PruneOnBoot != "yes" {
		t.Errorf("AstContact.PruneOnBoot = %v, expected %v", contact.PruneOnBoot, "yes")
	}
}

func TestAstContactStructEmpty(t *testing.T) {
	contact := AstContact{}

	if contact.ID != "" {
		t.Errorf("AstContact.ID should be empty, got %v", contact.ID)
	}
	if contact.URI != "" {
		t.Errorf("AstContact.URI should be empty, got %v", contact.URI)
	}
	if contact.ExpirationTime != 0 {
		t.Errorf("AstContact.ExpirationTime should be 0, got %v", contact.ExpirationTime)
	}
	if contact.ViaPort != 0 {
		t.Errorf("AstContact.ViaPort should be 0, got %v", contact.ViaPort)
	}
}
