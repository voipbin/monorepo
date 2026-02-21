package ari

// ChannelCause type
type ChannelCause int

// List of ChannelCause type
const (
	ChannelCauseUnknown            ChannelCause = 0
	ChannelCauseUnallocated        ChannelCause = 1 // code -> sip: 404
	ChannelCauseNoRouteTransitNet  ChannelCause = 2 // code -> sip: 404
	ChannelCauseNoRouteDestination ChannelCause = 3 // code -> sip: 404

	ChannelCauseMisdialedTrunkPrefix ChannelCause = 5 // code -> sip:
	ChannelCauseChannelUnacceptable  ChannelCause = 6 // code -> sip:
	ChannelCauseCallAwardedDelivered ChannelCause = 7 // code -> sip:
	ChannelCausePreEmpted            ChannelCause = 8 // code -> sip:

	ChannelCauseNumberPortedNotHere ChannelCause = 14 // code -> sip:

	ChannelCauseNormalClearing             ChannelCause = 16 // code -> sip:
	ChannelCauseUserBusy                   ChannelCause = 17 // code -> sip: 486
	ChannelCauseNoUserResponse             ChannelCause = 18 // code -> sip: 408
	ChannelCauseNoAnswer                   ChannelCause = 19 // code -> sip: 480
	ChannelCauseSubscriberAbsent           ChannelCause = 20 // code -> sip: 480
	ChannelCauseCallRejected               ChannelCause = 21 // code -> sip: 403
	ChannelCauseNumberChanged              ChannelCause = 22 // code -> sip: 410
	ChannelCauseRedirectedToNewDestination ChannelCause = 23 // code -> sip:

	ChannelCauseAnsweredElsewhere       ChannelCause = 26 // code -> sip:
	ChannelCauseDestinatioOutOfOrder    ChannelCause = 27 // code -> sip: 502
	ChannelCauseInvalidNumberFormat     ChannelCause = 28 // code -> sip: 484
	ChannelCauseFacilityRejected        ChannelCause = 29 // code -> sip: 501
	ChannelCauseResponseToStatusEnquiry ChannelCause = 30 // code -> sip:
	ChannelCauseNormalUnspecified       ChannelCause = 31 // code -> sip: 480

	ChannelCauseNormalCircuitCongestion ChannelCause = 34 // code -> sip: 503

	ChannelCauseNetworkOutOfOrder ChannelCause = 38 // code -> sip: 500

	ChannelCauseNormalTemporaryFailure ChannelCause = 41 // code -> sip:
	ChannelCauseSwitchCongestion       ChannelCause = 42 // code -> sip: 503
	ChannelCauseAccessInfoDiscarded    ChannelCause = 43 // code -> sip:
	ChannelCauseRequestedChanUnavail   ChannelCause = 44 // code -> sip:

	ChannelCauseFacilityNotSubscribed ChannelCause = 50 // code -> sip:

	ChannelCauseOutgoingCallBarred ChannelCause = 52 // code -> sip:

	ChannelCauseIncomingCallBarred ChannelCause = 54 // code -> sip:

	ChannelCauseBearerCapabilityNotauth  ChannelCause = 57 // code -> sip:
	ChannelCauseBearerCapabilityNotavail ChannelCause = 58 // code -> sip: 488

	ChannelCauseBearerCapabilityNotimpl ChannelCause = 65 // code -> sip:
	ChannelCauseChanNotImplemented      ChannelCause = 66 // code -> sip: 503

	ChannelCauseFacilityNotImplemented ChannelCause = 69 // code -> sip:

	ChannelCauseInvalidCallReference ChannelCause = 81 // code -> sip:

	ChannelCauseIncompatibleDestination ChannelCause = 88 // code -> sip:

	ChannelCauseInvalidMsgUnspecified  ChannelCause = 95  // code -> sip:
	ChannelCauseMandatoryIeMissing     ChannelCause = 96  // code -> sip:
	ChannelCauseMessageTypeNonexist    ChannelCause = 97  // code -> sip:
	ChannelCauseWrongMessage           ChannelCause = 98  // code -> sip:
	ChannelCauseIeNonexist             ChannelCause = 99  // code -> sip:
	ChannelCauseInvalidIeContents      ChannelCause = 100 // code -> sip:
	ChannelCauseWrongCallState         ChannelCause = 101 // code -> sip:
	ChannelCauseRecoveryIeTimerExpire  ChannelCause = 102 // code -> sip:
	ChannelCauseMandatoryIeLengthError ChannelCause = 103 // code -> sip:

	ChannelCauseProtocolError ChannelCause = 111 // code -> sip:

	ChannelCauseInterworking ChannelCause = 127 // code -> sip: 500

	/// VoIPBIN defined cause code.
	ChannelCauseCallDurationTimeout ChannelCause = 200 // call progress timeout
	ChannelCauseCallAMD             ChannelCause = 201 // call's amd hangup
)

// ChannelCauseAll list of all ChannelCauses
var ChannelCauseAll = []ChannelCause{
	ChannelCauseUnknown,
	ChannelCauseUnallocated,
	ChannelCauseNoRouteTransitNet,
	ChannelCauseNoRouteDestination,

	ChannelCauseMisdialedTrunkPrefix,
	ChannelCauseChannelUnacceptable,
	ChannelCauseCallAwardedDelivered,
	ChannelCausePreEmpted,

	ChannelCauseNumberPortedNotHere,

	ChannelCauseNormalClearing,
	ChannelCauseUserBusy,
	ChannelCauseNoUserResponse,
	ChannelCauseNoAnswer,
	ChannelCauseSubscriberAbsent,
	ChannelCauseCallRejected,
	ChannelCauseNumberChanged,
	ChannelCauseRedirectedToNewDestination,

	ChannelCauseAnsweredElsewhere,
	ChannelCauseDestinatioOutOfOrder,
	ChannelCauseInvalidNumberFormat,
	ChannelCauseFacilityRejected,
	ChannelCauseResponseToStatusEnquiry,
	ChannelCauseNormalUnspecified,

	ChannelCauseNormalCircuitCongestion,

	ChannelCauseNetworkOutOfOrder,

	ChannelCauseNormalTemporaryFailure,
	ChannelCauseSwitchCongestion,
	ChannelCauseAccessInfoDiscarded,
	ChannelCauseRequestedChanUnavail,

	ChannelCauseFacilityNotSubscribed,

	ChannelCauseOutgoingCallBarred,

	ChannelCauseIncomingCallBarred,

	ChannelCauseBearerCapabilityNotauth,
	ChannelCauseBearerCapabilityNotavail,

	ChannelCauseBearerCapabilityNotimpl,
	ChannelCauseChanNotImplemented,

	ChannelCauseFacilityNotImplemented,

	ChannelCauseInvalidCallReference,

	ChannelCauseIncompatibleDestination,

	ChannelCauseInvalidMsgUnspecified,
	ChannelCauseMandatoryIeMissing,
	ChannelCauseMessageTypeNonexist,
	ChannelCauseWrongMessage,
	ChannelCauseIeNonexist,
	ChannelCauseInvalidIeContents,
	ChannelCauseWrongCallState,
	ChannelCauseRecoveryIeTimerExpire,
	ChannelCauseMandatoryIeLengthError,

	ChannelCauseProtocolError,

	ChannelCauseInterworking,

	ChannelCauseCallDurationTimeout,
	ChannelCauseCallAMD,
}
