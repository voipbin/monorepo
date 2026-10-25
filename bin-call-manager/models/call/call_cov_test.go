package call

import (
	"strings"
	"testing"
	"time"

	"monorepo/bin-call-manager/models/ari"
)

func Test_CallMatches_cov(t *testing.T) {
	now := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		name   string
		a      *Call
		b      *Call
		expect bool
	}{
		{
			name:   "same",
			a:      &Call{ChannelID: "ch-1", Status: StatusDialing},
			b:      &Call{ChannelID: "ch-1", Status: StatusDialing},
			expect: true,
		},
		{
			name:   "test channel id wildcard",
			a:      &Call{ChannelID: TestChannelID, Status: StatusDialing},
			b:      &Call{ChannelID: "any-generated-channel-id", Status: StatusDialing},
			expect: true,
		},
		{
			name:   "timestamps are ignored",
			a:      &Call{ChannelID: "ch-1", TMCreate: &now},
			b:      &Call{ChannelID: "ch-1", TMCreate: &later, TMUpdate: &later, TMRinging: &later, TMProgressing: &later, TMHangup: &later},
			expect: true,
		},
		{
			name:   "different channel id without wildcard",
			a:      &Call{ChannelID: "ch-1"},
			b:      &Call{ChannelID: "ch-2"},
			expect: false,
		},
		{
			name:   "different status",
			a:      &Call{ChannelID: "ch-1", Status: StatusDialing},
			b:      &Call{ChannelID: "ch-1", Status: StatusRinging},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := tt.a.Matches(tt.b); res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_CallString_cov(t *testing.T) {
	c := &Call{ChannelID: "ch-string-id", Status: StatusRinging}

	res := c.String()
	if !strings.Contains(res, "ch-string-id") || !strings.Contains(res, string(StatusRinging)) {
		t.Errorf("Wrong match. the string must contain the channel id and status. got: %s", res)
	}
}

func Test_CalculateHangupReason_cov(t *testing.T) {
	tests := []struct {
		name       string
		direction  Direction
		lastStatus Status
		cause      ari.ChannelCause
		expect     HangupReason
	}{
		// dispatches to the outgoing table
		{"outgoing ringing user busy", DirectionOutgoing, StatusRinging, ari.ChannelCauseUserBusy, HangupReasonBusy},
		{"outgoing canceling", DirectionOutgoing, StatusCanceling, ari.ChannelCauseNormalClearing, HangupReasonCanceled},
		// dispatches to the incoming table; the outgoing table would return failed for this cause
		{"incoming ringing no answer", DirectionIncoming, StatusRinging, ari.ChannelCauseNoAnswer, HangupReasonNoanswer},
		{"incoming progressing normal", DirectionIncoming, StatusProgressing, ari.ChannelCauseNormalClearing, HangupReasonNormal},
		// any non-outgoing direction goes through the incoming table
		{"empty direction uses incoming", "", StatusRinging, ari.ChannelCauseCallRejected, HangupReasonNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := CalculateHangupReason(tt.direction, tt.lastStatus, tt.cause); res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}

func Test_ConvertHangupReasonToChannelCause_cov(t *testing.T) {
	tests := []struct {
		reason HangupReason
		expect ari.ChannelCause
	}{
		{HangupReasonNone, ari.ChannelCauseNormalClearing},
		{HangupReasonNormal, ari.ChannelCauseNormalClearing},
		{HangupReasonFailed, ari.ChannelCauseNoUserResponse},
		{HangupReasonBusy, ari.ChannelCauseUserBusy},
		{HangupReasonCanceled, ari.ChannelCauseNormalClearing},
		{HangupReasonTimeout, ari.ChannelCauseNormalClearing},
		{HangupReasonNoanswer, ari.ChannelCauseNoAnswer},
		{HangupReasonDialout, ari.ChannelCauseNoAnswer},
		{HangupReasonAMD, ari.ChannelCauseCallAMD},
		{HangupReason("unknown-reason"), ari.ChannelCauseNormalClearing},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			if res := ConvertHangupReasonToChannelCause(tt.reason); res != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, res)
			}
		})
	}
}
