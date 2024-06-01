package resourcehandler

import (
	"context"
	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Gets returns agents
func (h *resourceHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*resource.Resource, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.ResourceGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get agents info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns agent info.
func (h *resourceHandler) Get(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.ResourceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Create creates a new resource.
func (h *resourceHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	agentID uuid.UUID,
	referenceType string,
	data interface{},
) (*resource.Resource, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})
	log.Debug("Creating a new user.")

	id := h.utilHandler.UUIDCreate()

	tmp := &resource.Resource{
		ID:            id,
		CustomerID:    customerID,
		AgentID:       agentID,
		ReferenceType: referenceType,
		ReferenceID:   id,
		Data:          data,
	}

	if errCreate := h.db.ResourceCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create an agent. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "could not create an agent")
	}

	res, err := h.db.ResourceGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created resource info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the agent.
func (h *resourceHandler) Delete(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "Delete",
		"agent_id": id,
	})

	// check the agent is deletable
	if id == agent.GuestAgentID {
		return nil, errors.Errorf("agent is guest agent")
	}

	onlyAdmin, err := h.isOnlyAdmin(ctx, id)
	if err != nil {
		log.Errorf("Could not check the agent is the only admin. erR: %v", err)
		return nil, errors.Wrapf(err, "could not check the agent is the only admin")
	} else if onlyAdmin {
		return nil, errors.Errorf("the agent is the only admin")
	}

	res, err := h.deleteForce(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the agent. err: %v", err)
		return nil, errors.Wrap(err, "could not delete the agent")
	}

	return res, nil
}
