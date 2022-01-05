package queuecallhandler

import (
	"context"
	"math/rand"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
)

// Execute search the available agent and dial to them.
func (h *queuecallHandler) Execute(ctx context.Context, queuecallID uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Execute",
			"queuecall_id": queuecallID,
		},
	)

	// get queuecall
	qc, err := h.db.QueuecallGet(ctx, queuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallExecute(ctx, queuecallID, defaultDelayQueuecallExecute)
		return
	}
	log.WithField("queuecall", qc).Debug("Found queuecall info.")

	// check the status
	if qc.Status != queuecall.StatusWait {
		// no need to execute anymore.
		return
	}

	// get agents
	agents, err := h.reqHandler.AMV1AgentGetsByTagIDsAndStatus(ctx, qc.UserID, qc.TagIDs, amagent.StatusAvailable)
	if err != nil {
		log.Errorf("Could not get available agents. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallExecute(ctx, qc.ID, defaultDelayQueuecallExecute)
		return
	} else if len(agents) == 0 {
		log.Info("No available agent now. Send the request again with 1 sec delay.")
		_ = h.reqHandler.QMV1QueuecallExecute(ctx, qc.ID, defaultDelayQueuecallExecute)
		return
	}

	targetAgent := amagent.Agent{}
	switch qc.RoutingMethod {
	case queue.RoutingMethodRandom:
		targetAgent = agents[rand.Intn(len(agents))]

	default:
		log.Errorf("Unsupported routing method. Exit from the queue. routing_method: %s", qc.RoutingMethod)
		if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ExitActionID, true); err != nil {
			log.Errorf("Could not forward the call. err: %v", err)
		}
		return
	}

	// dial to the agent
	if err := h.reqHandler.AMV1AgentDial(ctx, targetAgent.ID, &qc.Source, qc.ConfbridgeID); err != nil {
		log.Errorf("Could not dial to the agent. Send the request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QMV1QueuecallExecute(ctx, qc.ID, defaultDelayQueuecallExecute)
		return
	}

	// forward the action.
	if err := h.reqHandler.FMV1ActvieFlowUpdateForwardActionID(ctx, qc.ReferenceID, qc.ForwardActionID, true); err != nil {
		log.Errorf("Could not forward the active flow. err: %v", err)
		return
	}

	// update the queuecall
	if err := h.db.QueuecallSetServiceAgentID(ctx, qc.ID, targetAgent.ID); err != nil {
		log.Errorf("Could not ser the service agent id. err: %v", err)
		return
	}

	// get updated queuecall and notify
	tmp, err := h.db.QueuecallGet(ctx, qc.ID)
	if err != nil {
		log.Errorf("Could not get updated queuecall. err: %v", err)
		return
	}
	h.notifyhandler.NotifyEvent(ctx, notifyhandler.EventTypeQueuecallServiced, tmp.WebhookURI, tmp)

	// get wait duration and increase the serviced count
	waitDuration := getDuration(ctx, tmp.TMCreate, tmp.TMService)
	if err := h.db.QueueIncreaseTotalServicedCount(ctx, tmp.QueueID, tmp.ID, waitDuration); err != nil {
		log.Errorf("Could not increase the total serviced count. err: %v", err)
	}

	// send the queuecall timeout-service
	if tmp.TimeoutService > 0 {
		if err := h.reqHandler.QMV1QueuecallTiemoutService(ctx, tmp.ID, tmp.TimeoutService); err != nil {
			log.Errorf("Could not send the timeout-service request. err: %v", err)
		}
	}

}
