package mcpschema

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// updateGoldens regenerates testdata/**/*.golden.json and testdata/live from
// the current implementation. Goldens are reviewed by hand against the
// design (section 15.7 item 4) before they are checked in.
var updateGoldens = flag.Bool("update", false, "rewrite golden files from the current Normalize output")

// goldenReport is the part of Report pinned next to each golden.
type goldenReport struct {
	ToolDropped   bool     `json:"tool_dropped"`
	DropReason    string   `json:"drop_reason,omitempty"`
	DroppedProps  []string `json:"dropped_props,omitempty"`
	DroppedPropsN int      `json:"dropped_props_n"`
}

// goldenFile is the content of one *.golden.json.
type goldenFile struct {
	Schema map[string]any `json:"schema"`
	Report goldenReport   `json:"report"`
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("could not decode %s: %v", path, err)
	}
	return m
}

// roundTrip normalizes number and slice types for DeepEqual.
func roundTrip(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal: %v", err)
	}
	var res any
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatalf("could not unmarshal: %v", err)
	}
	return res
}

func writeGolden(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("could not marshal golden: %v", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("could not write %s: %v", path, err)
	}
}

// checkGoldenDir normalizes every <name>.json in dir (other than goldens)
// and compares it with <name>.golden.json as decoded maps.
func checkGoldenDir(t *testing.T, dir string, wantNames []string) map[string]goldenFile {
	t.Helper()
	inputs, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	got := map[string]goldenFile{}
	names := []string{}
	for _, in := range inputs {
		if strings.HasSuffix(in, ".golden.json") {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(in), ".json")
		names = append(names, name)

		out, rep := Normalize(readJSONFile(t, in), defaultMaxOut)
		g := goldenFile{Schema: out, Report: goldenReport{
			ToolDropped:   rep.ToolDropped,
			DropReason:    rep.DropReason,
			DroppedProps:  rep.DroppedProps,
			DroppedPropsN: rep.DroppedPropsN,
		}}
		got[name] = g

		goldenPath := filepath.Join(dir, name+".golden.json")
		if *updateGoldens {
			writeGolden(t, goldenPath, g)
			continue
		}
		want := readJSONFile(t, goldenPath)
		if !reflect.DeepEqual(roundTrip(t, g), want) {
			b, _ := json.MarshalIndent(g, "", "  ")
			t.Errorf("Wrong match. %s differs from its golden\ngot: %s", name, b)
		}
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, wantNames) {
		t.Errorf("Wrong match. fixtures expect: %v, got: %v", wantNames, names)
	}
	return got
}

// at walks a normalized schema by JSON pointer segments.
func at(t *testing.T, s map[string]any, segs ...string) any {
	t.Helper()
	if s == nil {
		return nil
	}
	var cur any = s
	for _, seg := range segs {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %v: %q is not under an object", segs, seg)
		}
		cur, ok = m[seg]
		if !ok {
			return nil
		}
	}
	return cur
}

