// Package response defines the wire response DTOs for the scheduler-manager
// RPC listener. Only wire-only shapes live here (root CLAUDE.md layering,
// style B): domain types are marshaled directly by the listener (style A).
package response

// V1ResponseExecutionsPrune is
// v1 response type struct for
// /v1/executions/prune POST
//
// Wraps the bare removed-row count — a scalar with no domain type (style B).
type V1ResponseExecutionsPrune struct {
	Removed int64 `json:"removed"`
}
