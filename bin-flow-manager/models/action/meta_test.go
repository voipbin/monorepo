package action

import "testing"

// Test_MetaCoversAllTypes locks MetaByType to TypeListAll: every action Type
// the flow engine knows about must have a Flow AI Builder Meta declaration,
// and MetaByType must not carry a stale entry for a type that no longer
// exists. See VOIP-1573 design doc Appendix B.
func Test_MetaCoversAllTypes(t *testing.T) {
	want := make(map[Type]bool, len(TypeListAll))
	for _, ty := range TypeListAll {
		want[ty] = true
		if _, ok := MetaByType[ty]; !ok {
			t.Errorf("TypeListAll has %q but MetaByType is missing it; add a Meta entry (VOIP-1573)", ty)
		}
	}
	for ty := range MetaByType {
		if !want[ty] {
			t.Errorf("MetaByType has stale entry %q that is not in TypeListAll; remove it", ty)
		}
	}
}

// TestBuilderExcludedTypesAreInTypeListAll pins the structurally-excluded
// set so a silent drift (e.g. someone renaming OptionCall.Actions away from
// a nested []Action, which would make exclusion unnecessary, or a new type
// gaining a nested []Action/[]Attachment without being added here) is
// visible instead of silently mis-including/excluding a type.
func Test_BuilderExcludedTypesAreInTypeListAll(t *testing.T) {
	want := map[Type]bool{
		TypeCall:      true,
		TypeEmailSend: true,
	}
	if len(BuilderExcludedTypes) != len(want) {
		t.Fatalf("BuilderExcludedTypes = %v, want exactly %v", BuilderExcludedTypes, want)
	}
	for ty := range want {
		if !BuilderExcludedTypes[ty] {
			t.Errorf("BuilderExcludedTypes is missing %q", ty)
		}
		found := false
		for _, l := range TypeListAll {
			if l == ty {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BuilderExcludedTypes has %q which is not in TypeListAll", ty)
		}
	}
}

func Test_IsBuilderExposable(t *testing.T) {
	tests := []struct {
		name string
		ty   Type
		want bool
	}{
		{"core type is exposable", TypeTalk, true},
		{"sensitive type is exposable", TypeConnect, true},
		{"internal type is not exposable", TypeExternalMediaStart, false},
		{"goto is internal and not exposable", TypeGoto, false},
		{"call is structurally excluded despite sensitive Exposure", TypeCall, false},
		{"email_send is structurally excluded despite sensitive Exposure", TypeEmailSend, false},
		{"unknown type is not exposable", Type("does_not_exist"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBuilderExposable(tt.ty); got != tt.want {
				t.Errorf("IsBuilderExposable(%q) = %v, want %v", tt.ty, got, tt.want)
			}
		})
	}
}
