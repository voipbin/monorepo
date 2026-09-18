package summaryhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/summary"
	cmcall "monorepo/bin-call-manager/models/call"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// defaultOutputLanguage is the fallback summary output language when the
	// caller does not specify one.
	defaultOutputLanguage = "en-US"

	// defaultSTTFallbackLanguage is the STT language used only when no existing
	// transcribe can be reused for a recording. It is intentionally independent
	// of the output language (reusing the output language for STT would regress
	// empty-transcript behaviour). See §4.2 of the design doc.
	defaultSTTFallbackLanguage = "en-US"
)

// normalizeOutputLanguage returns the confirmed summary output language,
// defaulting to en-US when the caller passes an empty value.
func normalizeOutputLanguage(language string) string {
	if language == "" {
		return defaultOutputLanguage
	}
	return language
}

func (h *summaryHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType summary.ReferenceType,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {

	// normalize the output language before the dedup lookup so that an empty
	// language and its normalized default (en-US) do not diverge into two
	// separate summaries. All downstream points (dedup, startReferenceType*,
	// Create) use this confirmed value.
	language = normalizeOutputLanguage(language)

	tmp, err := h.GetByCustomerIDAndReferenceIDAndLanguage(ctx, customerID, referenceID, language)
	if err == nil {
		// already exists
		return tmp, nil
	}

	promSummaryStartTotal.WithLabelValues(string(referenceType)).Inc()

	switch referenceType {
	case summary.ReferenceTypeTranscribe:
		return h.startReferenceTypeTranscribe(ctx, customerID, activeflowID, onEndFlowID, referenceID, language)

	case summary.ReferenceTypeRecording:
		return h.startReferenceTypeRecording(ctx, customerID, activeflowID, onEndFlowID, referenceID, language)

	case summary.ReferenceTypeCall:
		return h.startReferenceTypeCall(ctx, customerID, activeflowID, onEndFlowID, referenceID, language)

	case summary.ReferenceTypeConference:
		return h.startReferenceTypeConference(ctx, customerID, activeflowID, onEndFlowID, referenceID, language)

	default:
		return nil, errors.Errorf("unsupported reference type: %s", referenceType)
	}
}

func (h *summaryHandler) startReferenceTypeCall(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeCall",
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	// get call info
	c, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the call data")
	}

	if c.Status == cmcall.StatusHangup {
		return nil, fmt.Errorf("the call has already been hung up")
	}

	if activeflowID == uuid.Nil {
		log.Debugf("ActiveflowID is nil. Set the activeflowID as the call's activeflowID.")
		activeflowID = c.ActiveflowID
	}

	// transcribe start
	// note: here, we set the customer id as the ai manager id
	// thie is required becasue if we use the customer id, the created transcribe will be shown to the
	// customer's transcribe list.
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		uuid.Nil,
		cmcustomer.IDAIManager,
		activeflowID,
		uuid.Nil,
		tmtranscribe.ReferenceTypeCall,
		referenceID,
		language,
		tmtranscribe.DirectionBoth,
		tmtranscribe.ProviderEmpty,
		5000,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the transcribe.")
	}
	log.WithField("transcribe", tr).Debugf("Started transcribe. transcribe_id: %s", tr.ID)

	res, err := h.Create(
		ctx,
		customerID,
		activeflowID,
		onEndFlowID,
		summary.ReferenceTypeCall,
		referenceID,
		summary.StatusProgressing,
		language,
		"",
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the summary")
	}

	return res, nil
}

