package server

import (
	"context"
	stderrors "errors"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// translateToVoipbinError maps any error returned from a servicehandler
// into a *VoipbinError. Priority order:
//  1. Typed passthrough (errors.As).
//  2. Sentinel match (errors.Is against serviceerrors.Err* and the bare
//     requesthandler.Err* HTTP-status sentinels).
//  3. Transport-failure detection (context.Canceled / DeadlineExceeded).
//  4. Default: Internal with the original error wrapped as Cause.
//
// The whole function is wrapped in defer recover() so a panic inside
// any branch degrades to INTERNAL rather than dropping the response.
//
// Migration status (2026-04, complete): All RPC traffic from upstream
// managers emits typed *cerrors.VoipbinError on miss (PR2-PR29). The
// api-manager servicehandler layer has been fully migrated to typed
// sentinels — permission checks (PR-perm), validation errors
// (PR-validation), not-found errors (PR-notfound), and the remaining
// state/insufficient/internal/auth strings (PR-final). The legacy
// substring-fallback step has been removed; any unmatched error
// correctly degrades to INTERNAL via the default branch.
//
// Bare status codes: backend services that return a bare status code
// (e.g. simpleResponse(404)) without a typed VoipbinError body produce a
// requesthandler.Err* sentinel (via HttpStatusErrorMap) instead of a
// VoipbinError. Step 2 maps the closed set of these sentinels
// (400/401/402/403/404/409/429/503/500) back to the canonical cerrors
// status, mirroring cerrors.HTTPStatusFor in reverse so the client sees
// the same HTTP code the backend emitted. Statuses outside that set fall
// through to the Default branch (INTERNAL).
func translateToVoipbinError(err error) (out *cerrors.VoipbinError) {
	defer func() {
		if r := recover(); r != nil {
			out = cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.")
		}
	}()

	if err == nil {
		return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.")
	}

	// 1. Typed passthrough.
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		return ve
	}

	// 2. Sentinel match.
	switch {
	case stderrors.Is(err, serviceerrors.ErrAuthenticationRequired):
		return cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required.")
	case stderrors.Is(err, serviceerrors.ErrPermissionDenied):
		return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "You do not have permission to access this resource.")
	case stderrors.Is(err, serviceerrors.ErrDirectAccessNotSupported):
		return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "DIRECT_ACCESS_NOT_SUPPORTED", "Direct access is not supported for this endpoint.")
	case stderrors.Is(err, serviceerrors.ErrNotFound):
		return cerrors.NotFound(commonoutline.ServiceNameAPIManager, "RESOURCE_NOT_FOUND", "The requested resource was not found.")
	case stderrors.Is(err, serviceerrors.ErrInvalidArgument):
		return cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "The request is invalid.")
	// Bare requesthandler HTTP-status sentinels (backend returned a bare
	// status with no typed body). Wrap behavior mirrors the serviceerrors
	// analogues above: 401/403/404 return unwrapped; the rest .Wrap(err) to
	// keep the originating sentinel in the server-side Cause chain.
	case stderrors.Is(err, requesthandler.ErrBadRequest):
		return cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ARGUMENT", "The request contains invalid data.")
	case stderrors.Is(err, requesthandler.ErrUnauthorized):
		return cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required.")
	case stderrors.Is(err, requesthandler.ErrPaymentRequired):
		return cerrors.PaymentRequired(commonoutline.ServiceNameAPIManager, "INSUFFICIENT_BALANCE",
			"Customer balance is below the minimum required for this operation.").Wrap(err)
	case stderrors.Is(err, requesthandler.ErrForbidden):
		return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "You do not have permission to access this resource.")
	case stderrors.Is(err, requesthandler.ErrNotFound):
		return cerrors.NotFound(commonoutline.ServiceNameAPIManager, "RESOURCE_NOT_FOUND", "The requested resource was not found.")
	case stderrors.Is(err, requesthandler.ErrConflict):
		return cerrors.FailedPrecondition(commonoutline.ServiceNameAPIManager, "STATE_INVALID",
			"The operation is invalid for the current resource state.").Wrap(err)
	case stderrors.Is(err, requesthandler.ErrTooManyRequests):
		return cerrors.ResourceExhausted(commonoutline.ServiceNameAPIManager, "RATE_LIMIT_EXCEEDED",
			"Too many requests. Please retry later.").Wrap(err)
	case stderrors.Is(err, requesthandler.ErrServiceUnavailable):
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "SERVICE_UNAVAILABLE",
			"An upstream service is temporarily unavailable.").Wrap(err)
	case stderrors.Is(err, requesthandler.ErrInternal):
		return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
	case stderrors.Is(err, serviceerrors.ErrInternal):
		return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
	case stderrors.Is(err, serviceerrors.ErrIdentityVerificationRequired):
		return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "IDENTITY_VERIFICATION_REQUIRED",
			"Customer identity verification is required for this operation.").Wrap(err)
	case stderrors.Is(err, serviceerrors.ErrStateInvalid):
		return cerrors.FailedPrecondition(commonoutline.ServiceNameAPIManager, "STATE_INVALID",
			"The operation is invalid for the current resource state.").Wrap(err)
	case stderrors.Is(err, serviceerrors.ErrServiceUnavailable):
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "SERVICE_UNAVAILABLE",
			"An upstream service is temporarily unavailable.").Wrap(err)
	case stderrors.Is(err, serviceerrors.ErrInsufficientBalance):
		return cerrors.PaymentRequired(commonoutline.ServiceNameAPIManager, "INSUFFICIENT_BALANCE",
			"Customer balance is below the minimum required for this operation.").Wrap(err)
	}

	// 3. Transport failures.
	if stderrors.Is(err, context.Canceled) {
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "REQUEST_CANCELED", "The request was canceled.").Wrap(err)
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "REQUEST_TIMEOUT", "The request timed out.").Wrap(err)
	}

	// 4. Default.
	return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
}
