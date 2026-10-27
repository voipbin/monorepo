package twilio

import (
	"reflect"
	"testing"

	"monorepo/bin-number-manager/models/availablenumber"
)

func Test_ConvertAvailableNumber_mms(t *testing.T) {
	in := &AvailableNumber{
		PhoneNumber:  "+155****0002",
		Capabilities: Capabilities{MMS: true},
	}

	res := in.ConvertAvailableNumber()

	expect := []availablenumber.Feature{availablenumber.FeatureMMS}
	if !reflect.DeepEqual(res.Features, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, res.Features)
	}
}
