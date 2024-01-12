package queuehandler

import (
	"context"
	"math/rand"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Execute handles queue execution.
// it checks the waiting queuecall and dials to the available agent
func (h *queueHandler) Execute(ctx context.Context, id uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Execute",
		"queue_id": id,
	})

	// get queue
	q, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get queue info. Stopping the queue execution. err: %v", err)
		_, _ = h.UpdateExecute(ctx, id, queue.ExecuteStop)
		return
	}

	// check the queue execute status
	if q.Execute == queue.ExecuteStop {
		log.Infof("The queue execution stopped. queue_id: %s", id)
		return
	}

	filters := map[string]string{
		"queue_id": q.ID.String(),
		"status":   string(queuecall.StatusWaiting),
	}

	// get queuecalls
	qcs, err := h.reqHandler.QueueV1QueuecallGets(ctx, q.CustomerID, h.utilhandler.TimeGetCurTime(), 1, filters)
	if err != nil {
		log.Errorf("Could not get queuecalls. err: %v", err)
		_ = h.reqHandler.QueueV1QueueExecuteRun(ctx, id, defaultExecuteDelay) // retry after 1 sec.
		return
	}

	if len(qcs) == 0 {
		// no more waiting queuecall left.
		// stop queue execute
		log.Debugf("No more queuecall left. Stop the queue execution. queue_id: %s", id)
		_, _ = h.UpdateExecute(ctx, id, queue.ExecuteStop)
		return
	}

	// pick target queuecall
	qc := qcs[0]
	log.WithField("queuecall", qc).Debugf("Found target queuecall. queuecall_id: %s", qc.ID)

	// get available agents
	agents, err := h.GetAgents(ctx, q.ID, amagent.StatusAvailable)
	if err != nil {
		log.Errorf("Could not get available agents. Send the queue execution request again with 1 sec delay. err: %v", err)
		_ = h.reqHandler.QueueV1QueueExecuteRun(ctx, id, defaultExecuteDelay)
		return
	}

	if len(agents) == 0 {
		log.Info("No available agent now. Send the queue execution request again with 1 sec delay.")
		_ = h.reqHandler.QueueV1QueueExecuteRun(ctx, id, defaultExecuteDelay)
		return
	}

	// pick target agent
	targetAgent := amagent.Agent{}
	switch q.RoutingMethod {
	case queue.RoutingMethodRandom:
		targetAgent = agents[rand.Intn(len(agents))]

	default:
		log.Errorf("Unsupported routing method. Exit from the queue. routing_method: %s", q.RoutingMethod)
		if errForward := h.reqHandler.FlowV1ActiveflowUpdateForwardActionID(ctx, qc.ReferenceActiveflowID, qc.ExitActionID, true); errForward != nil {
			log.Errorf("Could not forward the call. err: %v", errForward)
		}
		return
	}

	// execute the queuecall with target agent
	tmp, err := h.reqHandler.QueueV1QueuecallExecute(ctx, qc.ID, targetAgent.ID)
	if err != nil {
		log.Errorf("Could not handle the queuecall execution correctly. err: %v", err)
		_ = h.reqHandler.QueueV1QueueExecuteRun(ctx, id, defaultExecuteDelay)
		return
	}
	log.WithField("queuecall", tmp).Debugf("Executed queuecall correctly. queue_id: %s, queuecall_id: %s", q.ID, tmp.ID)

	_ = h.reqHandler.QueueV1QueueExecuteRun(ctx, id, 100)
}
