package streaminghandler

import (
	"github.com/aws/aws-sdk-go-v2/service/transcribestreaming"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// AWSRunForTest runs the AWS speech process of a handler that uses the given client. It lets the
// external test package drive awsRun without exposing it in the product code.
func AWSRunForTest(
	client *transcribestreaming.Client,
	notifyHandler notifyhandler.NotifyHandler,
	transcriptHandler transcripthandler.TranscriptHandler,
	st *streaming.Streaming,
) error {
	h := &streamingHandler{
		awsClient:         client,
		notifyHandler:     notifyHandler,
		transcriptHandler: transcriptHandler,
	}

	return h.awsRun(st)
}
