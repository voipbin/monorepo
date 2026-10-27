package bridge

import (
	"reflect"
	"testing"
)

func Test_ParseBridgeName_cov2_malformedPair(t *testing.T) {
	res := ParseBridgeName("reference_type=call,malformed,reference_id=abc")
	expect := map[string]string{"reference_type": "call", "reference_id": "abc"}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res)
	}
}
