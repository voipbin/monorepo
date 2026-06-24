package actioncatalog

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"
)

// TestActionCatalogMatchesTypeListAll locks the describe_action catalog's type
// set to flow-manager action.TypeListAll: one entry per type, no missing, no
// extra, no duplicate. It iterates the raw slice (not the lookup map) so a
// duplicated entry is caught rather than silently deduped.
func TestActionCatalogMatchesTypeListAll(t *testing.T) {
	seen := map[fmaction.Type]bool{}
	got := make([]string, 0, len(actionCatalog))
	for _, e := range actionCatalog {
		if seen[e.Type] {
			t.Errorf("duplicate catalog entry for type %s", e.Type)
			continue
		}
		seen[e.Type] = true
		got = append(got, string(e.Type))
	}
	sort.Strings(got)

	want := make([]string, len(fmaction.TypeListAll))
	for i, ty := range fmaction.TypeListAll {
		want[i] = string(ty)
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("catalog has %d unique types, TypeListAll has %d. catalog: %v, TypeListAll: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("type[%d] = %s, want %s (catalog: %v, TypeListAll: %v)", i, got[i], want[i], got, want)
		}
	}
}

// TestActionCatalogFieldsMatchOptionStructs verifies, for every action type,
// that the catalog entry's option field NAMES equal the real option struct's
// top-level json field names (via action.OptionStructByType). This closes the
// stale-option-field gap automatically without codegen. Field TYPE/Description
// text is NOT checked (reflection cannot read comments) and is maintained by
// hand per the option.go sync note.
func TestActionCatalogFieldsMatchOptionStructs(t *testing.T) {
	for _, ty := range fmaction.TypeListAll {
		entry, ok := catalogByType[ty]
		if !ok {
			t.Errorf("%s: no catalog entry (covered by TestActionCatalogMatchesTypeListAll)", ty)
			continue
		}

		st, ok := fmaction.OptionStructByType[ty]
		if !ok {
			t.Errorf("%s: no entry in action.OptionStructByType", ty)
			continue
		}

		wantFields := topLevelJSONFieldNames(reflect.TypeOf(st))
		gotFields := map[string]bool{}
		for _, o := range entry.Options {
			gotFields[o.Name] = true
		}

		for name := range wantFields {
			if !gotFields[name] {
				t.Errorf("%s: option struct has json field %q but catalog entry does not list it", ty, name)
			}
		}
		for name := range gotFields {
			if !wantFields[name] {
				t.Errorf("%s: catalog lists option %q but option struct has no such top-level json field", ty, name)
			}
		}
	}
}

// topLevelJSONFieldNames returns the set of top-level json field names for a
// struct type. It honors json tag semantics: drops the ",omitempty"/",string"
// suffix, skips json:"-" fields, skips unexported fields, and falls back to the
// Go field name when an exported field has no json tag. Non-struct types (e.g.
// struct{}{} used for option-less actions) yield an empty set.
func topLevelJSONFieldNames(rt reflect.Type) map[string]bool {
	out := map[string]bool{}
	if rt == nil {
		return out
	}
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return out
	}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		tag, hasTag := f.Tag.Lookup("json")
		name := f.Name
		if hasTag {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
		}
		out[name] = true
	}
	return out
}

// TestRenderActionCatalogEntry pins the exact rendered text for one
// option-bearing and one option-less entry. The render format is a contract
// consumed by the LLM, so accidental format changes must fail loudly.
func TestRenderActionCatalogEntry(t *testing.T) {
	optionBearing := actionCatalogEntry{
		Type:    fmaction.TypeSleep,
		Summary: "Pause the flow for a duration.",
		Options: []actionOptionField{
			{Name: "duration", Type: "int (ms)", Required: true, Description: "Sleep duration in milliseconds."},
		},
	}
	wantBearing := "action: sleep\n" +
		"summary: Pause the flow for a duration.\n" +
		"options:\n" +
		"  - duration (int (ms), required): Sleep duration in milliseconds."
	if got := renderActionCatalogEntry(optionBearing); got != wantBearing {
		t.Errorf("option-bearing render mismatch.\n got: %q\nwant: %q", got, wantBearing)
	}

	optionLess := actionCatalogEntry{
		Type:    fmaction.TypeAnswer,
		Summary: "Answer the incoming call.",
		Options: nil,
	}
	wantLess := "action: answer\n" +
		"summary: Answer the incoming call.\n" +
		"options: this action takes no options."
	if got := renderActionCatalogEntry(optionLess); got != wantLess {
		t.Errorf("option-less render mismatch.\n got: %q\nwant: %q", got, wantLess)
	}
}

// TestDescribeAction covers the exported lookup: success, unknown type, and
// empty type (both error paths wrap ErrUnknownActionType with an echo + hint).
func TestDescribeAction(t *testing.T) {
	// success
	got, err := DescribeAction("talk")
	if err != nil {
		t.Fatalf("DescribeAction(talk) unexpected error: %v", err)
	}
	if !strings.Contains(got, "action: talk") || !strings.Contains(got, "text") {
		t.Errorf("DescribeAction(talk) missing expected content: %q", got)
	}

	// unknown type
	_, err = DescribeAction("not_a_real_action")
	if err == nil {
		t.Errorf("DescribeAction(unknown) expected error")
	} else if !strings.Contains(err.Error(), "not_a_real_action") {
		t.Errorf("unknown-type error should echo the value: %v", err)
	}

	// empty type
	_, err = DescribeAction("")
	if err == nil {
		t.Errorf("DescribeAction(empty) expected error")
	}
}
