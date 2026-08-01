// Package request defines the wire request DTOs for the schedule-manager
// RPC listener. The shapes are pinned byte-for-byte by the client methods in
// bin-common-handler/pkg/requesthandler/schedule_schedule.go — keep them in
// sync (rpc.md §9.5: flat structs, no wrapper).
package request

// Pagination is pagination structure for request
type Pagination struct {
	PageSize  uint64 `form:"page_size" json:"page_size"`
	PageToken string `form:"page_token" json:"page_token"`
}
