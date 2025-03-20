package recordinghandler

import (
	"context"
	"fmt"
	"monorepo/bin-call-manager/models/recording"
	"strings"

	"github.com/gofrs/uuid"
)

// setVariablesCall sets the variables
func (h *recordingHandler) setVariablesCall(ctx context.Context, activeflowID uuid.UUID, c *recording.Recording) error {

	filenames := strings.Join(c.Filenames, ",")

	variables := map[string]string{

		variableRecordingID: c.ID.String(),

		variableRecordingReferenceType: string(c.ReferenceType),
		variableRecordingReferenceID:   c.ReferenceID.String(),
		variableRecordingFormat:        string(c.Format),

		variableRecordingRecordingName: c.RecordingName,
		variableRecordingFilenames:     filenames,
	}

	if errSet := h.reqHandler.FlowV1VariableSetVariable(ctx, activeflowID, variables); errSet != nil {
		return fmt.Errorf("could not set the variable. variables: %s, err: %v", variables, errSet)
	}

	return nil
}
