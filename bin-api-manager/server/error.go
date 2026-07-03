package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"monorepo/bin-api-manager/lib/apierror"
	"monorepo/bin-api-manager/lib/middleware"
	commonoutline "monorepo/bin-common-handler/models/outline"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// abortWithError writes the VoipbinError as a JSON body, sets the
// correct HTTP status code, and aborts the Gin context. The request
// ID is read from middleware.RequestIDFromContext, so that middleware
// must run before any handler that calls this.
//
// The external envelope omits the internal Domain field — see
// bin-api-manager/lib/apierror for the boundary.
//
// A nil VoipbinError falls back to a StatusInternal response so the
// helper never panics on a caller oversight (handled inside
// apierror.EnvelopeFor).
func abortWithError(c *gin.Context, e *cerrors.VoipbinError) {
	status := cerrors.StatusInternal
	if e != nil {
		status = e.Status
	}
	c.AbortWithStatusJSON(
		cerrors.HTTPStatusFor(status),
		apierror.EnvelopeFor(e, middleware.RequestIDFromContext(c)),
	)
}

// abortWithServiceError runs any error returned from servicehandler
// through the translator (see server/error_translate.go), then aborts
// with the resulting VoipbinError.
func abortWithServiceError(c *gin.Context, err error) {
	abortWithError(c, translateToVoipbinError(err))
}

// reBindingParamName extracts the parameter name from an oapi-codegen
// binding-error message. Matches only the outer, fixed prefix strings
// oapi-codegen's generated wrapper code emits — never any inner
// %w-wrapped text — so a wrapped parser error that happens to contain
// look-alike words never confuses extraction. See docs/plans/2026-07-04-
// standardize-binding-error-envelope.md "Message construction" for the
// full design rationale, including why [\w-]+ (not \w+) is required to
// avoid truncating hyphenated parameter names (e.g. billing-id).
var reBindingParamName = regexp.MustCompile(`(?:Invalid format for parameter |Query argument |Header parameter )([\w-]+)`)

// extractBindingParamName returns the parameter name embedded in an
// oapi-codegen binding-error message, or ok=false if the text doesn't
// match either known oapi-codegen format. The extracted name is used
// only for cosmetic message text — it never drives a reason-code
// decision (see BindingErrorHandler).
func extractBindingParamName(errText string) (string, bool) {
	m := reBindingParamName.FindStringSubmatch(errText)
	if len(m) != 2 {
		return "", false
	}
	return m[1], true
}

// classifyBindingError maps an oapi-codegen binding-error message to a
// (reason, message) pair. The two oapi-codegen formats are fixed and
// mutually exclusive (verified against every siw.ErrorHandler( call
// site in gens/openapi_server/gen.go):
//   - "Invalid format for parameter <name>: %w"                  -> malformed value
//   - "Query argument <name> is required, but not found"         -> missing required param
//
// HasPrefix is checked before Contains so that even if a future
// oapi-codegen version's %w-wrapped inner error text happened to
// contain "is required, but not found", the malformed-value case
// still wins (its own outer text always starts with the first
// prefix) — this ordering is deliberate defense-in-depth, not just
// current-day happenstance. See
// TestBindingErrorReason_PrefixOverlap_MalformedWins.
//
// The extracted parameter name only ever feeds the message text, not
// the reason returned — this is the key structural fix that avoids
// the name-based-classification bugs found in earlier revisions of
// this design (hyphenated names, non-suffixed ID params, missing-vs-
// malformed conflation).
func classifyBindingError(errText string) (reason, message string) {
	switch {
	case strings.HasPrefix(errText, "Invalid format for parameter "):
		if name, ok := extractBindingParamName(errText); ok {
			return "INVALID_REQUEST_PARAMETER", fmt.Sprintf("The parameter %q has an invalid format.", name)
		}
		return "INVALID_REQUEST_PARAMETER", "One or more request parameters have an invalid format."
	case strings.Contains(errText, "is required, but not found"):
		if name, ok := extractBindingParamName(errText); ok {
			return "MISSING_REQUEST_PARAMETER", fmt.Sprintf("The parameter %q is required.", name)
		}
		return "MISSING_REQUEST_PARAMETER", "A required request parameter is missing."
	default:
		return "INVALID_REQUEST_PARAMETER", "One or more request parameters are invalid."
	}
}

// BindingErrorHandler is the oapi-codegen ErrorHandler wired into
// openapi_server.RegisterHandlersWithOptions in cmd/api-manager/main.go.
// It replaces oapi-codegen's default fallback ({"msg": err.Error()})
// with the platform's standard error envelope (apierror.EnvelopeFor),
// so parameter-binding failures caught by the generated wrapper code
// — before any handler runs — look identical on the wire to
// handler-level errors.
//
// See docs/plans/2026-07-04-standardize-binding-error-envelope.md for
// the full design and its review history (ETC-3).
func BindingErrorHandler(c *gin.Context, err error, statusCode int) {
	// Defensive guard: abortWithError derives the actual HTTP status
	// solely from e.Status, not from this statusCode argument. Every
	// current call site in gen.go passes http.StatusBadRequest
	// (verified by grepping every siw.ErrorHandler( call site — this
	// branch is therefore unreachable in production today), but this
	// is generated code we don't control — if a future `go generate`
	// starts passing something else (e.g. 500 on an internal binding
	// panic), fail safe to an INTERNAL envelope instead of silently
	// mislabeling a server-side condition as a 400 client error.
	if statusCode != http.StatusBadRequest {
		logrus.WithFields(logrus.Fields{
			"func":        "BindingErrorHandler",
			"status_code": statusCode,
			"error":       err.Error(),
		}).Warn("oapi-codegen binding error handler invoked with unexpected status code; falling back to INTERNAL.")
		abortWithError(c, cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err))
		return
	}

	reason, message := classifyBindingError(err.Error())
	e := cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, reason, message).Wrap(err)
	abortWithError(c, e)
}
