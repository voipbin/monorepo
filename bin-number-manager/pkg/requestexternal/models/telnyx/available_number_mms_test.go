package telnyx

import (
	"reflect"
	"testing"

	"monorepo/bin-number-manager/models/availablenumber"
)

func Test_ConvertAvailableNumber_mms(t *testing.T) {
	in := &AvailableNumber{
		PhoneNumber: "+161****0189",
		Features:    []AvailableFeature{{Name: "mms"}, {Name: "unknown-feature"}},
	}

	res := in.ConvertAvailableNumber()

	expect := []availablenumber.Feature{availablenumber.FeatureMMS}
	if !reflect.DeepEqual(res.Features, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res.Features)
	}
}
