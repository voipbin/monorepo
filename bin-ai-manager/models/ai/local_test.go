package ai

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// These tests change the global local registry; they do not use t.Parallel().

func resetLocal(t *testing.T) {
	t.Helper()
	SetLocalModels(nil)
	t.Cleanup(func() { SetLocalModels(nil) })
}

func Test_ParseLocalModels(t *testing.T) {
	long := strings.Repeat("a", 300)

	tests := []struct {
		name string
		raw  string

		wantLoaded  int
		wantSkipped int
		wantDupDef  int
		wantErr     bool
	}{
		{name: "empty", raw: "", wantLoaded: 0},
		{name: "blank", raw: "  \n", wantLoaded: 0},
		{name: "empty list", raw: "[]", wantLoaded: 0},
		{name: "valid", raw: `[{"endpoint":"vllm-a","id":"qwen3-8b","label":"Qwen3 8B","capabilities":["tool-calling"]}]`, wantLoaded: 1},
		{name: "id with colon slash dot upper", raw: `[{"endpoint":"a","id":"Org/Model:7b.v2","label":"M"}]`, wantLoaded: 1},
		{name: "malformed json", raw: `[{`, wantErr: true},
		{name: "not a list", raw: `{"endpoint":"a"}`, wantErr: true},
		{name: "bad endpoint upper", raw: `[{"endpoint":"A","id":"m","label":"M"}]`, wantSkipped: 1},
		{name: "bad endpoint dot", raw: `[{"endpoint":"a.b","id":"m","label":"M"}]`, wantSkipped: 1},
		{name: "bad endpoint empty", raw: `[{"endpoint":"","id":"m","label":"M"}]`, wantSkipped: 1},
		{name: "bad id", raw: `[{"endpoint":"a","id":"-m","label":"M"}]`, wantSkipped: 1},
		{name: "bad id space", raw: `[{"endpoint":"a","id":"m m","label":"M"}]`, wantSkipped: 1},
		{name: "bad label non ascii", raw: "[{\"endpoint\":\"a\",\"id\":\"m\",\"label\":\"모델\"}]", wantSkipped: 1},
		{name: "bad label empty", raw: `[{"endpoint":"a","id":"m","label":""}]`, wantSkipped: 1},
		{name: "too long id", raw: fmt.Sprintf(`[{"endpoint":"a","id":"%s","label":"M"}]`, long), wantSkipped: 1},
		{name: "duplicate", raw: `[{"endpoint":"a","id":"m","label":"M"},{"endpoint":"a","id":"m","label":"N"}]`, wantLoaded: 1, wantSkipped: 1},
		{name: "same id other endpoint", raw: `[{"endpoint":"a","id":"m","label":"M"},{"endpoint":"b","id":"m","label":"M"}]`, wantLoaded: 2},
		{name: "second default loses flag", raw: `[{"endpoint":"a","id":"m","label":"M","default":true},{"endpoint":"a","id":"n","label":"N","default":true}]`, wantLoaded: 2, wantDupDef: 1},
		{name: "oversized input", raw: strings.Repeat(" ", maxLocalModelsInputBytes+1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ParseLocalModels(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if len(res.Entries) != tt.wantLoaded || res.Skipped != tt.wantSkipped || res.DuplicateDefaults != tt.wantDupDef {
				t.Errorf("loaded=%d skipped=%d dupdef=%d, want %d %d %d", len(res.Entries), res.Skipped, res.DuplicateDefaults, tt.wantLoaded, tt.wantSkipped, tt.wantDupDef)
			}
		})
	}
}

func Test_ParseLocalModelsLimit(t *testing.T) {
	items := make([]string, 0, maxLocalModels+1)
	for i := 0; i < maxLocalModels+1; i++ {
		items = append(items, fmt.Sprintf(`{"endpoint":"a","id":"m%d","label":"M"}`, i))
	}

	res, err := ParseLocalModels("[" + strings.Join(items, ",") + "]")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(res.Entries) != maxLocalModels || res.Skipped != 1 {
		t.Errorf("loaded=%d skipped=%d, want %d 1", len(res.Entries), res.Skipped, maxLocalModels)
	}
}

func Test_ParseLocalModelsEntryShape(t *testing.T) {
	res, err := ParseLocalModels(`[{"endpoint":"vllm-a","id":"qwen3-8b","label":"Qwen3 8B","capabilities":["tool-calling","bogus","low-cost","tool-calling"],"default":true}]`)
	if err != nil || len(res.Entries) != 1 {
		t.Fatalf("err=%v entries=%d", err, len(res.Entries))
	}

	e := res.Entries[0]
	if e.ID != "local.vllm-a.qwen3-8b" || e.Route != RouteLocal || e.Vendor != "Local" || !e.Recommended || e.Description != "Local model." {
		t.Errorf("unexpected entry: %+v", e)
	}
	if len(e.Tags) != 2 || e.Tags[0] != TagToolCalling || e.Tags[1] != TagLowCost {
		t.Errorf("tags = %v", e.Tags)
	}
}

func Test_ConfigureLocalModelsErrorEmptiesRegistry(t *testing.T) {
	resetLocal(t)

	if _, err := ConfigureLocalModels(`[{"endpoint":"a","id":"m","label":"M"}]`); err != nil {
		t.Fatal(err)
	}
	if !IsValidEngineModel("local.a.m") {
		t.Fatal("configured model must be valid")
	}

	if _, err := ConfigureLocalModels(`[{`); err == nil {
		t.Fatal("want error")
	}
	if IsValidEngineModel("local.a.m") {
		t.Error("registry must be empty after a failed configure")
	}
}

func Test_SetLocalModelsCopies(t *testing.T) {
	resetLocal(t)

	in := []ModelEntry{{ID: "local.a.m", Label: "M", Route: RouteLocal, Tags: []string{TagLowCost}}}
	SetLocalModels(in)
	in[0].Tags[0] = "changed"
	in[0].Label = "changed"

	view := CatalogView()
	got := view[len(view)-1]
	if got.Label != "M" || got.Tags[0] != TagLowCost {
		t.Errorf("registry changed through caller slice: %+v", got)
	}
}

func Test_LocalResolve(t *testing.T) {
	resetLocal(t)

	if _, o := ResolveEngine("local.a.m"); o != OutcomeRejected {
		t.Fatal("unconfigured local model must be rejected")
	}

	SetLocalModels(mustParseLocal(t, `[{"endpoint":"a","id":"Org/M:7b.v2","label":"M"}]`))

	r, o := ResolveEngine("local.a.Org/M:7b.v2")
	if o != OutcomeCatalog || !r.BlankKey || r.RequireKey || r.RunnerType != "local.a.Org/M:7b.v2" || r.Entry == nil || r.Entry.Route != RouteLocal {
		t.Errorf("resolved = %+v outcome = %v", r, o)
	}

	for _, m := range []EngineModel{"local.a.other", "local.b.Org/M:7b.v2", "local.", "LOCAL.a.Org/M:7b.v2"} {
		if _, o := ResolveEngine(m); o != OutcomeRejected {
			t.Errorf("%s must be rejected", m)
		}
	}

	if err := ValidateEngine("local.a.Org/M:7b.v2", "sk-customer", true); err != nil {
		t.Errorf("ValidateEngine = %v", err)
	}
}

func Test_LocalCatalogView(t *testing.T) {
	resetLocal(t)

	base := CatalogView()
	baseRecommended := 0
	for _, m := range base {
		if m.Recommended {
			baseRecommended++
		}
	}
	if baseRecommended != 1 {
		t.Fatalf("unconfigured recommended = %d", baseRecommended)
	}

	SetLocalModels(mustParseLocal(t, `[{"endpoint":"vllm-a","id":"m1","label":"Model One","capabilities":["tool-calling"],"default":true},{"endpoint":"vllm-a","id":"m2","label":"Model Two"}]`))

	view := CatalogView()
	if len(view) != len(base)+2 {
		t.Fatalf("len = %d, want %d", len(view), len(base)+2)
	}

	recommended := []EngineModel{}
	for _, m := range view {
		if m.Recommended {
			recommended = append(recommended, m.ID)
		}
	}
	if len(recommended) != 1 || recommended[0] != "local.vllm-a.m1" {
		t.Errorf("recommended = %v", recommended)
	}

	l1 := view[len(view)-2]
	if l1.KeyMode != KeyModePlatform || !l1.PlatformManaged || l1.Vendor != "Local" || len(l1.Tags) != 1 || l1.Tags[0] != TagToolCalling {
		t.Errorf("local view = %+v", l1)
	}
	if l2 := view[len(view)-1]; l2.Tags == nil || len(l2.Tags) != 0 {
		t.Errorf("tags must be an empty list: %+v", l2)
	}

	// static catalog is not mutated: after clearing, the built-in default is back.
	SetLocalModels(nil)
	again := CatalogView()
	n := 0
	for _, m := range again {
		if m.Recommended {
			n++
		}
	}
	if n != 1 {
		t.Errorf("recommended after clear = %d", n)
	}
}

func Test_LocalCatalogViewJSONHasNoConnectionData(t *testing.T) {
	resetLocal(t)
	SetLocalModels(mustParseLocal(t, `[{"endpoint":"vllm-a","id":"m1","label":"Model One"}]`))

	data, err := json.Marshal(CatalogView())
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"base_url", "api_key", "http://", "https://"} {
		if strings.Contains(string(data), banned) {
			t.Errorf("view contains %q", banned)
		}
	}
}

func Test_LogFieldsCountsOnly(t *testing.T) {
	res, _ := ParseLocalModels(`[{"endpoint":"a","id":"m","label":"M","default":true},{"endpoint":"A","id":"m","label":"M"}]`)
	f := res.LogFields()
	if f["local_models_loaded"] != 1 || f["local_models_skipped"] != 1 || f["local_models_default"] != 1 {
		t.Errorf("fields = %v", f)
	}
}

func mustParseLocal(t *testing.T, raw string) []ModelEntry {
	t.Helper()
	res, err := ParseLocalModels(raw)
	if err != nil || res.Skipped != 0 {
		t.Fatalf("parse err=%v skipped=%d", err, res.Skipped)
	}
	return res.Entries
}
