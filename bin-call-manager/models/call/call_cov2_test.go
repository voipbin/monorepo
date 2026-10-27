package call

import (
	"testing"

	"monorepo/bin-call-manager/models/ari"
)

func Test_GetStatusByChannelState_cov2_unknownState(t *testing.T) {
	res := GetStatusByChannelState(ari.ChannelState("not-a-state"))
	if res != StatusHangup {
		t.Errorf("Wrong match. expect: %s, got: %s", StatusHangup, res)
	}
}

func Test_CalculateHangupReason_cov2_unknownStatus(t *testing.T) {
	tests := []struct {
		name string

		direction  Direction
		lastStatus Status

		expect HangupReason
	}{
		{name: "outgoing unknown status", direction: DirectionOutgoing, lastStatus: Status("unknown"), expect: HangupReasonNormal},
		{name: "incoming unknown status", direction: DirectionIncoming, lastStatus: Status("unknown"), expect: HangupReasonNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := CalculateHangupReason(tt.direction, tt.lastStatus, ari.ChannelCauseNormalClearing)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

func Test_CalculateHangupBy_cov2_unknownStatus(t *testing.T) {
	if res := CalculateHangupBy(Status("unknown")); res != HangupByRemote {
		t.Errorf("Wrong match. expect: %s, got: %s", HangupByRemote, res)
	}
}

func Test_IsUpdatableStatus_cov2_unknownOldStatus(t *testing.T) {
	if IsUpdatableStatus(Status("unknown"), StatusProgressing) {
		t.Errorf("Wrong match. expect: false, got: true")
	}
}
