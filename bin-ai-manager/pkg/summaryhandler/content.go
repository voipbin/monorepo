package summaryhandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"monorepo/bin-ai-manager/models/summary"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type RequestContent struct {
	Prompt         string                    `json:"prompt,omitempty"`
	ReferenceType  string                    `json:"reference_type,omitempty"`
	OutputLanguage string                    `json:"output_language,omitempty"`
	Transcripts    []tmtranscript.Transcript `json:"transcripts,omitempty"`
	Variables      map[string]string         `json:"variables,omitempty"`
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

	transcripts, err := h.contentGetTranscripts(ctx, sm.ReferenceID)
	if err != nil {
		return errors.Wrapf(err, "could not get the transcripts")
	}

	content, err := h.contentGet(ctx, sm.ActiveflowID, sm.ReferenceType, transcripts, sm.Language)
	if err != nil {
		return errors.Wrapf(err, "could not send the request")
	}
	log.WithField("content", content).Debugf("Parsed summary content.")

	tmp, err := h.UpdateStatusDone(ctx, sm.ID, content)
	if stderrors.Is(err, ErrSummaryAlreadyDone) {
		// already finalized by a previous delivery -- clean no-op, not a failure.
		log.Debugf("Summary already done, skipping reprocessing. summary_id: %s", sm.ID)
		return nil
	}
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

	transcripts, err := h.contentGetTranscripts(ctx, cf.ConfbridgeID)
	if err != nil {
		return errors.Wrapf(err, "could not get the transcripts")
	}

	content, err := h.contentGet(ctx, sm.ActiveflowID, sm.ReferenceType, transcripts, sm.Language)
	if err != nil {
		return errors.Wrapf(err, "could not send the request")
	}
	log.WithField("content", content).Debugf("Parsed summary content.")

	tmp, err := h.UpdateStatusDone(ctx, sm.ID, content)
	if stderrors.Is(err, ErrSummaryAlreadyDone) {
		// bin-conference-manager can publish conference_deleted twice for the same
		// conference (Delete()'s own publish, then Destroy()'s once the
		// asynchronously-kicked participants actually leave) -- see
		// ErrSummaryAlreadyDone's doc comment and EventCMConferenceUpdated's. This is
		// the second delivery finding the first already finalized it: clean no-op,
		// not a failure.
		log.Debugf("Summary already done, skipping reprocessing. summary_id: %s", sm.ID)
		return nil
	}
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

func (h *summaryHandler) contentGetTranscripts(ctx context.Context, referenceID uuid.UUID) ([]tmtranscript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "contentGetTranscripts",
		"reference_id": referenceID,
	})

	// get transcribe
	transcribeFilters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldDeleted:     false,
		tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManager.String(),
		tmtranscribe.FieldReferenceID: referenceID.String(),
	}

	tr, err := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 1, transcribeFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	} else if len(tr) == 0 {
		return nil, errors.Errorf("could not find the transcribe data")
	}
	log.WithField("transcribe", tr).Debugf("Found transcribe. transcribe_id: %s", tr[0].ID)

	transcriptFilters := map[tmtranscript.Field]any{
		tmtranscript.FieldDeleted:      false,
		tmtranscript.FieldTranscribeID: tr[0].ID.String(),
	}
	res, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", 1000, transcriptFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	return res, nil
}

func (h *summaryHandler) contentGet(ctx context.Context, activeflowID uuid.UUID, referenceType summary.ReferenceType, ts []tmtranscript.Transcript, outputLanguage string) (string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "contentGet",
		"activeflow_id": activeflowID,
	})

	// contentGet may be invoked without the upstream Start normalization (e.g.
	// direct callers/tests). Confirm the effective output language locally so the
	// system prompt is never malformed ("write ... in  (BCP47)") and so
	// shouldVerify / RequestContent.OutputLanguage all use a consistent value.
	effectiveLang := outputLanguage
	if effectiveLang == "" {
		effectiveLang = defaultOutputLanguage
	}

	// get variables
	var variables map[string]string
	if activeflowID != uuid.Nil {
		tmp, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
		if err != nil {
			return "", errors.Wrapf(err, "could not get the variable")
		}
		log.WithField("variable", tmp).Debugf("Received variable")

		variables = tmp.Variables
	}

	requestContent := RequestContent{
		Prompt:         defaultSummaryGeneratePrompt,
		ReferenceType:  string(referenceType),
		OutputLanguage: effectiveLang,
		Transcripts:    ts,
		Variables:      variables,
	}

	tmpContent, err := json.Marshal(requestContent)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshal the data")
	}
	log.WithField("request_content", requestContent).Debugf("Created request content.")

	// First-pass enforcement: pin the output language by value in a system
	// message ([system, user]), removing the previous self-reference indirection.
	req := &openai.ChatCompletionRequest{
		Model:           h.model,
		Temperature:     summaryTemperature,
		ReasoningEffort: h.reasoningEffort,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: fmt.Sprintf(languageSystemPromptFmt, effectiveLang),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: string(tmpContent),
			},
		},
	}

	generateOnce := func() (string, error) {
		tmpRes, errSend := h.engineOpenaiHandler.Send(ctx, req)
		if errSend != nil {
			return "", errors.Wrapf(errSend, "could not send the request")
		}
		log.WithField("response", tmpRes).Debugf("Received response")

		if tmpRes == nil || len(tmpRes.Choices) == 0 {
			log.Debugf("Received response with empty choices")
			return "", nil
		}
		return tmpRes.Choices[0].Message.Content, nil
	}

	// Second-pass safety net is only meaningful for non-English targets
	// (see design §5.1.1). For skipped targets keep the original single-Send path.
	if !h.shouldVerify(effectiveLang) {
		return generateOnce()
	}

	maxAttempts := 1 + maxSummaryRegenerations
	var lastNonEmpty string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		content, errGen := generateOnce()
		if errGen != nil {
			if attempt == 1 {
				// First attempt failure means there is no summary at all; propagate.
				return "", errGen
			}
			// A regeneration (attempt>=2) hard-failure must not discard an already
			// secured result: the harness's extra call's transient failure should
			// not turn a good summary into a hard failure. Fall back best-effort.
			log.Warnf("regeneration send failed at attempt %d/%d, falling back to last good summary. err=%v", attempt, maxAttempts, errGen)
			break
		}
		if content == "" {
			// Empty result carries no language to verify and must not overwrite an
			// earlier non-empty result.
			break
		}
		lastNonEmpty = content
		if h.verifyOutputLanguage(ctx, content, effectiveLang) {
			return content, nil
		}
		log.Warnf("summary output language mismatch, attempt %d/%d, want %s", attempt, maxAttempts, effectiveLang)
	}

	// Cap reached / empty result / regeneration send failure: return the last
	// non-empty result secured (empty string if none).
	return lastNonEmpty, nil
}

