package channelhandler

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-call-manager/models/channel"
)

func Test_AddressGetDestinationWithoutSpecificType_cov2_invalidEscape(t *testing.T) {
	h, _ := newCovHandler(t)

	// the destination can not be unescaped, so the raw value is used as an extension.
	cn := &channel.Channel{DestinationNumber: "bad%zzvalue", DestinationName: "c2-name"}
	expect := &commonaddress.Address{
		Type:       commonaddress.TypeExtension,
		Target:     "bad%zzvalue",
		TargetName: "c2-name",
	}

	res := h.AddressGetDestinationWithoutSpecificType(cn)
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}
