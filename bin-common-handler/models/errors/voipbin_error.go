package errors

// VoipbinError is the canonical error shape returned from the external
// VoIPbin API and (eventually) over RPC between internal managers.
// The Cause field is for server-side logging only and is never
// serialized to clients.
type VoipbinError struct {
	Status  Status `json:"status"`
	Reason  string `json:"reason"`
	Domain  string `json:"domain"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}
