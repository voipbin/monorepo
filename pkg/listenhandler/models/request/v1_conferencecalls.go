package request

// V1DataConferencecallsIDHealthCheckPost is
// v1 data type request struct for
// /v1/conferencecalls/<conferencecall-id>/health-check POST
type V1DataConferencecallsIDHealthCheckPost struct {
	RetryCount int `json:"retry_count"`
}
