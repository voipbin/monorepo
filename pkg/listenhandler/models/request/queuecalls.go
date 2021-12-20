package request

// V1DataQueuecallsPost is
// v1 data type request struct for
// /v1/queuecalls POST
type V1DataQueuecallsPost struct {
	UserID uint64 `json:"user_id"`

	QueueID       string `json:"queue_id"`
	ReferenceType string `json:"reference_type"`
	ReferenceID   string `json:"reference_id"`

	WebhookURI    string `json:"webhook_uri"`
	WebhookMethod string `json:"webhook_method"`
}
