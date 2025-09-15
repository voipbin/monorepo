package transcribehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-transcribe-manager/models/transcribe"
)

// Stop stops the progressing transcribe process.
func (h *transcribeHandler) Stop(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Stop",
		"transcribe_id": id,
	})

	// get transcribe and evaluate
	tr, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get the transcribe. transcribe_id: %s", id)
	}
	log.WithField("transcribe", tr).Debugf("Found the transcribe. transcribe_id: %s", tr.ID)

	if tr.Status == transcribe.StatusDone {
		// already stopped
		log.WithField("transcribe", tr).Debugf("Already stopped. transcribe_id: %s", tr.ID)
		return tr, nil
	}

	var res *transcribe.Transcribe
	switch tr.ReferenceType {
	case transcribe.ReferenceTypeCall, transcribe.ReferenceTypeConfbridge:
		res, err = h.stopLive(ctx, tr)

	default:
		log.Errorf("Invalid reference type. reference_type: %s", tr.ReferenceType)
		return nil, fmt.Errorf("invalid reference type")
	}

	if err != nil {
		return nil, errors.Wrapf(err, "could not stop the transcribe. transcribe_id: %s", tr.ID)
	}

	if res.OnEndFlowID == uuid.Nil {
		return res, nil
	}

	// create activeflow
	af, err := h.reqHandler.FlowV1ActiveflowCreate(ctx, uuid.Nil, res.CustomerID, res.OnEndFlowID, fmactiveflow.ReferenceTypeTranscribe, res.ID, res.ActiveflowID)
	if err != nil {
		// we could not create the activeflow, but continue to stop the transcribe
		log.Errorf("Could not create the activeflow. err: %v", err)
		return res, nil
	}
	log.WithField("activeflow", af).Debugf("Created activeflow. activeflow_id: %s", af.ID)

	if errSet := h.variableSet(ctx, af.ID, res); errSet != nil {
		// we could not set the variable, but continue to handle the on end flow execution
		log.Errorf("Could not set the variable. err: %v", errSet)
	}

	if errExecute := h.reqHandler.FlowV1ActiveflowExecute(ctx, af.ID); errExecute != nil {
		// we could not execute the activeflow, but continue to stop the transcribe
		log.Errorf("Could not execute the activeflow. activeflow_id: %s, err: %v", af.ID, errExecute)
		return res, nil
	}

	return res, nil
}

// stopLive stops live transcribing.
func (h *transcribeHandler) stopLive(ctx context.Context, tr *transcribe.Transcribe) (*transcribe.Transcribe, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "stopLive",
		"transcribe_id": tr.ID,
	})

	for _, streamingID := range tr.StreamingIDs {
		st, err := h.streamingHandler.Stop(ctx, streamingID)
		if err != nil {
			// could not stop the streaming, but continue to stop the other streamings
			log.Infof("Could not stop the streaming. But consider already stopped. streaming_id: %s, err: %v", streamingID, err)
			continue
		}
		log.WithField("streaming", st).Debugf("Stopped streaming. streaming_id: %s", st.ID)
	}

	res, err := h.UpdateStatus(ctx, tr.ID, transcribe.StatusDone)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update the status. transcribe_id: %s", tr.ID)
	}
	log.WithField("transcribe", res).Debugf("Updated transcribe status done. transcribe_id: %s", res.ID)

	return res, nil
}
