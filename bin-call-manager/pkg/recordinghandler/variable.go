package recordinghandler

import (
	"context"
	"fmt"
	"maps"
	"monorepo/bin-call-manager/models/recording"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *recordingHandler) variablesGet(ctx context.Context, c *recording.Recording) map[string]string {
	filenames := strings.Join(c.Filenames, ",")

	return map[string]string{

		variableRecordingID: c.ID.String(),

		variableRecordingReferenceType: string(c.ReferenceType),
		variableRecordingReferenceID:   c.ReferenceID.String(),
		variableRecordingFormat:        string(c.Format),

		variableRecordingRecordingName: c.RecordingName,
		variableRecordingFilenames:     filenames,
	}
}

// variablesSet sets the variables
func (h *recordingHandler) variablesSet(ctx context.Context, activeflowID uuid.UUID, r *recording.Recording) error {

	variables := h.variablesGet(ctx, r)

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}

func (h *recordingHandler) variableUpdateFromReference(ctx context.Context, r *recording.Recording, activeflowID uuid.UUID) error {

	var variables map[string]string
	var err error
	switch r.ReferenceType {
	case recording.ReferenceTypeCall:
		variables, err = h.variableGetReferenceTypeCall(ctx, r)

	case recording.ReferenceTypeConfbridge:
		variables, err = h.variableGetReferenceTypeConfbridge(ctx, r)

	default:
		return fmt.Errorf("unsupported reference type. reference_type: %s", r.ReferenceType)
	}
	if err != nil {
		return errors.Wrapf(err, "could not get variables for reference info")
	}

	// get and overwrite variables for current activeflow
	curVariables, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return errors.Wrapf(err, "could not get variables for current activeflow")
	}
	maps.Copy(variables, curVariables.Variables)

	// get and overwrite variables for the recording
	recVariables := h.variablesGet(ctx, r)
	maps.Copy(variables, recVariables)

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return errors.Wrapf(errSet, "could not set variables")
	}

	return nil
}

func (h *recordingHandler) variableGetReferenceTypeCall(ctx context.Context, r *recording.Recording) (map[string]string, error) {
	c, err := h.reqHandler.CallV1CallGet(ctx, r.ReferenceID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get call info")
	}

	res, err := h.reqHandler.FlowV1VariableGet(ctx, c.ActiveFlowID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get variables")
	}

	return res.Variables, nil
}

func (h *recordingHandler) variableGetReferenceTypeConfbridge(ctx context.Context, r *recording.Recording) (map[string]string, error) {
	// todo: need to be implemented

	return map[string]string{}, nil
}
