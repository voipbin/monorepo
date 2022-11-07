package call

import (
	"testing"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

func TestIsUpdatableStatus(t *testing.T) {
	type test struct {
		name      string
		oldStatus Status
		newStatus Status
		expect    bool
	}

	tests := []test{
		{"dialing->dialing", StatusDialing, StatusDialing, false},
		{"dialing->ringing", StatusDialing, StatusRinging, true},
		{"dialing->progressing", StatusDialing, StatusProgressing, true},
		{"dialing->terminating", StatusDialing, StatusTerminating, true},
		{"dialing->canceling", StatusDialing, StatusCanceling, true},
		{"dialing->hangup", StatusDialing, StatusHangup, true},

		{"ringing->dialing", StatusRinging, StatusDialing, false},
		{"ringing->ringing", StatusRinging, StatusRinging, false},
		{"ringing->progressing", StatusRinging, StatusProgressing, true},
		{"ringing->terminating", StatusRinging, StatusTerminating, true},
		{"ringing->canceling", StatusRinging, StatusCanceling, true},
		{"ringing->hangup", StatusRinging, StatusHangup, true},

		{"progressing->dialing", StatusProgressing, StatusDialing, false},
		{"progressing->ringing", StatusProgressing, StatusRinging, false},
		{"progressing->progressing", StatusProgressing, StatusProgressing, false},
		{"progressing->terminating", StatusProgressing, StatusTerminating, true},
		{"progressing->canceling", StatusProgressing, StatusCanceling, false},
		{"progressing->hangup", StatusProgressing, StatusHangup, true},

		{"terminating->dialing", StatusTerminating, StatusDialing, false},
		{"terminating->ringing", StatusTerminating, StatusRinging, false},
		{"terminating->progressing", StatusTerminating, StatusProgressing, false},
		{"terminating->terminating", StatusTerminating, StatusTerminating, false},
		{"terminating->canceling", StatusTerminating, StatusCanceling, false},
		{"terminating->hangup", StatusTerminating, StatusHangup, true},

		{"canceling->dialing", StatusCanceling, StatusDialing, false},
		{"canceling->ringing", StatusCanceling, StatusRinging, false},
		{"canceling->progressing", StatusCanceling, StatusProgressing, false},
		{"canceling->terminating", StatusCanceling, StatusTerminating, false},
		{"canceling->canceling", StatusCanceling, StatusCanceling, false},
		{"canceling->hangup", StatusCanceling, StatusHangup, true},

		{"hangup->dialing", StatusHangup, StatusDialing, false},
		{"hangup->ringing", StatusHangup, StatusRinging, false},
		{"hangup->progressing", StatusHangup, StatusProgressing, false},
		{"hangup->terminating", StatusHangup, StatusTerminating, false},
		{"hangup->canceling", StatusHangup, StatusCanceling, false},
		{"hangup->hangup", StatusHangup, StatusHangup, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := IsUpdatableStatus(tt.oldStatus, tt.newStatus); ret != tt.expect {
				t.Errorf("Wrong match. expect: %t, got: %t", tt.expect, ret)
			}
		})
	}
}

func TestGetStatusByChannelState(t *testing.T) {
	type test struct {
		name         string
		chanState    ari.ChannelState
		expectStatus Status
	}

	tests := []test{
		{"state down", ari.ChannelStateDown, StatusDialing},
		{"state rsrvd", ari.ChannelStateRsrvd, StatusDialing},
		{"state offhook", ari.ChannelStateOffHook, StatusDialing},
		{"state dialing", ari.ChannelStateDialing, StatusDialing},
		{"state busy", ari.ChannelStateBusy, StatusDialing},
		{"state dialing off hook", ari.ChannelStateDialingOffHook, StatusDialing},
		{"state pre ring", ari.ChannelStatePreRing, StatusDialing},
		{"state unknown", ari.ChannelStateUnknown, StatusDialing},

		{"state ringing", ari.ChannelStateRinging, StatusRinging},
		{"state ring", ari.ChannelStateRing, StatusRinging},

		{"state up", ari.ChannelStateUp, StatusProgressing},
		{"state mute", ari.ChannelStateMute, StatusProgressing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := GetStatusByChannelState(tt.chanState); ret != tt.expectStatus {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectStatus, ret)
			}
		})
	}
}

