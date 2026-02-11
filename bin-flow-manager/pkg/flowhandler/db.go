package flowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
)

// maxFlowCount is the hard limit on persisted flows per customer.
//
// This limit is enforced locally in flow-manager rather than through
// billing-manager's per-tier resource limits. The original tier-based
// limits (e.g. Free=5, Basic=50) blocked temporary flow creation that
// is essential for call processing — temporary flows go through the
// same creation path but are stored only in Redis and auto-expire.
// Low per-tier limits caused all flow creation (including temporary)
// to fail once the persisted count reached the tier cap.
//
// We use a single high hard cap (10,000) for all tiers instead.
// This is an internal safety limit to prevent abuse, not a
// customer-facing restriction, so it is not exposed in billing
// tiers or public documentation.
//
// The count checks persisted (database) flows only. Temporary flows
// are excluded because they auto-expire from Redis.
const maxFlowCount = 10000

// Get returns flow
func (h *flowHandler) Get(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
		"id":   id,
	})

	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new flow
func (h *flowHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	flowType flow.Type,
	name string,
	detail string,
	persist bool,
	actions []action.Action,
	onCompleteFlowID uuid.UUID,
) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "Create",
		"customer_id":         customerID,
		"flow_type":           flowType,
		"name":                name,
		"detail":              detail,
		"persist":             persist,
		"actions":             actions,
		"on_complete_flow_id": onCompleteFlowID,
	})

	// Check the hard flow count limit locally instead of calling billing-manager's
	// per-tier resource limit. See the maxFlowCount comment for rationale.
	count, errCount := h.db.FlowCountByCustomerID(ctx, customerID)
	if errCount != nil {
		log.Errorf("Could not get flow count. err: %v", errCount)
		return nil, fmt.Errorf("could not get flow count: %w", errCount)
	}
	if count >= maxFlowCount {
		log.Infof("Flow hard limit reached for customer. customer_id: %s, count: %d, limit: %d", customerID, count, maxFlowCount)
		return nil, fmt.Errorf("resource limit exceeded")
	}

	// generates the actions
	a, err := h.actionHandler.GenerateFlowActions(ctx, actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, err
	}

	id := h.util.UUIDCreate()
	f := &flow.Flow{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Type: flowType,

		Name:   name,
		Detail: detail,

		Persist: persist,

		Actions: a,

		OnCompleteFlowID: onCompleteFlowID,

		TMCreate: h.util.TimeNow(),
		TMUpdate: nil,
		TMDelete: nil,
	}
	log.WithField("flow", f).Debug("Creating a new flow.")

	switch {
	case f.Persist:
		if err := h.db.FlowCreate(ctx, f); err != nil {
			log.Errorf("Could not create the flow in the database. err: %v", err)
			return nil, err
		}

	default:
		if err := h.db.FlowSetToCache(ctx, f); err != nil {
			log.Errorf("Could not create the flow in the cache. err: %v", err)
			return nil, err
		}
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowCreated, res)

	return res, nil
}

// List returns list of flows
func (h *flowHandler) List(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "List",
		"token": token,
		"size":  size,
		"limit": size,
	})

	res, err := h.db.FlowList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get flows. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Update updates the flow info and return the updated flow
func (h *flowHandler) Update(ctx context.Context, id uuid.UUID, name string, detail string, actions []action.Action, onCompleteFlowID uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "Update",
		"flow_id":             id,
		"name":                name,
		"detail":              detail,
		"actions":             actions,
		"on_complete_flow_id": onCompleteFlowID,
	})
	log.Debug("Updating the flow.")

	// generates the tmpActions
	tmpActions, err := h.actionHandler.GenerateFlowActions(ctx, actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, err
	}
	log.WithField("new_actions", tmpActions).Debug("Created the new actions tmp.")

	fields := map[flow.Field]any{
		flow.FieldName:             name,
		flow.FieldDetail:           detail,
		flow.FieldActions:          tmpActions,
		flow.FieldOnCompleteFlowID: onCompleteFlowID,
	}

	if err := h.db.FlowUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update the flow info. err: %v", err)
		return nil, err
	}

	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowUpdated, res)

	return res, nil
}

// Delete deletes the flow
// And it also removes the related flow_id from the number-manager
func (h *flowHandler) Delete(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Delete",
		"flow_id": id,
	})
	log.Debug("Deleting the flow.")

	err := h.db.FlowDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the flow. err: %v", err)
		return nil, err
	}

	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted flow. err: %v", err)
		return nil, fmt.Errorf("could not get deleted flow")
	}
	h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowDeleted, res)

	return res, nil
}

// UpdateActions updates the actions and return the updated flow
func (h *flowHandler) UpdateActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*flow.Flow, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "UpdateActions",
		"flow_id": id,
		"actions": actions,
	})
	log.Debug("Updating the flow actions.")

	// generates the tmpActions
	tmpActions, err := h.actionHandler.GenerateFlowActions(ctx, actions)
	if err != nil {
		log.Errorf("Could not generate the flow actions. err: %v", err)
		return nil, err
	}
	log.WithField("new_actions", tmpActions).Debug("Created the new actions tmp.")

	fileds := map[flow.Field]any{
		flow.FieldActions: tmpActions,
	}

	if errUpdate := h.db.FlowUpdate(ctx, id, fileds); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the flow actions. flow_id: %s", id)
	}

	res, err := h.db.FlowGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated flow. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, flow.EventTypeFlowUpdated, res)

	return res, nil
}
