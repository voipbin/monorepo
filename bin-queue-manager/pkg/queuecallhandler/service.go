package queuecallhandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

func (h *queuecallHandler) ServiceStart(
	ctx context.Context,
	queueID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType queuecall.ReferenceType,
	referenceID uuid.UUID,
) (*commonservice.Service, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceStart",
		"queue_id":       queueID,
		"activeflow_id":  activeflowID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})
	log.Debugf("Starting queuecall service. queue_id: %s activeflow_id: %s, reference_type: %s, reference_id: %s",
		queueID,
		activeflowID,
		referenceType,
		referenceID,
	)

	// check the reference type. currently support the call type only.
	if referenceType != queuecall.ReferenceTypeCall {
		log.Errorf("Unsupported reference type")
		return nil, fmt.Errorf("unsupported reference type")
	}

	// get queue
	q, err := h.queueHandler.Get(ctx, queueID)
	if err != nil {
		log.Errorf("Could not get queue info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get queue info")
	}
	log.WithField("queue", q).Debugf("Found queue info.")

	// get call
	c, err := h.reqHandler.CallV1CallGet(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get reference info. err: %v", err)
		return nil, fmt.Errorf("reference info not found")
	}
	log.WithField("call", c).Debugf("Found call info.")

	// generate queucall id
	queuecallID := h.utilHandler.UUIDCreate()

	// create confbridge
	cb, err := h.reqHandler.CallV1ConfbridgeCreate(ctx, q.CustomerID, activeflowID, cmconfbridge.ReferenceTypeQueue, queuecallID, cmconfbridge.TypeConnect)
	if err != nil {
		log.Errorf("Could not create the confbridge for queuecall join. err: %v", err)
		return nil, errors.Wrap(err, "Could not create the confbridge for queuecall join.")
	}
	log.WithField("confbridge", cb).Debugf("Created confbridge for queuecall join. confbridge_id: %s", cb.ID)

	// get source
	source := h.getSource(c)
	log.WithField("source", source).Debugf("Source address info.")

	// create actions and forward action id
	actions, forwardActionID, err := h.createActions(ctx, q, cb.ID)
	if err != nil {
		log.Errorf("Could not create service actions. err: %v", err)
		return nil, errors.Wrap(err, "Could not create service actions")
	}

	// create a new queuecall
	qc, err := h.Create(
		ctx,
		q,
		queuecallID,
		referenceType,
		referenceID,
		activeflowID,
		forwardActionID,
		cb.ID,
		source,
	)
	if err != nil {
		log.Errorf("Could not create the queuecall. err: %v", err)
		return nil, err
	}

	res := &commonservice.Service{
		ID:          qc.ID,
		Type:        commonservice.TypeQueuecall,
		PushActions: actions,
	}

	return res, nil
}

// getSource returns source address for queuecall.
func (h *queuecallHandler) getSource(c *cmcall.Call) commonaddress.Address {

	if c.Direction == cmcall.DirectionIncoming {
		return c.Source
	}
	return c.Destination
}

// createActions creates the actions for queue join service.
func (h *queuecallHandler) createActions(ctx context.Context, q *queue.Queue, confbridgeID uuid.UUID) ([]fmaction.Action, uuid.UUID, error) {
	targetID := h.utilHandler.UUIDCreate()
	loopID := h.utilHandler.UUIDCreate()
	forwardActionID := h.utilHandler.UUIDCreate()

	res := []fmaction.Action{
		{
			ID:   targetID,
			Type: fmaction.TypeFetchFlow,
			Option: fmaction.ConvertOption(fmaction.OptionFetchFlow{
				FlowID: q.WaitFlowID,
			}),
		},
		{
			ID:     loopID,
			Type:   fmaction.TypeEmpty,
			Option: fmaction.ConvertOption(fmaction.OptionEmpty{}),
			NextID: targetID,
		},
		{
			ID:   forwardActionID,
			Type: fmaction.TypeConfbridgeJoin,
			Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
				ConfbridgeID: confbridgeID,
			}),
		},
	}
	return res, forwardActionID, nil

	// // append the action confbridge_join
	// act := fmaction.Action{
	// 	ID:   forwardActionID,
	// 	Type: fmaction.TypeConfbridgeJoin,
	// 	Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
	// 		ConfbridgeID: confbridgeID,
	// 	}),
	// }
	// res = append(res, act)

	// return res, forwardActionID, nil

	// fmaction.TypeFetchFlow

	// if len(q.WaitActions) == 0 {
	// 	// append the default wait actions for empty wait actions
	// 	act := fmaction.Action{
	// 		ID:   h.utilHandler.UUIDCreate(),
	// 		Type: fmaction.TypeSleep,
	// 		Option: fmaction.ConvertOption(fmaction.OptionSleep{
	// 			Duration: 10000,
	// 		}),
	// 	}
	// 	res = append(res, act)
	// } else {
	// 	for _, act := range q.WaitActions {
	// 		if act.ID == uuid.Nil {
	// 			act.ID = h.utilHandler.UUIDCreate()
	// 		}
	// 		res = append(res, act)
	// 	}
	// }

	// // set next id for loop
	// res[len(res)-1].NextID = res[0].ID

	// // append the action confbridge_join
	// forwardActionID := h.utilHandler.UUIDCreate()
	// act := fmaction.Action{
	// 	ID:   forwardActionID,
	// 	Type: fmaction.TypeConfbridgeJoin,
	// 	Option: fmaction.ConvertOption(fmaction.OptionConfbridgeJoin{
	// 		ConfbridgeID: confbridgeID,
	// 	}),
	// }
	// res = append(res, act)

	// return res, forwardActionID, nil
}
