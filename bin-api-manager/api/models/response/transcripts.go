package response

import (
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
)

// BodyTranscriptsGET is response body define for
// GET /v1.0/transcripts
type BodyTranscriptsGET struct {
	Result []*tmtranscript.WebhookMessage `json:"result"`
	Pagination
}
