package action

import "testing"

// Test_OptionStructByType_CoversTypeListAll asserts every action Type in
// TypeListAll has an entry in OptionStructByType (and no extra entries), so the
// map cannot silently drift from the authoritative type list.
func Test_OptionStructByType_CoversTypeListAll(t *testing.T) {
	for _, ty := range TypeListAll {
		if _, ok := OptionStructByType[ty]; !ok {
			t.Errorf("OptionStructByType missing entry for type %s", ty)
		}
	}
	if len(OptionStructByType) != len(TypeListAll) {
		t.Errorf("OptionStructByType has %d entries, TypeListAll has %d (extra/duplicate entries?)", len(OptionStructByType), len(TypeListAll))
	}
}
