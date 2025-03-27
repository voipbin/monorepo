package summaryhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-ai-manager/models/summary"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

func (h *summaryHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType summary.ReferenceType,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {

	switch referenceType {
	case summary.ReferenceTypeTranscribe:
		return h.startReferenceTypeTranscribe(ctx, customerID, activeflowID, referenceID, language)

	default:
		return nil, errors.Errorf("unsupported reference type: %s", referenceType)
	}
}

type RequestContent struct {
	Prompt      string `json:"prompt,omitempty"`
	Transcripts []tmtranscript.Transcript
	Variables   map[string]string
}

func (h *summaryHandler) startReferenceTypeTranscribe(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	// get transcripts
	filters := map[string]string{
		"reference_id": referenceID.String(),
	}
	ts, err := h.reqestHandler.TranscribeV1TranscriptGets(ctx, "", 1000, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.getContent(ctx, activeflowID, ts)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}

	res, err := h.Create(ctx, customerID, summary.ReferenceTypeTranscribe, referenceID, language, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the summary")
	}

	return res, nil
}

func (h *summaryHandler) startReferenceTypeRecording(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	// get transcripts
	filters := map[string]string{
		"reference_id": referenceID.String(),
	}
	ts, err := h.reqestHandler.TranscribeV1TranscriptGets(ctx, "", 1000, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.getContent(ctx, activeflowID, ts)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}

	res, err := h.Create(ctx, customerID, summary.ReferenceTypeTranscribe, referenceID, language, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the summary")
	}

	return res, nil
}

func (h *summaryHandler) getContent(ctx context.Context, activeflowID uuid.UUID, ts []tmtranscript.Transcript) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "getContent",
		"activeflowID": activeflowID,
	})

	// get variable
	variable, err := h.reqestHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return "", errors.Wrapf(err, "could not get the variable")
	}
	log.WithField("variable", variable).Debugf("Received variable")

	// create message
	requestcontent := &RequestContent{
		Prompt:      "Generate a concise yet informative call summary based on the provided transcription, recording link, conference details and other variables. Focus on key points, action items, and important decisions made during the call.",
		Transcripts: ts,
		Variables:   variable.Variables,
	}
	content, err := json.Marshal(requestcontent)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal the data")
	}
	log.WithField("request_content", requestcontent).Debugf("Created request content.")

	req := &openai.ChatCompletionRequest{
		Model: defaultModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: string(content),
			},
		},
	}
	tmpRes, err := h.engineOpenaiHandler.Send(ctx, req)
	if err != nil {
		return "", errors.Wrapf(err, "could not send the request")
	}
	log.WithField("response", tmpRes).Debugf("Received response")

	if tmpRes == nil || len(tmpRes.Choices) == 0 {
		log.Debugf("Received response with empty choices")
		return "", nil
	}

	res := tmpRes.Choices[0].Message.Content
	return res, nil
}
