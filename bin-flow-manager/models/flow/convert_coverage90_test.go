package flow

import "testing"

func Test_ConvertStringMapToFieldMap_unconvertibleValue_coverage90(t *testing.T) {
	// a number can not be converted to the uuid of the id field.
	res, err := ConvertStringMapToFieldMap(map[string]any{"id": 12345})
	if err == nil {
		t.Fatalf("expected a conversion failure, got nil")
	}
	if res != nil {
		t.Errorf("no map is expected with the error. got: %v", res)
	}
}
