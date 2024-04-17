package queuecallhandler

import (
	"context"
	"strconv"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// setVariables sets the queue's variables
func (h *queuecallHandler) setVariables(ctx context.Context, q *queue.Queue, qc *queuecall.Queuecall) error {

	variables := map[string]string{
		"voipbin.queue.id":     q.ID.String(),
		"voipbin.queue.name":   q.Name,
		"voipbin.queue.detail": q.Detail,

		"voipbin.queuecall.id":              qc.ID.String(),
		"voipbin.queuecall.timeout_wait":    strconv.Itoa(qc.TimeoutWait),
		"voipbin.queuecall.timeout_service": strconv.Itoa(qc.TimeoutService),
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, qc.ReferenceActiveflowID, variables); errSet != nil {
		return errSet
	}

	return nil
}

// deleteVariables deletes queue's variables
func (h *queuecallHandler) deleteVariables(ctx context.Context, qc *queuecall.Queuecall) error {

	keys := []string{
		"voipbin.queue.id",
		"voipbin.queue.name",
		"voipbin.queue.detail",

		"voipbin.queuecall.id",
		"voipbin.queuecall.timeout_wait",
		"voipbin.queuecall.timeout_service",
	}

	for _, key := range keys {
		if err := h.reqHandler.FlowV1VariableDeleteVariable(ctx, qc.ReferenceActiveflowID, key); err != nil {
			return err
		}
	}

	return nil
}
