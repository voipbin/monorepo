package queuehandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// Create creates a new queue.
// waitTimeout: wait timeout(MS)
// serviceTimeout: service timeout(MS)
func (h *queueHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	routingMethod queue.RoutingMethod,
	tagIDs []uuid.UUID,
	waitActions []fmaction.Action,
	waitTimeout int,
	serviceTimeout int,
) (*queue.Queue, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})
	log.Debug("Creating a new queue.")

	// generate queue id
	id := uuid.Must(uuid.NewV4())
	log = log.WithField("queue_id", id)

	if routingMethod != queue.RoutingMethodRandom {
		log.Errorf("Unsupported routing method. Currently, support random only. routingMethod: %s", routingMethod)
		return nil, fmt.Errorf("wrong routing_method")
	}

	// create a new queue
	a := &queue.Queue{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		RoutingMethod: routingMethod,
		TagIDs:        tagIDs,

		Execute: queue.ExecuteStop,

		WaitActions:         waitActions,
		WaitQueueCallIDs:    []uuid.UUID{},
		WaitTimeout:         waitTimeout,
		ServiceQueueCallIDs: []uuid.UUID{},
		ServiceTimeout:      serviceTimeout,

		TotalIncomingCount:   0,
		TotalServicedCount:   0,
		TotalAbandonedCount:  0,
		TotalWaitDuration:    0,
		TotalServiceDuration: 0,

		TMCreate: getCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}

	if err := h.db.QueueCreate(ctx, a); err != nil {
		log.Errorf("Could not create a new queue. err: %v", err)
		return nil, err
	}

	res, err := h.db.QueueGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created queue info. err: %v", err)
		return nil, err
	}
	log.WithField("queue", res).Debug("Created a new queue.")

	return res, nil
}

// createQueueFlow creates a queue flow and returns created flow.
func (h *queueHandler) createQueueFlow(ctx context.Context, customerID uuid.UUID, queueID uuid.UUID, confbridgeID uuid.UUID, waitActions []fmaction.Action) (*fmflow.Flow, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":     "createQueueFlow",
			"queue_id": queueID,
		})

	// create flow actions
	actions, err := h.createQueueFlowActions(waitActions, confbridgeID)
	if err != nil {
		log.Errorf("Could not create actions. err: %v", err)
		return nil, err
	}
	log.WithField("actions", actions).Debugf("Created queue flow actions. actions: %v", actions)

	// create flow name
	flowName := fmt.Sprintf("queue-%s", queueID.String())

	// create flow
	resFlow, err := h.reqHandler.FMV1FlowCreate(ctx, customerID, fmflow.TypeQueue, flowName, "generated for queue by queue-manager.", actions, false)
	if err != nil {
		log.Errorf("Could not create a queue flow. err: %v", err)
		return nil, err
	}
	log.Debugf("Created a queue flow. res: %v", resFlow)

	return resFlow, nil
}

// createQueueFlowActions creates the actions for queue join.
func (h *queueHandler) createQueueFlowActions(waitActions []fmaction.Action, confbridgeID uuid.UUID) ([]fmaction.Action, error) {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":          "createQueueFlowActions",
			"confbridge_id": confbridgeID,
		})

	res := []fmaction.Action{}

	// append the wait actions
	if len(waitActions) == 0 {
		act := fmaction.Action{
			ID:     uuid.Must(uuid.NewV4()),
			Type:   fmaction.TypeSleep,
			Option: []byte(`{"duration": 10000}`),
		}
		res = append(res, act)
	} else {
		for _, act := range waitActions {
			if act.ID == uuid.Nil {
				act.ID = uuid.Must(uuid.NewV4())
			}
			res = append(res, act)
		}
	}

	// set next id for loop
	res[len(res)-1].NextID = res[0].ID

	// append the confbridge join
	option := fmaction.OptionConfbridgeJoin{
		ConfbridgeID: confbridgeID,
	}
	opt, err := json.Marshal(option)
	if err != nil {
		log.Errorf("Could not marshal the option. err: %v", err)
		return nil, err
	}
	act := fmaction.Action{
		Type:   fmaction.TypeConfbridgeJoin,
		Option: opt,
	}
	res = append(res, act)

	return res, nil
}

// getForwardActionID returns action id for froward.
func (h *queueHandler) getForwardActionID(ctx context.Context, f *fmflow.Flow) (uuid.UUID, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "getForwardActionID",
			"flow_id": f.ID,
		},
	)

	res := uuid.Nil
	for _, act := range f.Actions {
		if act.Type == fmaction.TypeConfbridgeJoin {
			res = act.ID
		}
	}

	if res == uuid.Nil {
		log.Errorf("Could not find forward action id.")
		return uuid.Nil, fmt.Errorf("forward action id not found")
	}

	return res, nil
}
