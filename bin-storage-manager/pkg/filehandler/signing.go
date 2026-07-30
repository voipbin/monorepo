package filehandler

import (
	stderrors "errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameStorageManager)

// downloadURIFailureReason labels promDownloadURIFailureTotal.
const (
	downloadURIFailureReasonNotConfigured = "not_configured" // no signing credential at all
	downloadURIFailureReasonSignerError   = "signer_error"   // a credential exists but the signer rejected it
)

var (
	// promSigningAvailable is 1 when a usable signing credential was loaded at startup
	// and 0 otherwise. Without it the degraded mode is only visible in the boot log,
	// which is not alertable.
	promSigningAvailable = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "signing_available",
			Help:      "1 when a GCS signing credential is configured, 0 when signed download URLs are unavailable",
		},
	)

	// promDownloadURIFailureTotal counts file creations that had to persist an empty
	// URIDownload. A rotated-to-broken key would otherwise silently produce unusable
	// file records on every upload.
	promDownloadURIFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "download_uri_failure_total",
			Help:      "Number of file creations persisted without a download URI because signing failed",
		},
		[]string{"reason"},
	)
)

func init() {
	prometheus.MustRegister(
		promSigningAvailable,
		promDownloadURIFailureTotal,
	)
}

// errSigningNotConfiguredReason is the VoipbinError reason returned by the signing
// dependent paths when no GCS signing credential was configured at startup
// (GOOGLE_APPLICATION_CREDENTIALS unset). The service keeps running so that every
// other storage-manager capability stays available - file get/list/delete, account
// bookkeeping and the customer-deleted cascade all work without a signing key, and
// file Create persists the record with an empty URIDownload. Only the signed
// download URL paths are unavailable. Matches this codebase's structured-error
// convention (see pkg/filehandler/file.go's FILE_NOT_FOUND, and
// bin-transcribe-manager's STT_NOT_CONFIGURED for the same degradation shape) so API
// callers get a typed, domain/reason-tagged error instead of an opaque 500.
const errSigningNotConfiguredReason = "SIGNING_NOT_CONFIGURED"

// newErrSigningNotConfigured returns a fresh VoipbinError for the signing-dependent
// paths to return when no private key is available.
func newErrSigningNotConfigured() error {
	return cerrors.Unavailable(
		commonoutline.ServiceNameStorageManager,
		errSigningNotConfiguredReason,
		"No GCS signing credential is configured on this instance. Signed download URLs are unavailable; all other storage-manager functionality is unaffected.",
	)
}

// isSigningNotConfigured reports whether err is the structured
// signing-not-configured error, walking any wrapping.
func isSigningNotConfigured(err error) bool {
	var ve *cerrors.VoipbinError
	return stderrors.As(err, &ve) && ve.Reason == errSigningNotConfiguredReason
}

// logDownloadURIFailure logs a download-URI generation failure at the level it
// deserves. On a deployment with no signing credential this fires on every download
// attempt, so it is the expected steady state there rather than an incident and is
// logged as a warning. Anything else (a rejected key, a transient signer failure)
// stays at error level.
func logDownloadURIFailure(log *logrus.Entry, err error) {
	if isSigningNotConfigured(err) {
		log.Warnf("Could not generate download URI. Signing is not configured. err: %v", err)
		return
	}

	log.Errorf("Could not generate download URI. err: %v", err)
}
