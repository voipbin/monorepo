package variablehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
)

func (h *variableHandler) Create(ctx context.Context, activeflowID uuid.UUID, variables map[string]string) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Create",
		"activeflow_id": activeflowID,
	})

	v := &variable.Variable{
		ID:        activeflowID,
		Variables: variables,
	}

	if err := h.db.VariableCreate(ctx, v); err != nil {
		log.Errorf("Could not create the variable. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, activeflowID)
	if err != nil {
		log.Errorf("Could not get created variable. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *variableHandler) Get(ctx context.Context, id uuid.UUID) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Get",
		"variable_id": id,
	})

	res, err := h.db.VariableGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get variable from the database. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *variableHandler) Set(ctx context.Context, t *variable.Variable) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Set",
		"variable_id": t.ID,
	})

	res, err := h.db.VariableUpdate(ctx, t)
	if err != nil {

		log.Errorf("Could not update the variable. err: %v", err)
		return nil, err

	}

	return res, nil
}

// SetVariable sets the variable with value
func (h *variableHandler) SetVariable(ctx context.Context, id uuid.UUID, key string, value string) (*variable.Variable, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SetVariable",
		"variable_id": id,
		"key":         key,
		"value":       value,
	})

	// get variable
	v, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get variable. err: %v", err)
		return nil, err
	}

	v.Variables[key] = value
	res, err := h.Set(ctx, v)
	if err != nil {
		log.Errorf("Could not set variable. err: %v", err)
		return nil, err
	}

	return res, nil
}
