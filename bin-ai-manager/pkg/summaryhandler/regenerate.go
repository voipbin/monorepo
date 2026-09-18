package summaryhandler

import (
	"context"

	"monorepo/bin-ai-manager/models/summary"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Regenerate re-generates the content of an existing summary in place (VOIP-1535).
//
// The caller (api-manager servicehandler) targets a specific summary by id, so this
// handler does NOT re-derive the replacement target (no GetByReferenceID newest-guess,
// no orphan ambiguity) and does NOT touch the create-path dedup. Ownership is verified
// exclusively at the servicehandler layer (fetch-then-permission, AISummaryDelete
// pattern); ai-manager receives no customerID and trusts the api-manager auth gate.
//
// Scope is recording-only: call/conference fill content asynchronously (not inside a
// synchronous contentGet call), so the regenerate-then-update model does not hold for
// them; transcribe uses a different content path and is out of scope for this change.
//
// The new content is generated FIRST (VOIP-1532 output-language enforcement runs via
// contentGet's OutputLanguage), and the record is only overwritten on success, so a
// failed regenerate leaves the existing content untouched (no data-loss window).
func (h *summaryHandler) Regenerate(ctx context.Context, summaryID uuid.UUID, language string) (*summary.Summary, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Regenerate",
		"summary_id": summaryID,
		"language":   language,
	})

	// load the target (id-targeting: no re-query/guess).
	existing, err := h.Get(ctx, summaryID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the summary")
	}

	// scope: recording only. reject any other reference type explicitly.
	if existing.ReferenceType != summary.ReferenceTypeRecording {
		return nil, errors.Errorf("regenerate is supported only for recording reference type. reference_type: %s", existing.ReferenceType)
	}

	// confirm the output language: empty keeps the existing summary's language
	// (no re-selection), otherwise use the requested language (language swap).
	lang := language
	if lang == "" {
		lang = existing.Language
	}

	// regenerate the content FIRST (VOIP-1532 language enforcement + transcript reuse).
	// activeflowID is uuid.Nil: a manual regenerate has no flow-variable context, and
	// contentGet skips the variable lookup when activeflowID is Nil.
	transcripts, err := h.getRecordingTranscripts(ctx, uuid.Nil, existing.ReferenceID)
	if err != nil {
		// existing record is left untouched.
		return nil, errors.Wrapf(err, "could not get the transcripts")
	}

	newContent, err := h.contentGet(ctx, uuid.Nil, summary.ReferenceTypeRecording, transcripts, lang)
	if err != nil {
		// existing record is left untouched.
		return nil, errors.Wrapf(err, "could not generate the summary content")
	}

	// contentGet can return ("", nil) without a hard error (empty LLM choices, or a
	// non-English target whose last non-empty result was empty). Unlike the Start
	// create-path (which has no prior content to lose), Regenerate would overwrite an
	// existing good summary with an empty string, which is a real data-loss regression.
	// Preserve the existing record and surface an error instead (no data-loss window).
	if newContent == "" {
		return nil, errors.Errorf("regenerated summary content is empty; keeping the existing summary")
	}
	log.WithField("content", newContent).Debugf("Regenerated the summary content.")

	// only now overwrite the same record (content + language together, so a language
	// swap stays on the same id).
	res, err := h.UpdateContentLanguage(ctx, existing.ID, newContent, lang)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the summary")
	}

	return res, nil
}
