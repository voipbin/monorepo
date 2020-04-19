package ari

// hangup cause codes
const (
	HangupUnallocated        = 1 // code -> sip: 404
	HangupNoRouteTransitNet  = 2 // code -> sip: 404
	HangupNoRouteDestination = 3 // code -> sip: 404

	HangupMisdialedTrunkPrefix = 5 // code -> sip:
	HangupChannelUnacceptable  = 6 // code -> sip:
	HangupCallAwardedDelivered = 7 // code -> sip:
	HangupPreEmpted            = 8 // code -> sip:

	HangupNumberPortedNotHere = 14 // code -> sip:

	HangupNormalClearing             = 16 // code -> sip:
	HangupUserBusy                   = 17 // code -> sip: 486
	HangupNoUserResponse             = 18 // code -> sip: 408
	HangupNoAnswer                   = 19 // code -> sip: 480
	HangupSubscriberAbsent           = 20 // code -> sip: 480
	HangupCallRejected               = 21 // code -> sip: 403
	HangupNumberChanged              = 22 // code -> sip: 410
	HangupRedirectedToNewDestination = 23 // code -> sip:

	HangupAnsweredElsewhere       = 26 // code -> sip:
	HangupDestinatioOutOfOrder    = 27 // code -> sip: 502
	HangupInvalidNumberFormat     = 28 // code -> sip: 484
	HangupFacilityRejected        = 29 // code -> sip: 501
	HangupResponseToStatusEnquiry = 30 // code -> sip:
	HangupNormalUnspecified       = 31 // code -> sip: 480

	HangupNormalCircuitCongestion = 34 // code -> sip: 503

	HangupNetworkOutOfOrder = 38 // code -> sip: 500

	HangupNormalTemporaryFailure = 41 // code -> sip:
	HangupSwitchCongestion       = 42 // code -> sip: 503
	HangupAccessInfoDiscarded    = 43 // code -> sip:
	HangupRequestedChanUnavail   = 44 // code -> sip:

	HangupFacilityNotSubscribed = 50 // code -> sip:

	HangupOutgoingCallBarred = 52 // code -> sip:

	HangupIncomingCallBarred = 54 // code -> sip:

	HangupBearerCapabilityNotauth  = 57 // code -> sip:
	HangupBearerCapabilityNotavail = 58 // code -> sip: 488

	HangupBearerCapabilityNotimpl = 65 // code -> sip:
	HangupChanNotImplemented      = 66 // code -> sip: 503

	HangupFacilityNotImplemented = 69 // code -> sip:

	HangupInvalidCallReference = 81 // code -> sip:

	HangupIncompatibleDestination = 88 // code -> sip:

	HangupInvalidMsgUnspecified  = 95  // code -> sip:
	HangupMandatoryIeMissing     = 96  // code -> sip:
	HangupMessageTypeNonexist    = 97  // code -> sip:
	HangupWrongMessage           = 98  // code -> sip:
	HangupIeNonexist             = 99  // code -> sip:
	HangupInvalidIeContents      = 100 // code -> sip:
	HangupWrongCallState         = 101 // code -> sip:
	HangupRecoveryIeTimerExpire  = 102 // code -> sip:
	HangupMandatoryIeLengthError = 103 // code -> sip:

	HangupProtocolError = 111 // code -> sip:

	HangupInterworking = 127 // code -> sip: 500
)
