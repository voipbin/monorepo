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

func TestScanRPCDependencies_NoPkgDir(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-empty-svc")
	os.MkdirAll(svcDir, 0755) // no pkg/ subdirectory

	scanner := NewScanner(dir)
	services := []Service{{Name: "empty-svc", Directory: svcDir}}
	deps, err := scanner.ScanRPCDependencies(services)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps for service without pkg/, got %d", len(deps))
	}
}

func TestScanRPCDependencies_SelfCallIgnored(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-call-manager")
	pkgDir := filepath.Join(svcDir, "pkg", "handler")
	os.MkdirAll(pkgDir, 0755)

	// call-manager calling CallV1* should be ignored (self-reference)
	content := `package handler

func (h *handler) work(ctx context.Context) {
	h.reqHandler.CallV1CallGet(ctx, id)
	h.reqHandler.FlowV1FlowCreate(ctx, data)
}
`
	os.WriteFile(filepath.Join(pkgDir, "handler.go"), []byte(content), 0644)

	scanner := NewScanner(dir)
	services := []Service{{Name: "call-manager", Directory: svcDir}}
	deps, err := scanner.ScanRPCDependencies(services)
	if err != nil {
		t.Fatal(err)
	}

	// Should only have flow-manager (self-call to call-manager skipped)
	if len(deps) != 1 {
		t.Errorf("expected 1 dep (self-call filtered), got %d: %+v", len(deps), deps)
		return
	}
	if deps[0].To != "flow-manager" {
		t.Errorf("expected dep to flow-manager, got %q", deps[0].To)
	}
}

func TestScanRPCDependencies_MockFileIgnored(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-test-svc")
	pkgDir := filepath.Join(svcDir, "pkg", "handler")
	os.MkdirAll(pkgDir, 0755)

	mockContent := `package handler

func (m *mockHandler) work(ctx context.Context) {
	m.reqHandler.BillingV1BillingGet(ctx, id)
}
`
	os.WriteFile(filepath.Join(pkgDir, "mock_handler.go"), []byte(mockContent), 0644)

	scanner := NewScanner(dir)
	services := []Service{{Name: "test-svc", Directory: svcDir}}
	deps, err := scanner.ScanRPCDependencies(services)
	if err != nil {
		t.Fatal(err)
	}

	if len(deps) != 0 {
		t.Errorf("expected 0 deps (mock files ignored), got %d", len(deps))
	}
}

func TestScanEventDependencies(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-flow-manager")
	subDir := filepath.Join(svcDir, "pkg", "subscribehandler")
	os.MkdirAll(subDir, 0755)

	content := `package subscribehandler

import (
	"monorepo/bin-common-handler/pkg/commonoutline"
)

func (h *handler) processEvent(ctx context.Context, m *commonoutline.Message) {
	if m.Publisher == string(commonoutline.ServiceNameCallManager) {
		// handle call manager events
	}
	if m.Publisher == string(commonoutline.ServiceNameCustomerManager) {
		// handle customer manager events
	}
}
`
	os.WriteFile(filepath.Join(subDir, "handler.go"), []byte(content), 0644)

	scanner := NewScanner(dir)
	services := []Service{{Name: "flow-manager", Directory: svcDir}}
	deps, err := scanner.ScanEventDependencies(services)
	if err != nil {
		t.Fatal(err)
	}

	if len(deps) < 2 {
		t.Errorf("expected at least 2 event deps, got %d: %+v", len(deps), deps)
		return
	}

	targets := make(map[string]bool)
	for _, d := range deps {
		targets[d.To] = true
		if d.From != "flow-manager" {
			t.Errorf("dep.From = %q, want 'flow-manager'", d.From)
		}
		if d.Type != DepEvent {
			t.Errorf("dep.Type = %v, want DepEvent", d.Type)
		}
	}

	if !targets["call-manager"] {
		t.Error("expected event dep on call-manager")
	}
	if !targets["customer-manager"] {
		t.Error("expected event dep on customer-manager")
	}
}

func TestScanEventDependencies_NoSubscribeDir(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-test-svc")
	os.MkdirAll(filepath.Join(svcDir, "pkg"), 0755) // no subscribehandler/

	scanner := NewScanner(dir)
	services := []Service{{Name: "test-svc", Directory: svcDir}}
	deps, err := scanner.ScanEventDependencies(services)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 event deps without subscribehandler, got %d", len(deps))
	}
}

