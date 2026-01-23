package astaor

import (
	"testing"
)

func TestAstAORStruct(t *testing.T) {
	id := "aor_123"
	maxContacts := 5
	removeExisting := "yes"
	defaultExpiration := 3600
	minimumExpiration := 60
	maximumExpiration := 7200
	outboundProxy := "sip:proxy.example.com"
	supportPath := "yes"
	authenticateQualify := "no"
	qualifyFrequency := 60
	qualifyTimeout := float32(3.0)
	contact := "sip:user@example.com"
	mailboxes := "1000@default"
	voicemailExtension := "*97"

	aor := AstAOR{
		ID:                  &id,
		MaxContacts:         &maxContacts,
		RemoveExisting:      &removeExisting,
		DefaultExpiration:   &defaultExpiration,
		MinimumExpiration:   &minimumExpiration,
		MaximumExpiration:   &maximumExpiration,
		OutboundProxy:       &outboundProxy,
		SupportPath:         &supportPath,
		AuthenticateQualify: &authenticateQualify,
		QualifyFrequency:    &qualifyFrequency,
		QualifyTimeout:      &qualifyTimeout,
		Contact:             &contact,
		Mailboxes:           &mailboxes,
		VoicemailExtension:  &voicemailExtension,
	}

	if aor.ID == nil || *aor.ID != id {
		t.Errorf("AstAOR.ID = %v, expected %v", aor.ID, &id)
	}
	if aor.MaxContacts == nil || *aor.MaxContacts != maxContacts {
		t.Errorf("AstAOR.MaxContacts = %v, expected %v", aor.MaxContacts, &maxContacts)
	}
	if aor.RemoveExisting == nil || *aor.RemoveExisting != removeExisting {
		t.Errorf("AstAOR.RemoveExisting = %v, expected %v", aor.RemoveExisting, &removeExisting)
	}
	if aor.DefaultExpiration == nil || *aor.DefaultExpiration != defaultExpiration {
		t.Errorf("AstAOR.DefaultExpiration = %v, expected %v", aor.DefaultExpiration, &defaultExpiration)
	}
	if aor.MinimumExpiration == nil || *aor.MinimumExpiration != minimumExpiration {
		t.Errorf("AstAOR.MinimumExpiration = %v, expected %v", aor.MinimumExpiration, &minimumExpiration)
	}
	if aor.MaximumExpiration == nil || *aor.MaximumExpiration != maximumExpiration {
		t.Errorf("AstAOR.MaximumExpiration = %v, expected %v", aor.MaximumExpiration, &maximumExpiration)
	}
	if aor.OutboundProxy == nil || *aor.OutboundProxy != outboundProxy {
		t.Errorf("AstAOR.OutboundProxy = %v, expected %v", aor.OutboundProxy, &outboundProxy)
	}
	if aor.SupportPath == nil || *aor.SupportPath != supportPath {
		t.Errorf("AstAOR.SupportPath = %v, expected %v", aor.SupportPath, &supportPath)
	}
	if aor.AuthenticateQualify == nil || *aor.AuthenticateQualify != authenticateQualify {
		t.Errorf("AstAOR.AuthenticateQualify = %v, expected %v", aor.AuthenticateQualify, &authenticateQualify)
	}
	if aor.QualifyFrequency == nil || *aor.QualifyFrequency != qualifyFrequency {
		t.Errorf("AstAOR.QualifyFrequency = %v, expected %v", aor.QualifyFrequency, &qualifyFrequency)
	}
	if aor.QualifyTimeout == nil || *aor.QualifyTimeout != qualifyTimeout {
		t.Errorf("AstAOR.QualifyTimeout = %v, expected %v", aor.QualifyTimeout, &qualifyTimeout)
	}
	if aor.Contact == nil || *aor.Contact != contact {
		t.Errorf("AstAOR.Contact = %v, expected %v", aor.Contact, &contact)
	}
	if aor.Mailboxes == nil || *aor.Mailboxes != mailboxes {
		t.Errorf("AstAOR.Mailboxes = %v, expected %v", aor.Mailboxes, &mailboxes)
	}
	if aor.VoicemailExtension == nil || *aor.VoicemailExtension != voicemailExtension {
		t.Errorf("AstAOR.VoicemailExtension = %v, expected %v", aor.VoicemailExtension, &voicemailExtension)
	}
}

func TestAstAORStructWithNilFields(t *testing.T) {
	aor := AstAOR{}

	if aor.ID != nil {
		t.Errorf("AstAOR.ID should be nil, got %v", aor.ID)
	}
	if aor.MaxContacts != nil {
		t.Errorf("AstAOR.MaxContacts should be nil, got %v", aor.MaxContacts)
	}
	if aor.RemoveExisting != nil {
		t.Errorf("AstAOR.RemoveExisting should be nil, got %v", aor.RemoveExisting)
	}
	if aor.DefaultExpiration != nil {
		t.Errorf("AstAOR.DefaultExpiration should be nil, got %v", aor.DefaultExpiration)
	}
}
