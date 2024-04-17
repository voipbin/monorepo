package variablehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/variable"
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

func (h *variableHandler) Set(ctx context.Context, t *variable.Variable) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Set",
		"variable_id": t.ID,
	})

	if err := h.db.VariableUpdate(ctx, t); err != nil {

		log.Errorf("Could not update the variable. err: %v", err)
		return err

	}

	return nil
}

// SetVariable sets the variable with value
// func (h *variableHandler) SetVariable(ctx context.Context, id uuid.UUID, variables map[string]string  key string, value string) error {
func (h *variableHandler) SetVariable(ctx context.Context, id uuid.UUID, variables map[string]string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SetVariable",
		"variable_id": id,
	})
	log.WithField("variables", variables).Debug("Setting a variable.")

	// get variable
	vars, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get variable. err: %v", err)
		return err
	}

	for k, v := range variables {
		val := h.SubstituteString(ctx, v, vars)
		vars.Variables[k] = val
	}

	// val := h.SubstituteString(ctx, value, vars)
	// vars.Variables[key] = val

	if err := h.Set(ctx, vars); err != nil {
		log.Errorf("Could not set variable. err: %v", err)
		return err
	}

	return nil
}

// DeleteVariable deletes the variable
func (h *variableHandler) DeleteVariable(ctx context.Context, id uuid.UUID, key string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DeleteVariable",
		"variable_id": id,
		"key":         key,
	})
	log.Debugf("Deleting a variable.")

	// get variable
	v, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get variable. err: %v", err)
		return err
	}

	delete(v.Variables, key)
	if err := h.Set(ctx, v); err != nil {
		log.Errorf("Could not set variable. err: %v", err)
		return err
	}

	return nil
}
