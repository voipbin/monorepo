// Package serviceerrors defines sentinel errors used by servicehandler
// methods in bin-api-manager. The translator in server/error_translate.go
// matches these sentinels (via errors.Is) and converts them into
// VoipbinError responses for the client.
//
// This layer is intentionally thin: servicehandler code can choose to
// construct a richer *cerrors.VoipbinError directly (preferred when the
// site has context like a resource ID), and fall back to a sentinel
// only when there is no specific context to add.
package serviceerrors

import stderrors "errors"

var (
	ErrPermissionDenied         = stderrors.New("permission denied")
	ErrNotFound                 = stderrors.New("not found")
	ErrAuthenticationRequired   = stderrors.New("authentication required")
	ErrDirectAccessNotSupported = stderrors.New("direct access not supported")
	ErrInvalidArgument          = stderrors.New("invalid argument")
	ErrInternal                 = stderrors.New("internal error")
)
