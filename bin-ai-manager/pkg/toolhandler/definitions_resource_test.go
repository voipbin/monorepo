package toolhandler

import (
	"sort"
	"testing"

	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
)

// TestGetResourceEnumMatchesFetchers locks the get_resource JSON-schema enum
// in definitions.go to the aicallhandler fetcher table so the two cannot
// drift when Phase 2 adds resource types.
func TestGetResourceEnumMatchesFetchers(t *testing.T) {
	// find the get_resource definition
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameGetResource {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("get_resource tool definition not found in definitions.go")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource parameters has no properties map")
	}
	rt, ok := props["resource_type"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource has no resource_type property")
	}
	enum, ok := rt["enum"].([]string)
	if !ok {
		t.Fatalf("resource_type enum is not []string: %T", rt["enum"])
	}

	gotEnum := make([]string, len(enum))
	copy(gotEnum, enum)
	sort.Strings(gotEnum)

	want := aicallhandler.SupportedResourceTypes()

	if len(gotEnum) != len(want) {
		t.Fatalf("enum length %d != fetcher table length %d. enum: %v, fetchers: %v", len(gotEnum), len(want), gotEnum, want)
	}
	for i := range want {
		if gotEnum[i] != want[i] {
			t.Errorf("enum[%d] = %s, want %s (enum: %v, fetchers: %v)", i, gotEnum[i], want[i], gotEnum, want)
		}
	}
}

// TestGetResourceIncludeConfigSchema locks the include_config parameter shape
// in the get_resource JSON schema (design 2026-06-12 test 13): it must exist
// as a boolean property and must NOT be required (opt-in, default off).
func TestGetResourceIncludeConfigSchema(t *testing.T) {
	var def *tool.Tool
	for i := range toolDefinitions {
		if toolDefinitions[i].Name == tool.ToolNameGetResource {
			def = &toolDefinitions[i]
			break
		}
	}
	if def == nil {
		t.Fatalf("get_resource tool definition not found in definitions.go")
	}

	props, ok := def.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource parameters has no properties map")
	}

	ic, ok := props["include_config"].(map[string]any)
	if !ok {
		t.Fatalf("get_resource has no include_config property")
	}
	if got, _ := ic["type"].(string); got != "boolean" {
		t.Errorf("include_config type = %q, want \"boolean\"", got)
	}

	req, ok := def.Parameters["required"].([]string)
	if !ok {
		t.Fatalf("get_resource required is not []string: %T", def.Parameters["required"])
	}
	wantReq := []string{"resource_type", "resource_id"}
	if len(req) != len(wantReq) {
		t.Fatalf("required = %v, want exactly %v (include_config must stay optional)", req, wantReq)
	}
	for i := range wantReq {
		if req[i] != wantReq[i] {
			t.Errorf("required[%d] = %s, want %s", i, req[i], wantReq[i])
		}
	}
}
