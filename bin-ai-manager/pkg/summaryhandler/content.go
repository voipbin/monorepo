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

type RequestContent struct {
	Prompt      string                    `json:"prompt,omitempty"`
	Transcripts []tmtranscript.Transcript `json:"transcripts,omitempty"`
	Variables   map[string]string         `json:"variables,omitempty"`
}

func (h *summaryHandler) ContentProcess(ctx context.Context, sm *summary.Summary) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ContentProcess",
		"summary": sm,
	})

	var err error
	switch sm.ReferenceType {
	case summary.ReferenceTypeCall:
		err = h.contentProcessReferenceTypeCall(ctx, sm.ReferenceID)

	case summary.ReferenceTypeConference:
		err = h.contentProcessReferenceTypeConference(ctx, sm.ReferenceID)

	default:
		err = errors.Errorf("unsupported reference type: %s", sm.ReferenceType)
	}

	if err != nil {
		log.Errorf("Could not process the content. err: %v", err)
		return
	}
}

func (h *summaryHandler) contentProcessReferenceTypeCall(ctx context.Context, callID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "contentProcessReferenceTypeCall",
		"call_id": callID,
	})

	sm, err := h.GetByReferenceID(ctx, callID)
	if err != nil {
		return errors.Wrapf(err, "could not get the summary")
	}

	// get transcripts
	filters := map[string]string{
		"deleted":      "false",
		"reference_id": sm.ReferenceID.String(),
	}
	transcripts, err := h.reqHandler.TranscribeV1TranscriptGets(ctx, "", 1000, filters)
	if err != nil {
		return errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.contentGet(ctx, sm.ActiveflowID, transcripts)
	if err != nil {
		return errors.Wrapf(err, "could not send the request")
	}
	log.WithField("content", content).Debugf("Parsed summary content.")

	tmp, err := h.UpdateStatusDone(ctx, sm.ID, content)
	if err != nil {
		return errors.Wrapf(err, "could not update the status")
	}
	log.WithField("summary", tmp).Debugf("Updated the summary status")

	if errFlow := h.startOnEndFlow(ctx, tmp); errFlow != nil {
		// we could not start the on end flow, but we can continue the process
		log.Errorf("Could not start the on end flow. err: %v", errFlow)
	}

	return nil
}

func (h *summaryHandler) contentProcessReferenceTypeConference(ctx context.Context, conferenceID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "contentProcessReferenceTypeConference",
		"conference_id": conferenceID,
	})

	sm, err := h.GetByReferenceID(ctx, conferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get the summary")
	}

	// get conference info
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, conferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get the conference data")
	}

	// get transcripts
	filters := map[string]string{
		"deleted":      "false",
		"reference_id": cf.ConfbridgeID.String(),
	}
	transcripts, err := h.reqHandler.TranscribeV1TranscriptGets(ctx, "", 1000, filters)
	if err != nil {
		return errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.contentGet(ctx, sm.ActiveflowID, transcripts)
	if err != nil {
		return errors.Wrapf(err, "could not send the request")
	}
	log.WithField("content", content).Debugf("Parsed summary content.")

	tmp, err := h.UpdateStatusDone(ctx, sm.ID, content)
	if err != nil {
		return errors.Wrapf(err, "could not update the status")
	}
	log.WithField("summary", tmp).Debugf("Updated the summary status")

	if errFlow := h.startOnEndFlow(ctx, tmp); errFlow != nil {
		// we could not start the on end flow, but we can continue the process
		log.Errorf("Could not start the on end flow. err: %v", errFlow)
	}

	return nil
}

func (h *summaryHandler) contentGet(ctx context.Context, activeflowID uuid.UUID, ts []tmtranscript.Transcript) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "contentGet",
		"activeflow_id": activeflowID,
	})

	// get variable
	variable, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return "", errors.Wrapf(err, "could not get the variable")
	}
	log.WithField("variable", variable).Debugf("Received variable")

	requestContent := RequestContent{
		Prompt:      defaultSummaryGeneratePrompt,
		Transcripts: ts,
		Variables:   variable.Variables,
	}

	tmpContent, err := json.Marshal(requestContent)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal the data")
	}
	log.WithField("request_content", requestContent).Debugf("Created request content.")

	req := &openai.ChatCompletionRequest{
		Model: defaultModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: string(tmpContent),
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
