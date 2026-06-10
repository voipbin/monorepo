package server

import (
	"net/url"
	"strings"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

// validateWebhookMethod validates a caller-supplied per-activeflow webhook method.
// An empty method is allowed (no per-activeflow webhook method). Otherwise the method
// must be one of POST, GET, PUT or DELETE. It returns the typed WebhookMethod on success.
func validateWebhookMethod(method string) (fmactiveflow.WebhookMethod, *cerrors.VoipbinError) {
	switch fmactiveflow.WebhookMethod(method) {
	case fmactiveflow.WebhookMethodNone,
		fmactiveflow.WebhookMethodPost,
		fmactiveflow.WebhookMethodGet,
		fmactiveflow.WebhookMethodPut,
		fmactiveflow.WebhookMethodDelete:
		return fmactiveflow.WebhookMethod(method), nil
	default:
		return fmactiveflow.WebhookMethodNone, cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_WEBHOOK_METHOD",
			"webhook_method must be one of POST, GET, PUT or DELETE",
		)
	}
}

// validateWebhookURI validates a caller-supplied per-activeflow webhook URI.
// An empty URI is allowed (no per-activeflow webhook destination). Otherwise the URI
// must be a syntactically valid absolute http or https URL with a host.
func validateWebhookURI(uri string) *cerrors.VoipbinError {
	if uri == "" {
		return nil
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_WEBHOOK_URI",
			"webhook_uri must be a valid http or https URL",
		)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Host == "" {
		return cerrors.InvalidArgument(
			commonoutline.ServiceNameAPIManager,
			"INVALID_WEBHOOK_URI",
			"webhook_uri must be a valid http or https URL",
		)
	}

	return nil
}
