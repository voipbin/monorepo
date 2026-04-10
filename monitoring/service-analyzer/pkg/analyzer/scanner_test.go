package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"bin-call-manager", "call-manager"},
		{"bin-common-handler", "common-handler"},
		{"voip-asterisk-proxy", "asterisk-proxy"},
		{"voip-rtpengine-proxy", "rtpengine-proxy"},
		{"unknown-dir", "unknown-dir"},
	}
	for _, tt := range tests {
		got := extractServiceName(tt.input)
		if got != tt.want {
			t.Errorf("extractServiceName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveMethodTarget(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"CallV1CallCreate", "call-manager"},
		{"FlowV1FlowGet", "flow-manager"},
		{"CustomerV1CustomerDelete", "customer-manager"},
		{"AIV1AIGet", "ai-manager"},
		{"TTSV1TTSCreate", "tts-manager"},
		{"UnknownV1Something", ""},
		{"NoMatch", ""},
	}
	for _, tt := range tests {
		got := resolveMethodTarget(tt.method)
		if got != tt.want {
			t.Errorf("resolveMethodTarget(%q) = %q, want %q", tt.method, got, tt.want)
		}
	}
}

func TestAppendUnique(t *testing.T) {
	s := []string{"a", "b"}
	s = appendUnique(s, "c")
	if len(s) != 3 {
		t.Errorf("expected 3 elements, got %d", len(s))
	}
	s = appendUnique(s, "b")
	if len(s) != 3 {
		t.Errorf("expected 3 elements after duplicate, got %d", len(s))
	}
}

func TestScanFileForRPCCalls(t *testing.T) {
	// create a temp Go file with RPC calls
	dir := t.TempDir()
	content := `package handler

func (h *handler) doStuff(ctx context.Context) {
	res, err := h.reqHandler.CallV1CallCreate(ctx, id)
	res2, err2 := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	res3, err3 := h.reqHandler.CustomerV1CustomerDelete(ctx, custID)
}
`
	path := filepath.Join(dir, "handler.go")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	methods, err := scanFileForRPCCalls(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"CallV1CallCreate", "FlowV1FlowGet", "CustomerV1CustomerDelete"}
	if len(methods) != len(expected) {
		t.Errorf("expected %d methods, got %d: %v", len(expected), len(methods), methods)
		return
	}
	for i, m := range methods {
		if m != expected[i] {
			t.Errorf("method[%d] = %q, want %q", i, m, expected[i])
		}
	}
}

func TestDiscoverServices(t *testing.T) {
	// create a fake monorepo
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "bin-call-manager"), 0755)
	os.MkdirAll(filepath.Join(dir, "bin-flow-manager"), 0755)
	os.MkdirAll(filepath.Join(dir, "voip-asterisk-proxy"), 0755)
	os.MkdirAll(filepath.Join(dir, "monitoring"), 0755)   // should be ignored
	os.WriteFile(filepath.Join(dir, "README.md"), nil, 0644) // should be ignored

	scanner := NewScanner(dir)
	services, err := scanner.DiscoverServices()
	if err != nil {
		t.Fatal(err)
	}

	if len(services) != 3 {
		t.Errorf("expected 3 services, got %d", len(services))
	}

	names := make(map[string]bool)
	for _, s := range services {
		names[s.Name] = true
	}
	for _, expected := range []string{"call-manager", "flow-manager", "asterisk-proxy"} {
		if !names[expected] {
			t.Errorf("expected service %q not found", expected)
		}
	}
}

func TestComputeMetrics(t *testing.T) {
	g := &Graph{
		Services: []Service{
			{Name: "svc-a"},
			{Name: "svc-b"},
			{Name: "svc-c"},
		},
		Dependencies: []Dependency{
			{From: "svc-a", To: "svc-b", Type: DepRPC, Methods: []string{"MethodX"}},
			{From: "svc-a", To: "svc-c", Type: DepRPC, Methods: []string{"MethodY"}},
			{From: "svc-b", To: "svc-c", Type: DepEvent, Methods: []string{"evt_1"}},
		},
	}

	metrics := ComputeMetrics(g)
	if len(metrics) != 3 {
		t.Fatalf("expected 3 metrics entries, got %d", len(metrics))
	}

	metricsMap := make(map[string]ServiceMetrics)
	for _, m := range metrics {
		metricsMap[m.Name] = m
	}

	a := metricsMap["svc-a"]
	if a.RPCFanOut != 2 {
		t.Errorf("svc-a RPCFanOut = %d, want 2", a.RPCFanOut)
	}
	if a.RPCFanIn != 0 {
		t.Errorf("svc-a RPCFanIn = %d, want 0", a.RPCFanIn)
	}

	b := metricsMap["svc-b"]
	if b.RPCFanIn != 1 {
		t.Errorf("svc-b RPCFanIn = %d, want 1", b.RPCFanIn)
	}
	if b.EventConsumers != 1 {
		t.Errorf("svc-b EventConsumers = %d, want 1", b.EventConsumers)
	}

	c := metricsMap["svc-c"]
	if c.RPCFanIn != 1 {
		t.Errorf("svc-c RPCFanIn = %d, want 1", c.RPCFanIn)
	}
	if c.EventPublishers != 1 {
		t.Errorf("svc-c EventPublishers = %d, want 1", c.EventPublishers)
	}
}

func TestScanRPCDependencies(t *testing.T) {
	dir := t.TempDir()

	// create fake service with RPC calls
	svcDir := filepath.Join(dir, "bin-test-svc")
	pkgDir := filepath.Join(svcDir, "pkg", "handler")
	os.MkdirAll(pkgDir, 0755)

	content := `package handler

func (h *handler) work(ctx context.Context) {
	h.reqHandler.CallV1CallGet(ctx, id)
	h.reqHandler.FlowV1FlowCreate(ctx, data)
}
`
	os.WriteFile(filepath.Join(pkgDir, "handler.go"), []byte(content), 0644)

	// should not scan test files
	testContent := `package handler

func TestSomething(t *testing.T) {
	h.reqHandler.BillingV1BillingGet(ctx, id)
}
`
	os.WriteFile(filepath.Join(pkgDir, "handler_test.go"), []byte(testContent), 0644)

	scanner := NewScanner(dir)
	services := []Service{{Name: "test-svc", Directory: svcDir}}
	deps, err := scanner.ScanRPCDependencies(services)
	if err != nil {
		t.Fatal(err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d: %+v", len(deps), deps)
		return
	}

	targets := make(map[string]bool)
	for _, d := range deps {
		targets[d.To] = true
		if d.From != "test-svc" {
			t.Errorf("dep.From = %q, want 'test-svc'", d.From)
		}
		if d.Type != DepRPC {
			t.Errorf("dep.Type = %v, want DepRPC", d.Type)
		}
	}

	if !targets["call-manager"] {
		t.Error("expected dependency on call-manager")
	}
	if !targets["flow-manager"] {
		t.Error("expected dependency on flow-manager")
	}
	if targets["billing-manager"] {
		t.Error("billing-manager should not be found (only in test file)")
	}
}
