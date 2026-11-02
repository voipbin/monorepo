package servicehandler

import (
	"context"
	stderrors "errors"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/requesthandler"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ServiceAgentCallQueuecallGet returns the queue entry of a call the agent owns.
//
// The call is the server-side call (the agent leg) returned from
// ServiceAgentCallList. The queuecall is found by the call's master_call_id
// (the customer leg the queue was joined with) first, then by the call id
// itself (an agent that joined the queue directly). Every miss, whether the
// call has no queue entry, the entry belongs to another agent or customer, or
// the queue is gone, returns the bare serviceerrors.ErrNotFound so a caller
// cannot probe for the existence of other entries.
func (h *serviceHandler) ServiceAgentCallQueuecallGet(ctx context.Context, a *auth.AuthIdentity, callID uuid.UUID) (*openapi_server.QueueManagerQueuecallSummary, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentCallQueuecallGet",
		"customer_id": a.CustomerID,
		"agent_id":    a.AgentID(),
		"call_id":     callID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	c, err := h.callGet(ctx, callID)
	if err != nil {
		log.Infof("Could not get call info. err: %v", err)
		return nil, err
	}

	// the call must belong to this customer and be owned by this agent. An
	// empty owner never matches (fail-closed).
	if c.CustomerID != a.CustomerID || a.AgentID() == uuid.Nil || c.OwnerID != a.AgentID() {
		log.Info("The call is not owned by the agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	candidates := []uuid.UUID{}
	if c.MasterCallID != uuid.Nil {
		candidates = append(candidates, c.MasterCallID)
	}
	candidates = append(candidates, c.ID)

	var qc *qmqueuecall.Queuecall
	for _, referenceID := range candidates {
		tmp, errGet := h.reqHandler.QueueV1QueuecallGetByReferenceID(ctx, referenceID)
		if errGet != nil {
			if queuecallIsNotFound(errGet) {
				continue
			}
			log.Errorf("Could not get the queuecall. reference_id: %s, err: %v", referenceID, errGet)
			return nil, errors.Wrapf(errGet, "could not get the queuecall info")
		}

		qc = tmp
		break
	}
	if qc == nil {
		return nil, serviceerrors.ErrNotFound
	}

	// the entry must belong to this customer and have been handled by this agent.
	if qc.CustomerID != a.CustomerID || qc.ServiceAgentID != a.AgentID() {
		log.Info("The queuecall does not belong to the agent.")
		return nil, serviceerrors.ErrNotFound
	}

	// a deleted queue still returns its name.
	q, err := h.reqHandler.QueueV1QueueGet(ctx, qc.QueueID)
	if err != nil {
		if queuecallIsNotFound(err) {
			return nil, serviceerrors.ErrNotFound
		}
		log.Errorf("Could not get the queue. queue_id: %s, err: %v", qc.QueueID, err)
		return nil, errors.Wrapf(err, "could not get the queue info")
	}
	if q.CustomerID != a.CustomerID {
		log.Info("The queue does not belong to the customer.")
		return nil, serviceerrors.ErrNotFound
	}

	id := qc.ID.String()
	queueID := qc.QueueID.String()
	status := openapi_server.QueueManagerQueuecallStatus(qc.Status)
	return &openapi_server.QueueManagerQueuecallSummary{
		Id:        &id,
		QueueId:   &queueID,
		QueueName: &q.Name,
		Status:    &status,
		TmCreate:  qc.TMCreate,
	}, nil
}

// queuecallIsNotFound reports whether an RPC error means the target does not exist.
func queuecallIsNotFound(err error) bool {
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound {
		return true
	}

	return stderrors.Is(err, requesthandler.ErrNotFound)
}
