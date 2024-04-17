package request

// ParamTranscriptsGET is rquest param define for
// GET /v1.0/transcripts
type ParamTranscriptsGET struct {
	Pagination
	TranscribeID string `form:"transcribe_id"`
}
