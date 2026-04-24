// Package errors defines the shared VoipbinError type used across the
// VoIPbin monorepo for external API error responses and for typed
// errors on the RabbitMQ RPC channel.
//
// Import alias: this package shadows stdlib "errors". Most callers
// import it as "cerrors" alongside the standard library:
//
//	import (
//	    stderrors "errors"
//	    cerrors "monorepo/bin-common-handler/models/errors"
//	)
//
// Admission-rule note: this package currently has one day-1 consumer
// (bin-api-manager). The monorepo admission rule normally requires
// three consumers for bin-common-handler residency. Exception
// justification: VoipbinError is the shared RPC contract that every
// internal manager can emit via sock.Response.Data, so its cross-
// service consumer count will grow as managers migrate.
package errors

// Status is the canonical error status. It maps 1:1 to an HTTP status
// code via HTTPStatusFor. The set is intentionally closed — extending
// it requires a coordinated schema update across every consumer.
type Status string

const (
	StatusInvalidArgument    Status = "INVALID_ARGUMENT"
	StatusUnauthenticated    Status = "UNAUTHENTICATED"
	StatusPaymentRequired    Status = "PAYMENT_REQUIRED"
	StatusPermissionDenied   Status = "PERMISSION_DENIED"
	StatusNotFound           Status = "NOT_FOUND"
	StatusAlreadyExists      Status = "ALREADY_EXISTS"
	StatusFailedPrecondition Status = "FAILED_PRECONDITION"
	StatusResourceExhausted  Status = "RESOURCE_EXHAUSTED"
	StatusUnavailable        Status = "UNAVAILABLE"
	StatusInternal           Status = "INTERNAL"
)
