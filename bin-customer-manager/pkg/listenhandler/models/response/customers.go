// Package response defines the wire response DTOs for the customer-manager
// RPC listener. Only wire-only shapes live here (root CLAUDE.md layering,
// style B): domain types are marshaled directly by the listener (style A).
package response

// V1ResponseCustomersCleanupUnverified is
// v1 response type struct for
// /v1/customers/cleanup_unverified POST
//
// Wraps the bare row counts (scalars with no domain type, style B). Expired is
// the number of initial customers moved to expired, Purged the number of
// expired customers anonymized after the retention period.
type V1ResponseCustomersCleanupUnverified struct {
	Expired int `json:"expired"`
	Purged  int `json:"purged"`
}

// V1ResponseCustomersCleanupFrozenExpired is
// v1 response type struct for
// /v1/customers/cleanup_frozen_expired POST
//
// Wraps the bare processed-row count — a scalar with no domain type (style B).
type V1ResponseCustomersCleanupFrozenExpired struct {
	Processed int `json:"processed"`
}