func TestCalculateHangupBy(t *testing.T) {
	type test struct {
		name           string
		lastStatus     Status
		expectHangupby HangupBy
	}

	tests := []test{
		{"last status dialing", StatusDialing, HangupByRemote},
		{"last status ringing", StatusRinging, HangupByRemote},
		{"last status progressing", StatusProgressing, HangupByRemote},
		{"last status terminating", StatusTerminating, HangupByLocal},
		{"last status canceling", StatusCanceling, HangupByLocal},
		{"last status hangup", StatusHangup, HangupByRemote},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ret := CalculateHangupBy(tt.lastStatus); ret != tt.expectHangupby {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectHangupby, ret)
			}
		})
	}
}

func Test_calculateHangupReasonDirectionIncoming(t *testing.T) {
	type test struct {
		name string

		lastStatus []Status
		cause      []ari.ChannelCause

		expectRes HangupReason
	}

	tests := []test{
		{
			"dialing/ringing with noanswer",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseNoAnswer,
			},

			HangupReasonNoanswer,
		},
		{
			"dialing/ringing with userbusy",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUserBusy,
			},

			HangupReasonBusy,
		},
		{
			"dialing/ringing with others",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUnknown,
				ari.ChannelCauseUnallocated,
				ari.ChannelCauseNoRouteTransitNet,
				ari.ChannelCauseNoRouteDestination,
				ari.ChannelCauseMisdialedTrunkPrefix,
				ari.ChannelCauseChannelUnacceptable,
				ari.ChannelCauseCallAwardedDelivered,
				ari.ChannelCausePreEmpted,
				ari.ChannelCauseNumberPortedNotHere,
				ari.ChannelCauseNormalClearing,
				ari.ChannelCauseNoUserResponse,
				ari.ChannelCauseSubscriberAbsent,
				ari.ChannelCauseCallRejected,
				ari.ChannelCauseNumberChanged,
				ari.ChannelCauseRedirectedToNewDestination,
				ari.ChannelCauseAnsweredElsewhere,
				ari.ChannelCauseDestinatioOutOfOrder,
				ari.ChannelCauseInvalidNumberFormat,
				ari.ChannelCauseFacilityRejected,
				ari.ChannelCauseResponseToStatusEnquiry,
				ari.ChannelCauseNormalUnspecified,
				ari.ChannelCauseNormalCircuitCongestion,
				ari.ChannelCauseNetworkOutOfOrder,
				ari.ChannelCauseNormalTemporaryFailure,
				ari.ChannelCauseSwitchCongestion,
				ari.ChannelCauseAccessInfoDiscarded,
				ari.ChannelCauseRequestedChanUnavail,
				ari.ChannelCauseFacilityNotSubscribed,
				ari.ChannelCauseOutgoingCallBarred,
				ari.ChannelCauseIncomingCallBarred,
				ari.ChannelCauseBearerCapabilityNotauth,
				ari.ChannelCauseBearerCapabilityNotavail,
				ari.ChannelCauseBearerCapabilityNotimpl,
				ari.ChannelCauseChanNotImplemented,
				ari.ChannelCauseFacilityNotImplemented,
				ari.ChannelCauseInvalidCallReference,
				ari.ChannelCauseIncompatibleDestination,
				ari.ChannelCauseInvalidMsgUnspecified,
				ari.ChannelCauseMandatoryIeMissing,
				ari.ChannelCauseMessageTypeNonexist,
				ari.ChannelCauseWrongMessage,
				ari.ChannelCauseIeNonexist,
				ari.ChannelCauseInvalidIeContents,
				ari.ChannelCauseWrongCallState,
				ari.ChannelCauseRecoveryIeTimerExpire,
				ari.ChannelCauseMandatoryIeLengthError,
				ari.ChannelCauseProtocolError,

				ari.ChannelCauseInterworking,

				ari.ChannelCauseCallDurationTimeout,
				ari.ChannelCauseCallAMD,
			},

			HangupReasonNormal,
		},
		{
			"progressing with call duration timeout",

			[]Status{
				StatusProgressing,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseCallDurationTimeout,
			},

			HangupReasonTimeout,
		},
		{
			"progressing with others",

			[]Status{
				StatusProgressing,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUnknown,
				ari.ChannelCauseUnallocated,
				ari.ChannelCauseNoRouteTransitNet,
				ari.ChannelCauseNoRouteDestination,

				ari.ChannelCauseMisdialedTrunkPrefix,
				ari.ChannelCauseChannelUnacceptable,
				ari.ChannelCauseCallAwardedDelivered,
				ari.ChannelCausePreEmpted,

				ari.ChannelCauseNumberPortedNotHere,

				ari.ChannelCauseNormalClearing,
				ari.ChannelCauseUserBusy,
				ari.ChannelCauseNoUserResponse,
				ari.ChannelCauseNoAnswer,
				ari.ChannelCauseSubscriberAbsent,
				ari.ChannelCauseCallRejected,
				ari.ChannelCauseNumberChanged,
				ari.ChannelCauseRedirectedToNewDestination,

				ari.ChannelCauseAnsweredElsewhere,
				ari.ChannelCauseDestinatioOutOfOrder,
				ari.ChannelCauseInvalidNumberFormat,
				ari.ChannelCauseFacilityRejected,
				ari.ChannelCauseResponseToStatusEnquiry,
				ari.ChannelCauseNormalUnspecified,

				ari.ChannelCauseNormalCircuitCongestion,

				ari.ChannelCauseNetworkOutOfOrder,

				ari.ChannelCauseNormalTemporaryFailure,
				ari.ChannelCauseSwitchCongestion,
				ari.ChannelCauseAccessInfoDiscarded,
				ari.ChannelCauseRequestedChanUnavail,

				ari.ChannelCauseFacilityNotSubscribed,

				ari.ChannelCauseOutgoingCallBarred,

				ari.ChannelCauseIncomingCallBarred,

				ari.ChannelCauseBearerCapabilityNotauth,
				ari.ChannelCauseBearerCapabilityNotavail,

				ari.ChannelCauseBearerCapabilityNotimpl,
				ari.ChannelCauseChanNotImplemented,

				ari.ChannelCauseFacilityNotImplemented,

				ari.ChannelCauseInvalidCallReference,

				ari.ChannelCauseIncompatibleDestination,

				ari.ChannelCauseInvalidMsgUnspecified,
				ari.ChannelCauseMandatoryIeMissing,
				ari.ChannelCauseMessageTypeNonexist,
				ari.ChannelCauseWrongMessage,
				ari.ChannelCauseIeNonexist,
				ari.ChannelCauseInvalidIeContents,
				ari.ChannelCauseWrongCallState,
				ari.ChannelCauseRecoveryIeTimerExpire,
				ari.ChannelCauseMandatoryIeLengthError,

				ari.ChannelCauseProtocolError,

				ari.ChannelCauseInterworking,

				ari.ChannelCauseCallAMD,
			},

			HangupReasonNormal,
		},
		{
			"StatusCanceling/StatusTerminating/StatusHangup with all",

			[]Status{
				StatusCanceling,
				StatusTerminating,
				StatusHangup,
			},
			ari.ChannelCauseAll,

			HangupReasonNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, status := range tt.lastStatus {
				for _, cause := range tt.cause {

					res := calculateHangupReasonDirectionIncoming(status, cause)
					if res != tt.expectRes {
						t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
					}
				}
			}
		})
	}
}

