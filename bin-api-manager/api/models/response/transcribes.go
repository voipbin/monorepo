package response

import (
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// BodyTranscribesGET is response body define for
// GET /v1.0/transcripts
type BodyTranscribesGET struct {
	Result []*tmtranscribe.WebhookMessage `json:"result"`
	Pagination
}
