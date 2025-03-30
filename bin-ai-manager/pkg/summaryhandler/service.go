package summaryhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/summary"
	commonservice "monorepo/bin-common-handler/models/service"
	fmaction "monorepo/bin-flow-manager/models/action"
)

// ServiceStart is starting a new service for summary.
// it increases corresponded counter
func (h *summaryHandler) ServiceStart(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	onEndFlowID uuid.UUID,
	referenceType summary.ReferenceType,
	referenceID uuid.UUID,
	language string,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceStart",
		"customer_id":    customerID,
		"activeflow_id":  activeflowID,
		"on_end_flow_id": onEndFlowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
	})

	// start the summary
	sm, err := h.Start(
		ctx,
		customerID,
		activeflowID,
		onEndFlowID,
		referenceType,
		referenceID,
		language,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not start the summary. activeflow_id: %s", activeflowID)
	}
	log.WithField("summary", sm).Debugf("Started the summary. summary_id: %s", sm.ID)

	res := &commonservice.Service{
		ID:          sm.ID,
		Type:        commonservice.TypeAISummary,
		PushActions: []fmaction.Action{},
	}

	return res, nil
}
