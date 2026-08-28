package provider

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// Provider publishes on the global topic exchange (VOIP-1404/1419), so it must carry an explicit
// subscription address. The assertion pins the POINTER type: the event data reaches
// notifyhandler as a pointer and the interface assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Provider)(nil)

func TestProviderEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	p := &Provider{
		ID:       id,
		Type:     TypeSIP,
		Hostname: "sip.provider.com",
	}

	res := p.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}

	// Mutation check: a zero-id provider must resolve to the nil uuid, not to a stale or
	// invented address.
	empty := &Provider{}
	if resEmpty := empty.EventSubscriptionID(); resEmpty != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), resEmpty)
	}
}

func TestProviderStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	p := Provider{
		ID:          id,
		Type:        TypeSIP,
		Hostname:    "sip.provider.com",
		TechPrefix:  "+1",
		TechPostfix: "",
		TechHeaders: map[string]string{"X-Custom": "value"},
		Name:        "Primary Provider",
		Detail:      "Main SIP provider",
		Codecs:      "PCMU,PCMA",
	}

	if p.ID != id {
		t.Errorf("Provider.ID = %v, expected %v", p.ID, id)
	}
	if p.Type != TypeSIP {
		t.Errorf("Provider.Type = %v, expected %v", p.Type, TypeSIP)
	}
	if p.Hostname != "sip.provider.com" {
		t.Errorf("Provider.Hostname = %v, expected %v", p.Hostname, "sip.provider.com")
	}
	if p.TechPrefix != "+1" {
		t.Errorf("Provider.TechPrefix = %v, expected %v", p.TechPrefix, "+1")
	}
	if p.TechPostfix != "" {
		t.Errorf("Provider.TechPostfix = %v, expected %v", p.TechPostfix, "")
	}
	if len(p.TechHeaders) != 1 {
		t.Errorf("Provider.TechHeaders length = %v, expected %v", len(p.TechHeaders), 1)
	}
	if p.Name != "Primary Provider" {
		t.Errorf("Provider.Name = %v, expected %v", p.Name, "Primary Provider")
	}
	if p.Detail != "Main SIP provider" {
		t.Errorf("Provider.Detail = %v, expected %v", p.Detail, "Main SIP provider")
	}
	if p.Codecs != "PCMU,PCMA" {
		t.Errorf("Provider.Codecs = %v, expected %v", p.Codecs, "PCMU,PCMA")
	}
}

func TestTypeConstants(t *testing.T) {
	if TypeSIP != "sip" {
		t.Errorf("TypeSIP = %v, expected %v", TypeSIP, "sip")
	}
}
