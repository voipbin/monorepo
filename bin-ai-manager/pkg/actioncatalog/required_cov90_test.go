package actioncatalog

import (
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"
)

func Test_RequiredFields_UnknownType(t *testing.T) {
	res := RequiredFields(fmaction.Type("no_such_action_type"))
	if res == nil || len(res) != 0 {
		t.Errorf("expected a non-nil empty map for an unknown type, got: %#v", res)
	}
}

func Test_RequiredFields_KnownType(t *testing.T) {
	res := RequiredFields(fmaction.TypeAISummary)
	if len(res) != 2 || !res["reference_type"] || !res["reference_id"] {
		t.Errorf("expected exactly reference_type and reference_id to be required for the ai summary action, got: %v", res)
	}

	if res := RequiredFields(fmaction.TypeAnswer); res == nil || len(res) != 0 {
		t.Errorf("expected no required field for the answer action, got: %#v", res)
	}
}

func Test_RequiredFields_MatchesCatalog(t *testing.T) {
	checked := 0
	for actionType, entry := range catalogByType {
		expect := map[string]bool{}
		for _, o := range entry.Options {
			if o.Required {
				expect[o.Name] = true
			}
		}

		got := RequiredFields(actionType)
		if len(got) != len(expect) {
			t.Errorf("%s: expected %d required fields, got %d", actionType, len(expect), len(got))
		}
		for name := range expect {
			if !got[name] {
				t.Errorf("%s: required field %q is missing", actionType, name)
			}
		}
		for name := range got {
			if !expect[name] {
				t.Errorf("%s: field %q is reported required but the catalog does not require it", actionType, name)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatalf("the catalog is empty")
	}
}
