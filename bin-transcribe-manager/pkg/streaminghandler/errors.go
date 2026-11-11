package streaminghandler

import (
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// ErrSTTNotConfiguredReason is the VoipbinError reason returned by Start when no STT provider
// (GCP or AWS) is available at the time of the request. The service keeps running so that every
// other transcribe-manager capability stays available; only the live-streaming transcribe path
// is unavailable. Matches this codebase's structured-error convention (see
// pkg/transcribehandler/start.go's TRANSCRIBE_ALREADY_PROGRESSING for another example) so API
// callers get a typed, domain/reason-tagged error instead of an opaque 500.
//
// Exported so that other packages can match on this specific reason instead of on Status alone.
// pkg/transcribehandler/stop.go's isSafeToConsiderStopped still accepts Status == StatusUnavailable
// && Reason == ErrSTTNotConfiguredReason as "safe to consider already stopped" (a defensive
// branch: Stop of the streaming handler does not return it, an unknown session is
// STREAMING_NOT_FOUND). StatusUnavailable by itself is a general-purpose "transient, retry later"
// status, so the Reason match is required to avoid misclassifying a live session as already gone.
const ErrSTTNotConfiguredReason = "STT_NOT_CONFIGURED"

// newErrSTTNotConfigured returns a fresh VoipbinError for Start to return when no provider is
// available.
func newErrSTTNotConfigured() error {
	return cerrors.Unavailable(
		commonoutline.ServiceNameTranscribeManager,
		ErrSTTNotConfiguredReason,
		"No STT provider (GCP or AWS) is configured on this instance. Live-streaming transcribe is unavailable; all other transcribe-manager functionality is unaffected.",
	)
}

// ErrSTTKeyUnavailableReason is the VoipbinError reason returned by Start when no GCP client
// exists and the AWS credential could not be read (a database error, or a stored secret that can
// not be decrypted). It differs from ErrSTTNotConfiguredReason because the credential may exist:
// a read that failed is not "not configured", and the caller may retry. Stop never returns it
// (Stop does not read the credential).
const ErrSTTKeyUnavailableReason = "STT_KEY_UNAVAILABLE"

// newErrSTTKeyUnavailable returns a fresh VoipbinError for Start to return when the AWS
// credential could not be read and no other provider is available. It holds no key text.
func newErrSTTKeyUnavailable() error {
	return cerrors.Unavailable(
		commonoutline.ServiceNameTranscribeManager,
		ErrSTTKeyUnavailableReason,
		"The STT credential could not be read and no other STT provider is available. Live-streaming transcribe is unavailable; retry later.",
	)
}
