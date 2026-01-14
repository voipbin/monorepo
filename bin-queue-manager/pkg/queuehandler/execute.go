package queuehandler

import (
	"context"
	"math/rand"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
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

	filters := map[queuecall.Field]any{
		queuecall.FieldQueueID: q.ID.String(),
		queuecall.FieldStatus:  string(queuecall.StatusWaiting),
	}

	// get queuecalls
	qcs, err := h.reqHandler.QueueV1QueuecallGets(ctx, h.utilHandler.TimeGetCurTime(), 1, filters)
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
		if errStop := h.reqHandler.FlowV1ActiveflowServiceStop(ctx, qc.ReferenceActiveflowID, qc.ID, 0); errStop != nil {
			log.Errorf("Could not stop the queuecall service. err: %v", errStop)
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
