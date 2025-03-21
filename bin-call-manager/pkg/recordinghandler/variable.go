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

func (h *recordingHandler) variablesGet(r *recording.Recording) map[string]string {
	filenames := strings.Join(r.Filenames, ",")

	return map[string]string{

		variableRecordingID: r.ID.String(),

		variableRecordingReferenceType: string(r.ReferenceType),
		variableRecordingReferenceID:   r.ReferenceID.String(),
		variableRecordingFormat:        string(r.Format),

		variableRecordingRecordingName: r.RecordingName,
		variableRecordingFilenames:     filenames,
	}
}

// variablesSet sets the variables
func (h *recordingHandler) variablesSet(ctx context.Context, activeflowID uuid.UUID, r *recording.Recording) error {

	variables := h.variablesGet(r)

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}

func (h *recordingHandler) variableUpdateFromReferenceInfo(ctx context.Context, r *recording.Recording, activeflowID uuid.UUID) error {

	referenceActiveflowID := h.variableGetActiveflowID(ctx, r)
	if referenceActiveflowID == uuid.Nil {
		return fmt.Errorf("could not get activeflow id")
	}

	tmp, err := h.reqHandler.FlowV1VariableGet(ctx, referenceActiveflowID)
	if err != nil {
		return errors.Wrapf(err, "could not get variables")
	}
	variables := tmp.Variables

	// get and overwrite variables for current activeflow
	curVariables, err := h.reqHandler.FlowV1VariableGet(ctx, activeflowID)
	if err != nil {
		return errors.Wrapf(err, "could not get variables for current activeflow")
	}
	maps.Copy(variables, curVariables.Variables)

	// get and overwrite variables for the recording
	recVariables := h.variablesGet(r)
	maps.Copy(variables, recVariables)

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return errors.Wrapf(errSet, "could not set variables")
	}

	return nil
}

func (h *recordingHandler) variableGetActiveflowID(ctx context.Context, r *recording.Recording) uuid.UUID {
	switch r.ReferenceType {
	case recording.ReferenceTypeCall:
		c, err := h.reqHandler.CallV1CallGet(ctx, r.ReferenceID)
		if err != nil {
			return uuid.Nil
		}
		return c.ActiveFlowID

	case recording.ReferenceTypeConfbridge:
		return uuid.Nil

	default:
		return uuid.Nil
	}
}

func (h *recordingHandler) variableUpdateToReferenceInfo(ctx context.Context, r *recording.Recording) error {
	activeflowID := h.variableGetActiveflowID(ctx, r)
	if activeflowID == uuid.Nil {
		return fmt.Errorf("could not get activeflow id")
	}

	if errSet := h.variablesSet(ctx, activeflowID, r); errSet != nil {
		return errors.Wrapf(errSet, "could not set the variable")
	}

	return nil
}