func Test_Normalize_GitHubMCPFixtures(t *testing.T) {
	got := checkGoldenDir(t, filepath.Join("testdata", "github"), []string{
		"custom_properties_write",
		"get_file_contents",
		"incident_x_mcp_header",
		"issue_write",
		"projects_write",
		"update_issue_labels",
	})

	for name, g := range got {
		if g.Report.ToolDropped {
			t.Errorf("Wrong match. %s must be kept, got dropped (%s)", name, g.Report.DropReason)
		}
	}

	// Pinned expectations (design section 15.7 item 4). These do not come
	// from the goldens: a golden update cannot move them.
	pw := got["projects_write"]
	if !reflect.DeepEqual(pw.Report.DroppedProps, []string{"/properties/updated_field"}) || pw.Report.DroppedPropsN != 1 {
		t.Errorf("Wrong match. projects_write must drop only /properties/updated_field, got %v (%d)", pw.Report.DroppedProps, pw.Report.DroppedPropsN)
	}
	if v := at(t, pw.Schema, "properties", "updated_field"); v != nil {
		t.Errorf("Wrong match. projects_write updated_field must be removed, got %v", v)
	}
	if v := roundTrip(t, at(t, pw.Schema, "properties", "items", "items")); !reflect.DeepEqual(v, map[string]any{"type": "object"}) {
		t.Errorf("Wrong match. projects_write /properties/items/items expect: {type: object}, got: %v", v)
	}

	cp := got["custom_properties_write"]
	if v := roundTrip(t, at(t, cp.Schema, "properties", "properties", "items")); !reflect.DeepEqual(v, map[string]any{"type": "object"}) {
		t.Errorf("Wrong match. custom_properties_write /properties/properties/items expect: {type: object}, got: %v", v)
	}

	labels, _ := roundTrip(t, at(t, got["update_issue_labels"].Schema, "properties", "labels", "items")).(map[string]any)
	if _, hasType := labels["type"]; hasType {
		t.Errorf("Wrong match. update_issue_labels labels.items must have no type, got %v", labels)
	}
	members, _ := labels["anyOf"].([]any)
	if len(members) != 2 {
		t.Fatalf("Wrong match. update_issue_labels labels.items expect 2 anyOf members, got %v", labels)
	}
	if !reflect.DeepEqual(members[0], map[string]any{"type": "string", "description": "Label name"}) {
		t.Errorf("Wrong match. labels.items.anyOf[0] expect: {type: string, description: Label name}, got: %v", members[0])
	}
	m1, _ := members[1].(map[string]any)
	if m1["type"] != "object" || !reflect.DeepEqual(m1["required"], []any{"name"}) || m1["properties"] == nil {
		t.Errorf("Wrong match. labels.items.anyOf[1] expect an object requiring name, got: %v", m1)
	}

	value, _ := roundTrip(t, at(t, got["issue_write"].Schema, "properties", "issue_fields", "items", "properties", "value")).(map[string]any)
	if value["description"] == nil || value["type"] != nil {
		t.Errorf("Wrong match. issue_write value must carry the description on the parent and no type, got %v", value)
	}
	wantMembers := []any{
		map[string]any{"type": "string"},
		map[string]any{"type": "number"},
		map[string]any{"type": "boolean"},
	}
	if !reflect.DeepEqual(value["anyOf"], wantMembers) {
		t.Errorf("Wrong match. issue_write value.anyOf expect: %v, got: %v", wantMembers, value["anyOf"])
	}

	inc := roundTrip(t, got["incident_x_mcp_header"].Schema)
	b, _ := json.Marshal(inc)
	if strings.Contains(string(b), "x-mcp-header") {
		t.Errorf("Wrong match. incident fixture still carries x-mcp-header: %s", b)
	}
}

// liveInputs are the synthetic shapes from the analysis (section 7) that the
// pre-merge live provider call sends (design section 15.8). Their normalized
// form is checked in as testdata/live/<name>.json, the exact payload, and
// this test keeps it equal to the current Normalize output.
var liveInputs = map[string]string{
	"nested_free_form_object": `{"type":"object","properties":{"variables":{"type":"object",
		"description":"Free-form key/value variables.","additionalProperties":{"type":"string"}}},"required":["variables"]}`,
	"nested_empty_properties": `{"type":"object","properties":{"options":{"type":"object",
		"description":"Options object with explicit empty properties.","properties":{}}}}`,
	"nullable_anyof": `{"type":"object","title":"Args","properties":{"label":{"anyOf":[{"type":"string"},{"type":"null"}],
		"default":null,"title":"Label","description":"Optional label; null clears it."}}}`,
	"typeless_string_enum": `{"type":"object","properties":{"priority":{"enum":["low","medium","high"],
		"description":"Priority level."}},"required":["priority"]}`,
}

func Test_Normalize_LiveFixtures(t *testing.T) {
	for name, in := range liveInputs {
		t.Run(name, func(t *testing.T) {
			out, rep := Normalize(decodeJSON(t, in), defaultMaxOut)
			if rep.ToolDropped || rep.DroppedPropsN != 0 {
				t.Fatalf("Wrong match. live fixture must be kept whole, got %+v", rep)
			}
			path := filepath.Join("testdata", "live", name+".json")
			if *updateGoldens {
				writeGolden(t, path, out)
				return
			}
			if want := readJSONFile(t, path); !reflect.DeepEqual(roundTrip(t, out), any(want)) {
				b, _ := json.Marshal(out)
				t.Errorf("Wrong match. %s differs from Normalize output\ngot: %s", path, b)
			}
		})
	}

	files, err := filepath.Glob(filepath.Join("testdata", "live", "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) != len(liveInputs) {
		t.Errorf("Wrong match. testdata/live expect %d files, got %v", len(liveInputs), files)
	}
}