func (h *summaryHandler) startReferenceTypeConference(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "startReferenceTypeConference",
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	// get conference info
	cf, err := h.reqHandler.ConferenceV1ConferenceGet(ctx, referenceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the conference data")
	}

	if cf.Status != cfconference.StatusProgressing {
		return nil, fmt.Errorf("the conference is not progressing")
	}

	// transcribe start
	// note: here, we set the customer id as the ai manager id
	// thie is required becasue if we use the customer id, the created transcribe will be shown to the
	// customer's transcribe list.
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		uuid.Nil,
		cmcustomer.IDAIManager,
		activeflowID,
		uuid.Nil,
		tmtranscribe.ReferenceTypeConfbridge,
		cf.ConfbridgeID,
		language,
		tmtranscribe.DirectionIn,
		tmtranscribe.ProviderEmpty,
		5000,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the transcribe.")
	}
	log.WithField("transcribe", tr).Debugf("Started transcribe. transcribe_id: %s", tr.ID)

	res, err := h.Create(
		ctx,
		customerID,
		activeflowID,
		onEndFlowID,
		summary.ReferenceTypeConference,
		referenceID,
		summary.StatusProgressing,
		language,
		"",
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the summary")
	}

	return res, nil
}

func (h *summaryHandler) startReferenceTypeTranscribe(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startReferenceTypeTranscribe",
		"activeflow_id":  activeflowID,
		"on_end_flow_id": onEndFlowID,
		"reference_id":   referenceID,
		"language":       language,
	})

	// get transcripts
	filters := map[tmtranscript.Field]any{
		tmtranscript.FieldDeleted:      false,
		tmtranscript.FieldTranscribeID: referenceID.String(),
	}
	ts, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", 1000, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeTranscribe, ts, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}
	log.WithField("content", content).Debugf("Parsed summary content.")

	res, err := h.Create(ctx, customerID, activeflowID, onEndFlowID, summary.ReferenceTypeTranscribe, referenceID, summary.StatusDone, language, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the summary")
	}

	if errFlow := h.startOnEndFlow(ctx, res); errFlow != nil {
		// we could not start the on end flow, but we can continue the process
		log.Errorf("Could not start the on end flow. err: %v", errFlow)
	}

	return res, nil
}

func (h *summaryHandler) startReferenceTypeRecording(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "startReferenceTypeRecording",
		"activeflow_id":  activeflowID,
		"on_end_flow_id": onEndFlowID,
		"reference_id":   referenceID,
		"language":       language,
	})

	log.Debugf("Getting the transcripts for the recording summary.")

	transcripts, err := h.getRecordingTranscripts(ctx, activeflowID, referenceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcripts")
	}

	content, err := h.contentGet(ctx, activeflowID, summary.ReferenceTypeRecording, transcripts, language)
	if err != nil {
		return nil, errors.Wrapf(err, "could not send the request")
	}
	log.WithField("content", content).Debugf("Parsed summary content.")

	res, err := h.Create(ctx, customerID, activeflowID, onEndFlowID, summary.ReferenceTypeRecording, referenceID, summary.StatusDone, language, content)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the summary")
	}

	if errFlow := h.startOnEndFlow(ctx, res); errFlow != nil {
		// we could not start the on end flow, but we can continue the process
		log.Errorf("Could not start the on end flow. err: %v", errFlow)
	}

	return res, nil
}