func Test_calculateHangupReasonDirectionOutgoing(t *testing.T) {
	type test struct {
		name string

		lastStatuses []Status
		causes       []ari.ChannelCause

		expectRes HangupReason
	}

	tests := []test{
		{
			"dialing/ringing with noanswer/callrejected",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseNoAnswer,
				ari.ChannelCauseCallRejected,
			},

			HangupReasonNoanswer,
		},
		{
			"dialing/ringing with userbusy",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUserBusy,
			},

			HangupReasonBusy,
		},
		{
			"dialing/ringing with normalclearing/answeredelsewhere",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseNormalClearing,
				ari.ChannelCauseAnsweredElsewhere,
			},

			HangupReasonNormal,
		},
		{
			"dialing/ringing with unknown",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUnknown,
			},

			HangupReasonDialout,
		},
		{
			"dialing/ringing with others",

			[]Status{
				StatusDialing,
				StatusRinging,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUnallocated,
				ari.ChannelCauseNoRouteTransitNet,
				ari.ChannelCauseNoRouteDestination,

				ari.ChannelCauseMisdialedTrunkPrefix,
				ari.ChannelCauseChannelUnacceptable,
				ari.ChannelCauseCallAwardedDelivered,
				ari.ChannelCausePreEmpted,

				ari.ChannelCauseNumberPortedNotHere,

				ari.ChannelCauseNoUserResponse,
				ari.ChannelCauseSubscriberAbsent,
				ari.ChannelCauseNumberChanged,
				ari.ChannelCauseRedirectedToNewDestination,

				ari.ChannelCauseDestinatioOutOfOrder,
				ari.ChannelCauseInvalidNumberFormat,
				ari.ChannelCauseFacilityRejected,
				ari.ChannelCauseResponseToStatusEnquiry,
				ari.ChannelCauseNormalUnspecified,

				ari.ChannelCauseNormalCircuitCongestion,

				ari.ChannelCauseNetworkOutOfOrder,

				ari.ChannelCauseNormalTemporaryFailure,
				ari.ChannelCauseSwitchCongestion,
				ari.ChannelCauseAccessInfoDiscarded,
				ari.ChannelCauseRequestedChanUnavail,

				ari.ChannelCauseFacilityNotSubscribed,

				ari.ChannelCauseOutgoingCallBarred,

				ari.ChannelCauseIncomingCallBarred,

				ari.ChannelCauseBearerCapabilityNotauth,
				ari.ChannelCauseBearerCapabilityNotavail,

				ari.ChannelCauseBearerCapabilityNotimpl,
				ari.ChannelCauseChanNotImplemented,

				ari.ChannelCauseFacilityNotImplemented,

				ari.ChannelCauseInvalidCallReference,

				ari.ChannelCauseIncompatibleDestination,

				ari.ChannelCauseInvalidMsgUnspecified,
				ari.ChannelCauseMandatoryIeMissing,
				ari.ChannelCauseMessageTypeNonexist,
				ari.ChannelCauseWrongMessage,
				ari.ChannelCauseIeNonexist,
				ari.ChannelCauseInvalidIeContents,
				ari.ChannelCauseWrongCallState,
				ari.ChannelCauseRecoveryIeTimerExpire,
				ari.ChannelCauseMandatoryIeLengthError,

				ari.ChannelCauseProtocolError,

				ari.ChannelCauseInterworking,

				ari.ChannelCauseCallDurationTimeout,
				ari.ChannelCauseCallAMD,
			},

			HangupReasonFailed,
		},
		{
			"StatusProgressing with call timeout",

			[]Status{
				StatusProgressing,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseCallDurationTimeout,
			},

			HangupReasonTimeout,
		},
		{
			"StatusProgressing with others",

			[]Status{
				StatusProgressing,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUnknown,
				ari.ChannelCauseUnallocated,
				ari.ChannelCauseNoRouteTransitNet,
				ari.ChannelCauseNoRouteDestination,

				ari.ChannelCauseMisdialedTrunkPrefix,
				ari.ChannelCauseChannelUnacceptable,
				ari.ChannelCauseCallAwardedDelivered,
				ari.ChannelCausePreEmpted,

				ari.ChannelCauseNumberPortedNotHere,

				ari.ChannelCauseNormalClearing,
				ari.ChannelCauseUserBusy,
				ari.ChannelCauseNoUserResponse,
				ari.ChannelCauseNoAnswer,
				ari.ChannelCauseSubscriberAbsent,
				ari.ChannelCauseCallRejected,
				ari.ChannelCauseNumberChanged,
				ari.ChannelCauseRedirectedToNewDestination,

				ari.ChannelCauseAnsweredElsewhere,
				ari.ChannelCauseDestinatioOutOfOrder,
				ari.ChannelCauseInvalidNumberFormat,
				ari.ChannelCauseFacilityRejected,
				ari.ChannelCauseResponseToStatusEnquiry,
				ari.ChannelCauseNormalUnspecified,

				ari.ChannelCauseNormalCircuitCongestion,

				ari.ChannelCauseNetworkOutOfOrder,

				ari.ChannelCauseNormalTemporaryFailure,
				ari.ChannelCauseSwitchCongestion,
				ari.ChannelCauseAccessInfoDiscarded,
				ari.ChannelCauseRequestedChanUnavail,

				ari.ChannelCauseFacilityNotSubscribed,

				ari.ChannelCauseOutgoingCallBarred,

				ari.ChannelCauseIncomingCallBarred,

				ari.ChannelCauseBearerCapabilityNotauth,
				ari.ChannelCauseBearerCapabilityNotavail,

				ari.ChannelCauseBearerCapabilityNotimpl,
				ari.ChannelCauseChanNotImplemented,

				ari.ChannelCauseFacilityNotImplemented,

				ari.ChannelCauseInvalidCallReference,

				ari.ChannelCauseIncompatibleDestination,

				ari.ChannelCauseInvalidMsgUnspecified,
				ari.ChannelCauseMandatoryIeMissing,
				ari.ChannelCauseMessageTypeNonexist,
				ari.ChannelCauseWrongMessage,
				ari.ChannelCauseIeNonexist,
				ari.ChannelCauseInvalidIeContents,
				ari.ChannelCauseWrongCallState,
				ari.ChannelCauseRecoveryIeTimerExpire,
				ari.ChannelCauseMandatoryIeLengthError,

				ari.ChannelCauseProtocolError,

				ari.ChannelCauseInterworking,

				ari.ChannelCauseCallAMD,
			},

			HangupReasonNormal,
		},
		{
			"StatusCanceling with all",

			[]Status{
				StatusCanceling,
			},
			ari.ChannelCauseAll,

			HangupReasonCanceled,
		},
		{
			"/StatusTerminating with amd",

			[]Status{
				StatusTerminating,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseCallAMD,
			},

			HangupReasonAMD,
		},
		{
			"StatusTerminating with others",

			[]Status{
				StatusTerminating,
			},
			[]ari.ChannelCause{
				ari.ChannelCauseUnknown,
				ari.ChannelCauseUnallocated,
				ari.ChannelCauseNoRouteTransitNet,
				ari.ChannelCauseNoRouteDestination,

				ari.ChannelCauseMisdialedTrunkPrefix,
				ari.ChannelCauseChannelUnacceptable,
				ari.ChannelCauseCallAwardedDelivered,
				ari.ChannelCausePreEmpted,

				ari.ChannelCauseNumberPortedNotHere,

				ari.ChannelCauseNormalClearing,
				ari.ChannelCauseUserBusy,
				ari.ChannelCauseNoUserResponse,
				ari.ChannelCauseNoAnswer,
				ari.ChannelCauseSubscriberAbsent,
				ari.ChannelCauseCallRejected,
				ari.ChannelCauseNumberChanged,
				ari.ChannelCauseRedirectedToNewDestination,

				ari.ChannelCauseAnsweredElsewhere,
				ari.ChannelCauseDestinatioOutOfOrder,
				ari.ChannelCauseInvalidNumberFormat,
				ari.ChannelCauseFacilityRejected,
				ari.ChannelCauseResponseToStatusEnquiry,
				ari.ChannelCauseNormalUnspecified,

				ari.ChannelCauseNormalCircuitCongestion,

				ari.ChannelCauseNetworkOutOfOrder,

				ari.ChannelCauseNormalTemporaryFailure,
				ari.ChannelCauseSwitchCongestion,
				ari.ChannelCauseAccessInfoDiscarded,
				ari.ChannelCauseRequestedChanUnavail,

				ari.ChannelCauseFacilityNotSubscribed,

				ari.ChannelCauseOutgoingCallBarred,

				ari.ChannelCauseIncomingCallBarred,

				ari.ChannelCauseBearerCapabilityNotauth,
				ari.ChannelCauseBearerCapabilityNotavail,

				ari.ChannelCauseBearerCapabilityNotimpl,
				ari.ChannelCauseChanNotImplemented,

				ari.ChannelCauseFacilityNotImplemented,

				ari.ChannelCauseInvalidCallReference,

				ari.ChannelCauseIncompatibleDestination,

				ari.ChannelCauseInvalidMsgUnspecified,
				ari.ChannelCauseMandatoryIeMissing,
				ari.ChannelCauseMessageTypeNonexist,
				ari.ChannelCauseWrongMessage,
				ari.ChannelCauseIeNonexist,
				ari.ChannelCauseInvalidIeContents,
				ari.ChannelCauseWrongCallState,
				ari.ChannelCauseRecoveryIeTimerExpire,
				ari.ChannelCauseMandatoryIeLengthError,

				ari.ChannelCauseProtocolError,

				ari.ChannelCauseInterworking,

				ari.ChannelCauseCallDurationTimeout,
			},

			HangupReasonNormal,
		},
		{
			"StatusHangup with all",

			[]Status{
				StatusHangup,
			},
			ari.ChannelCauseAll,

			HangupReasonNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			for _, status := range tt.lastStatuses {
				for _, cause := range tt.causes {

					res := calculateHangupReasonDirectionOutgoing(status, cause)
					if res != tt.expectRes {
						t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
					}
				}
			}
		})
	}
}
