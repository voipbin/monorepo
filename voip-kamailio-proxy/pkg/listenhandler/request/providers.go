package request

// V1DataProvidersHealthPost is the request body for POST /v1/providers/health.
type V1DataProvidersHealthPost struct {
	Hostname string `json:"hostname"`
}

// V1ResponseProvidersHealthPost is the response body for POST /v1/providers/health.
type V1ResponseProvidersHealthPost struct {
	Status     string `json:"status"`      // "healthy" | "unhealthy"
	ResultCode string `json:"result_code"` // SIP response code e.g. "200", "404", or "timeout"
}