// getRecordingTranscripts returns the transcripts to summarize for a recording.
//
// It first tries to reuse an existing (original/admin) transcribe for the
// recording rather than always creating a new one. The reuse candidate must be:
//  1. a transcribe whose customer_id is NOT cmcustomer.IDAIManager (i.e. an
//     original transcribe created by the user/admin, not one that a previous
//     summary created). This exclusion is done in code, not via a DB filter:
//     the List NotEq filter is string-kind only and silently misbehaves on the
//     uuid.UUID customer_id (see bin-common-handler/databasehandler/main.go
//     SCOPE WARNING), and the reuse query intentionally does NOT filter by
//     customer_id (contentGetTranscripts filters by IDAIManager, which would
//     hide the admin transcribe).
//  2. a transcribe that actually has at least one transcript.
//
// TranscribeV1TranscribeList returns tm_create DESC, so the candidates are
// walked newest-first and the first non-IDAIManager candidate with transcripts
// is adopted immediately (short-circuit, N+1 mitigation). If no candidate
// qualifies, a new transcribe is created with the STT fallback language
// (en-US, independent of the output language) and its transcripts are used.
func (h *summaryHandler) getRecordingTranscripts(ctx context.Context, activeflowID uuid.UUID, referenceID uuid.UUID) ([]tmtranscript.Transcript, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "getRecordingTranscripts",
		"reference_id": referenceID,
	})

	// look for an existing (original) transcribe to reuse.
	// note: no customer_id filter here -- original transcribes carry the real
	// customer_id, and only summary-created ones are IDAIManager. The IDAIManager
	// exclusion is done in code below.
	reuseFilters := map[tmtranscribe.Field]any{
		tmtranscribe.FieldReferenceID:   referenceID.String(),
		tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeRecording,
		tmtranscribe.FieldStatus:        tmtranscribe.StatusDone,
		tmtranscribe.FieldDeleted:       false,
	}
	candidates, err := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 100, reuseFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not list the existing transcribes")
	}

	for _, cand := range candidates {
		if cand.CustomerID == cmcustomer.IDAIManager {
			// skip summary-created transcribes (potential empty/polluted source).
			continue
		}

		transcriptFilters := map[tmtranscript.Field]any{
			tmtranscript.FieldDeleted:      false,
			tmtranscript.FieldTranscribeID: cand.ID.String(),
		}
		transcripts, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", 1000, transcriptFilters)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get the transcripts for the candidate transcribe")
		}
		if len(transcripts) > 0 {
			// short-circuit: first original candidate with transcripts wins.
			log.Debugf("Reusing existing transcribe. transcribe_id: %s", cand.ID)
			return transcripts, nil
		}
	}

	// no reusable transcribe -- create a new one with the STT fallback language.
	// note: here, we set the customer id as the ai manager id
	// thie is required becasue if we use the customer id, the created transcribe will be shown to the
	// customer's transcribe list.
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		uuid.Nil,
		cmcustomer.IDAIManager,
		activeflowID,
		uuid.Nil,
		tmtranscribe.ReferenceTypeRecording,
		referenceID,
		defaultSTTFallbackLanguage,
		tmtranscribe.DirectionBoth,
		tmtranscribe.ProviderEmpty,
		300000,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the transcribe")
	}
	log.WithField("transcribe", tr).Debugf("Finished transcribe. transcribe_id: %s", tr.ID)

	transcriptFilters := map[tmtranscript.Field]any{
		tmtranscript.FieldDeleted:      false,
		tmtranscript.FieldTranscribeID: tr.ID.String(),
	}
	transcripts, err := h.reqHandler.TranscribeV1TranscriptList(ctx, "", 1000, transcriptFilters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	return transcripts, nil
}

func (h *summaryHandler) startOnEndFlow(ctx context.Context, sm *summary.Summary) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "startOnEndFlow",
		"summary": sm,
	})

	if sm.OnEndFlowID == uuid.Nil {
		// has no on end flow. nothing to do
		return nil
	}

	af, err := h.reqHandler.FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		sm.CustomerID,
		sm.OnEndFlowID,
		fmactiveflow.ReferenceTypeAI,
		sm.ID,
		sm.ActiveflowID,
		nil,
		"",
		fmactiveflow.WebhookMethodNone,
	)
	if err != nil {
		return errors.Wrapf(err, "could not create the activeflow")
	}
	log.WithField("activeflow", af).Debugf("Created the activeflow")

	if errSet := h.variableSet(ctx, af.ID, sm); errSet != nil {
		// we could not set the variable, but we can continue the process
		log.Errorf("could not set the variable. activeflow_id: %s", af.ID)
	}

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		return errors.Wrapf(errExecute, "could not execute the activeflow")
	}
	log.Debugf("Executed the activeflow")

	return nil
}
