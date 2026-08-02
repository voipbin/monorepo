package response

// V1ResponseFailedEventsRetry is
// v1 response type for
// /v1/failed_events/retry POST
type V1ResponseFailedEventsRetry struct {
	Retried   int `json:"retried"`
	Succeeded int `json:"succeeded"`
	Exhausted int `json:"exhausted"`
}
