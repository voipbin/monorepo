package server

import (
	"context"
	stderrors "errors"
	"strings"

	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// translateToVoipbinError maps any error returned from a servicehandler
// into a *VoipbinError. Priority order:
//  1. Typed passthrough (errors.As).
//  2. Sentinel match (errors.Is against serviceerrors.Err*).
//  3. Transport-failure detection (context.Canceled / DeadlineExceeded).
//  4. Substring fallback for legacy fmt.Errorf messages (shrinks over time).
//  5. Default: Internal with the original error wrapped as Cause.
//
// The whole function is wrapped in defer recover() so a panic inside
// any branch degrades to INTERNAL rather than dropping the response.
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
	case stderrors.Is(err, serviceerrors.ErrInternal):
		return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
	}

	// 3. Transport failures.
	if stderrors.Is(err, context.Canceled) {
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "REQUEST_CANCELED", "The request was canceled.").Wrap(err)
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "REQUEST_TIMEOUT", "The request timed out.").Wrap(err)
	}

	// 4. Substring fallback for legacy fmt.Errorf errors. This set is
	// intentionally small — each pattern here is a migration target
	// for a sentinel in subsequent PRs.
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no permission"):
		return cerrors.PermissionDenied(commonoutline.ServiceNameAPIManager, "PERMISSION_DENIED", "You do not have permission to access this resource.").Wrap(err)
	case strings.Contains(msg, "authentication required"):
		return cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required.").Wrap(err)
	case strings.Contains(msg, "not found"):
		return cerrors.NotFound(commonoutline.ServiceNameAPIManager, "RESOURCE_NOT_FOUND", "The requested resource was not found.").Wrap(err)
	case strings.Contains(msg, "unavailable"):
		return cerrors.Unavailable(commonoutline.ServiceNameAPIManager, "SERVICE_UNAVAILABLE", "An upstream service is temporarily unavailable.").Wrap(err)
	}

	// 5. Default.
	return cerrors.Internal(commonoutline.ServiceNameAPIManager, "INTERNAL", "An internal error occurred.").Wrap(err)
}
