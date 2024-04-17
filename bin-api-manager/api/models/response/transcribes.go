package response

import (
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
)

// BodyTranscribesGET is response body define for
// GET /v1.0/transcripts
type BodyTranscribesGET struct {
	Result []*tmtranscribe.WebhookMessage `json:"result"`
	Pagination
}