// shouldVerify reports whether the second-pass output-language verification
// should run for the given effective language. Only English (en) targets are
// skipped: the dominant default path is en-US and the original symptom
// (non-English target rendered in English) cannot occur for an English target,
// so verifying English would double LLM calls on dominant traffic for almost no
// benefit. Every non-English target (Latin or non-Latin) is verified.
func (h *summaryHandler) shouldVerify(outputLanguage string) bool {
	sub := canonPrimarySubtag(outputLanguage)
	return sub != "" && sub != englishPrimarySubtag
}

// canonPrimarySubtag extracts the BCP47 primary subtag and lowercases it
// (BCP47 is case-insensitive), e.g. "ko-KR"->"ko", "KO-KR"->"ko", "EN"->"en".
func canonPrimarySubtag(lang string) string {
	if lang == "" {
		return ""
	}
	primary := lang
	if idx := strings.IndexByte(lang, '-'); idx >= 0 {
		primary = lang[:idx]
	}
	return strings.ToLower(primary)
}

// verifyOutputLanguage asks a cheap detector LLM whether content is written in
// the requested language. It is fail-open: any uncertainty (empty/unexpected
// output, SendOnce error/timeout) returns true so a normal summary is never
// discarded. Verification is skipped (true) for skip targets and for summaries
// with too little prose to judge.
func (h *summaryHandler) verifyOutputLanguage(ctx context.Context, content string, outputLanguage string) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":            "verifyOutputLanguage",
		"output_language": outputLanguage,
	})

	if !h.shouldVerify(outputLanguage) {
		return true
	}

	// Minimum-prose guard: if the summary is essentially just headers and
	// "- None" items (prose below languageVerifyMinProse runes), there is no
	// natural-language prose to judge; treat as a pass to avoid wasting retries.
	if proseLen(content) < languageVerifyMinProse {
		log.Debugf("Not enough prose to verify output language, skipping.")
		return true
	}

	// Rune-safe truncation of the sample (never byte-slice multibyte text).
	sample := content
	if r := []rune(sample); len(r) > languageVerifySampleLen {
		sample = string(r[:languageVerifySampleLen])
	}

	verifyCtx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()

	req := &openai.ChatCompletionRequest{
		Model:           h.model,
		Temperature:     summaryTemperature,
		ReasoningEffort: h.reasoningEffort,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(languageVerifyPrompt, outputLanguage, sample),
			},
		},
	}

	tmpRes, err := h.engineOpenaiHandler.SendOnce(verifyCtx, req)
	if err != nil {
		log.Debugf("Could not verify output language, treating as pass. err: %v", err)
		return true
	}
	if tmpRes == nil || len(tmpRes.Choices) == 0 {
		log.Debugf("Empty verification response, treating as pass.")
		return true
	}

	answer := strings.ToLower(strings.TrimSpace(tmpRes.Choices[0].Message.Content))
	switch {
	case strings.HasPrefix(answer, "yes"):
		return true
	case strings.HasPrefix(answer, "no"):
		return false
	default:
		log.Debugf("Unexpected verification answer %q, treating as pass.", answer)
		return true
	}
}

// proseLen returns the number of runes of natural-language prose in content,
// excluding section-header lines (a label ending in ':') and "- None" items.
//
// Tradeoff: only the literal English "None" item is excluded. When a non-English
// summary renders an empty section with a translated placeholder (e.g. Korean
// "없음"), that placeholder IS counted as prose here, so a summary that is really
// all-empty may clear languageVerifyMinProse and get sent to the verifier. This
// is harmless by design: verification only runs for the target (non-English)
// language, the placeholder is already in that target language, so the verifier
// answers "yes" (a pass). The cost is at most one extra verify call, never a
// false rejection. Accepted per design §7 rather than maintaining a
// per-language "None" translation table.
func proseLen(content string) int {
	total := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Section header line, e.g. "Call Type:".
		if strings.HasSuffix(trimmed, ":") {
			continue
		}
		// Empty / "None" list items.
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		if item == "" || strings.EqualFold(item, "None") {
			continue
		}
		total += len([]rune(item))
	}
	return total
}
