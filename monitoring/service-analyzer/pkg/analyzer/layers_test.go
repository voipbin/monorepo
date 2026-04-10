package analyzer

import (
	"testing"
)

func TestDetectLayerViolations_NoViolations(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "call-manager"},
			{Name: "flow-manager"},
			{Name: "storage-manager"},
		},
		Dependencies: []Dependency{
			// Core -> Integration (allowed)
			{From: "call-manager", To: "storage-manager", Type: DepRPC},
			// Core -> Core (allowed)
			{From: "call-manager", To: "flow-manager", Type: DepRPC},
		},
	}

	layerMap := map[string]Layer{
		"call-manager":    LayerCore,
		"flow-manager":    LayerCore,
		"storage-manager": LayerIntegration,
	}

	violations := DetectLayerViolations(g, layerMap)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d: %+v", len(violations), violations)
	}
}

func TestDetectLayerViolations_ProxyToBusinessViolation(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "asterisk-proxy"},
			{Name: "campaign-manager"},
		},
		Dependencies: []Dependency{
			// Proxy -> Business (not allowed)
			{From: "asterisk-proxy", To: "campaign-manager", Type: DepRPC},
		},
	}

	layerMap := map[string]Layer{
		"asterisk-proxy":   LayerProxy,
		"campaign-manager": LayerBusiness,
	}

	violations := DetectLayerViolations(g, layerMap)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	v := violations[0]
	if v.From != "asterisk-proxy" {
		t.Errorf("violation From = %q, want 'asterisk-proxy'", v.From)
	}
	if v.FromLayer != LayerProxy {
		t.Errorf("violation FromLayer = %q, want Proxy", v.FromLayer)
	}
	if v.ToLayer != LayerBusiness {
		t.Errorf("violation ToLayer = %q, want Business", v.ToLayer)
	}
}

func TestDetectLayerViolations_IntegrationToBusiness(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "storage-manager"},
			{Name: "billing-manager"},
		},
		Dependencies: []Dependency{
			// Integration -> Business (not allowed)
			{From: "storage-manager", To: "billing-manager", Type: DepRPC},
		},
	}

	layerMap := map[string]Layer{
		"storage-manager": LayerIntegration,
		"billing-manager": LayerBusiness,
	}

	violations := DetectLayerViolations(g, layerMap)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

func TestDetectLayerViolations_UnknownServiceSkipped(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "unknown-svc"},
			{Name: "call-manager"},
		},
		Dependencies: []Dependency{
			{From: "unknown-svc", To: "call-manager", Type: DepRPC},
		},
	}

	layerMap := map[string]Layer{
		"call-manager": LayerCore,
		// unknown-svc not in layerMap
	}

	violations := DetectLayerViolations(g, layerMap)
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for unknown service, got %d", len(violations))
	}
}

func TestDetectLayerViolations_EventDepsCheckedToo(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "asterisk-proxy"},
			{Name: "agent-manager"},
		},
		Dependencies: []Dependency{
			// Proxy -> Business via event (still a violation)
			{From: "asterisk-proxy", To: "agent-manager", Type: DepEvent},
		},
	}

	layerMap := map[string]Layer{
		"asterisk-proxy": LayerProxy,
		"agent-manager":  LayerBusiness,
	}

	violations := DetectLayerViolations(g, layerMap)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for event dep, got %d", len(violations))
	}
	if violations[0].DepType != DepEvent {
		t.Errorf("violation dep type = %q, want 'event'", violations[0].DepType)
	}
}

func TestDetectLayerViolations_MultipleViolations(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "asterisk-proxy"},
			{Name: "campaign-manager"},
			{Name: "email-manager"},
		},
		Dependencies: []Dependency{
			{From: "asterisk-proxy", To: "campaign-manager", Type: DepRPC},
			{From: "asterisk-proxy", To: "email-manager", Type: DepRPC},
		},
	}

	layerMap := map[string]Layer{
		"asterisk-proxy":   LayerProxy,
		"campaign-manager": LayerBusiness,
		"email-manager":    LayerMessaging,
	}

	violations := DetectLayerViolations(g, layerMap)
	if len(violations) != 2 {
		t.Errorf("expected 2 violations, got %d", len(violations))
	}
}

func TestIsLayerAllowed(t *testing.T) {
	tests := []struct {
		from Layer
		to   Layer
		want bool
	}{
		{LayerCore, LayerCore, true},
		{LayerCore, LayerIntegration, true},
		{LayerProxy, LayerCore, true},
		{LayerProxy, LayerBusiness, false},
		{LayerProxy, LayerMessaging, false},
		{LayerIntegration, LayerBusiness, false},
		{LayerIntegration, LayerCore, true},
		{LayerGateway, LayerCore, true},
		{LayerGateway, LayerBusiness, true},
		{LayerBusiness, LayerCore, true},
		{LayerBusiness, LayerTelephony, true},
		{LayerMessaging, LayerBusiness, false},
		{LayerTooling, LayerCore, true},
	}

	for _, tt := range tests {
		got := isLayerAllowed(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("isLayerAllowed(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestLayerViolation_String(t *testing.T) {
	v := LayerViolation{
		From:      "asterisk-proxy",
		FromLayer: LayerProxy,
		To:        "campaign-manager",
		ToLayer:   LayerBusiness,
		DepType:   DepRPC,
	}

	s := v.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
	if len(s) < 10 {
		t.Errorf("String() too short: %q", s)
	}
}
