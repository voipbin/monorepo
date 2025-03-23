package recordinghandler

import (
	"context"
	"fmt"
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

func (h *recordingHandler) variableUpdateToReferenceInfo(ctx context.Context, r *recording.Recording) error {
	if r.ActiveflowID == uuid.Nil {
		// the activeflow id is nil. nothing to do
		return nil
	}

	if errSet := h.variablesSet(ctx, r.ActiveflowID, r); errSet != nil {
		return errors.Wrapf(errSet, "could not set the variable")
	}

	return nil
}