func TestScanEventDependencies_EventTypePattern(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-agent-manager")
	subDir := filepath.Join(svcDir, "pkg", "subscribehandler")
	os.MkdirAll(subDir, 0755)

	content := `package subscribehandler

import (
	cmcall "monorepo/bin-call-manager/pkg/cmcall"
)

func (h *handler) dispatchEvent(evtType string) {
	switch evtType {
	case string(cmcall.EventTypeCallCreated):
		h.handleCallCreated()
	case string(cmcall.EventTypeCallDeleted):
		h.handleCallDeleted()
	}
}
`
	os.WriteFile(filepath.Join(subDir, "handler.go"), []byte(content), 0644)

	scanner := NewScanner(dir)
	services := []Service{{Name: "agent-manager", Directory: svcDir}}
	deps, err := scanner.ScanEventDependencies(services)
	if err != nil {
		t.Fatal(err)
	}

	foundCallManager := false
	for _, d := range deps {
		if d.To == "call-manager" {
			foundCallManager = true
			// Should have detected specific event types
			if len(d.Methods) == 0 {
				t.Error("expected event type methods for call-manager dep")
			}
		}
	}
	if !foundCallManager {
		t.Error("expected event dep on call-manager from EventType pattern")
	}
}

func TestScanEventDependencies_SelfRefIgnored(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bin-call-manager")
	subDir := filepath.Join(svcDir, "pkg", "subscribehandler")
	os.MkdirAll(subDir, 0755)

	content := `package subscribehandler

func (h *handler) processEvent(ctx context.Context) {
	if m.Publisher == string(commonoutline.ServiceNameCallManager) {
		// self-reference should be ignored
	}
}
`
	os.WriteFile(filepath.Join(subDir, "handler.go"), []byte(content), 0644)

	scanner := NewScanner(dir)
	services := []Service{{Name: "call-manager", Directory: svcDir}}
	deps, err := scanner.ScanEventDependencies(services)
	if err != nil {
		t.Fatal(err)
	}

	if len(deps) != 0 {
		t.Errorf("expected 0 deps (self-ref ignored), got %d: %+v", len(deps), deps)
	}
}

func TestBuildGraph(t *testing.T) {
	dir := t.TempDir()

	// create two services
	svcADir := filepath.Join(dir, "bin-svc-a")
	svcBDir := filepath.Join(dir, "bin-svc-b")
	pkgA := filepath.Join(svcADir, "pkg", "handler")
	os.MkdirAll(pkgA, 0755)
	os.MkdirAll(svcBDir, 0755) // svc-b has no pkg/

	content := `package handler

func (h *handler) work(ctx context.Context) {
	h.reqHandler.CallV1CallGet(ctx, id)
}
`
	os.WriteFile(filepath.Join(pkgA, "handler.go"), []byte(content), 0644)

	scanner := NewScanner(dir)
	g, err := scanner.BuildGraph()
	if err != nil {
		t.Fatal(err)
	}

	if len(g.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(g.Services))
	}
	if len(g.Dependencies) == 0 {
		t.Error("expected at least 1 dependency")
	}
}

func TestDiscoverServices_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	scanner := NewScanner(dir)
	services, err := scanner.DiscoverServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services in empty dir, got %d", len(services))
	}
}

func TestDiscoverServices_InvalidDir(t *testing.T) {
	scanner := NewScanner("/nonexistent/path")
	_, err := scanner.DiscoverServices()
	if err == nil {
		t.Error("expected error for invalid directory")
	}
}

func TestScanFileForRPCCalls_AllPatterns(t *testing.T) {
	dir := t.TempDir()
	content := `package handler

func (h *handler) work(ctx context.Context) {
	h.reqHandler.CallV1CallGet(ctx, id)
	h.requestHandler.FlowV1FlowCreate(ctx, data)
	res := h.reqHandler.TTSV1TTSCreate(ctx, req)
	res2 := h.requestHandler.RTPEngineV1Allocate(ctx, req)
}
`
	path := filepath.Join(dir, "handler.go")
	os.WriteFile(path, []byte(content), 0644)

	methods, err := scanFileForRPCCalls(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(methods) != 4 {
		t.Errorf("expected 4 methods, got %d: %v", len(methods), methods)
	}
}

func TestScanFileForRPCCalls_NonexistentFile(t *testing.T) {
	_, err := scanFileForRPCCalls("/nonexistent/file.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestResolveMethodTarget_AllPrefixes(t *testing.T) {
	// verify each entry in methodPrefixToService resolves correctly
	for prefix, expectedService := range methodPrefixToService {
		method := prefix + "V1SomeAction"
		got := resolveMethodTarget(method)
		if got != expectedService {
			t.Errorf("resolveMethodTarget(%q) = %q, want %q", method, got, expectedService)
		}
	}
}
