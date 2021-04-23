package response

import "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

// BodyTranscribesPOST is response body define for POST /transcribes
type BodyTranscribesPOST struct {
	transcribe.Transcribe
}
