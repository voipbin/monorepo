package astendpoint

import (
	"testing"
)

func TestAstEndpointStruct(t *testing.T) {
	id := "endpoint_123"
	transport := "transport-tcp"
	aors := "aor_123"
	auth := "auth_123"
	context := "default"
	identifyBy := "username"
	fromDomain := "example.com"

	ep := AstEndpoint{
		ID:         &id,
		Transport:  &transport,
		AORs:       &aors,
		Auth:       &auth,
		Context:    &context,
		IdentifyBy: &identifyBy,
		FromDomain: &fromDomain,
	}

	if ep.ID == nil || *ep.ID != id {
		t.Errorf("AstEndpoint.ID = %v, expected %v", ep.ID, &id)
	}
	if ep.Transport == nil || *ep.Transport != transport {
		t.Errorf("AstEndpoint.Transport = %v, expected %v", ep.Transport, &transport)
	}
	if ep.AORs == nil || *ep.AORs != aors {
		t.Errorf("AstEndpoint.AORs = %v, expected %v", ep.AORs, &aors)
	}
	if ep.Auth == nil || *ep.Auth != auth {
		t.Errorf("AstEndpoint.Auth = %v, expected %v", ep.Auth, &auth)
	}
	if ep.Context == nil || *ep.Context != context {
		t.Errorf("AstEndpoint.Context = %v, expected %v", ep.Context, &context)
	}
	if ep.IdentifyBy == nil || *ep.IdentifyBy != identifyBy {
		t.Errorf("AstEndpoint.IdentifyBy = %v, expected %v", ep.IdentifyBy, &identifyBy)
	}
	if ep.FromDomain == nil || *ep.FromDomain != fromDomain {
		t.Errorf("AstEndpoint.FromDomain = %v, expected %v", ep.FromDomain, &fromDomain)
	}
}

func TestAstEndpointStructWithNilFields(t *testing.T) {
	ep := AstEndpoint{}

	if ep.ID != nil {
		t.Errorf("AstEndpoint.ID should be nil, got %v", ep.ID)
	}
	if ep.Transport != nil {
		t.Errorf("AstEndpoint.Transport should be nil, got %v", ep.Transport)
	}
	if ep.AORs != nil {
		t.Errorf("AstEndpoint.AORs should be nil, got %v", ep.AORs)
	}
	if ep.Auth != nil {
		t.Errorf("AstEndpoint.Auth should be nil, got %v", ep.Auth)
	}
	if ep.Context != nil {
		t.Errorf("AstEndpoint.Context should be nil, got %v", ep.Context)
	}
	if ep.IdentifyBy != nil {
		t.Errorf("AstEndpoint.IdentifyBy should be nil, got %v", ep.IdentifyBy)
	}
	if ep.FromDomain != nil {
		t.Errorf("AstEndpoint.FromDomain should be nil, got %v", ep.FromDomain)
	}
}
