package response

import (
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// BodyTranscriptsGET is response body define for
// GET /v1.0/transcripts
type BodyTranscriptsGET struct {
	Result []*tmtranscript.WebhookMessage `json:"result"`
	Pagination
}
