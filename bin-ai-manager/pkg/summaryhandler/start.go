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

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (h *summaryHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType summary.ReferenceType,
	referenceID uuid.UUID,
	language string,
) (*summary.Summary, error) {

	tmp, err := h.GetByCustomerIDAndReferenceIDAndLanguage(ctx, customerID, referenceID, language)
	if err == nil {
		// already exists
		return tmp, nil
	}

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
		cmcustomer.IDAIManager,
		activeflowID,
		uuid.Nil,
		tmtranscribe.ReferenceTypeCall,
		referenceID,
		language,
		tmtranscribe.DirectionBoth,
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
		cmcustomer.IDAIManager,
		activeflowID,
		uuid.Nil,
		tmtranscribe.ReferenceTypeConfbridge,
		cf.ConfbridgeID,
		language,
		tmtranscribe.DirectionIn,
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
	filters := map[string]string{
		"deleted":       "false",
		"transcribe_id": referenceID.String(),
	}
	ts, err := h.reqHandler.TranscribeV1TranscriptGets(ctx, "", 1000, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.contentGet(ctx, activeflowID, ts)
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
		"activeflow_id":  "activeflowID",
		"on_end_flow_id": onEndFlowID,
		"reference_id":   referenceID,
		"language":       language,
	})

	log.Debugf("Start the transcribe.")

	// note: here, we set the customer id as the ai manager id
	// thie is required becasue if we use the customer id, the created transcribe will be shown to the
	// customer's transcribe list.
	tr, err := h.reqHandler.TranscribeV1TranscribeStart(
		ctx,
		cmcustomer.IDAIManager,
		activeflowID,
		uuid.Nil,
		tmtranscribe.ReferenceTypeRecording,
		referenceID,
		language,
		tmtranscribe.DirectionBoth,
		300000,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not start the transcribe")
	}
	log.WithField("transcribe", tr).Debugf("Finished transcribe. transcribe_id: %s", tr.ID)

	// get transcripts
	filters := map[string]string{
		"deleted":       "false",
		"transcribe_id": tr.ID.String(),
	}
	transcripts, err := h.reqHandler.TranscribeV1TranscriptGets(ctx, "", 1000, filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe data")
	}

	content, err := h.contentGet(ctx, activeflowID, transcripts)
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
