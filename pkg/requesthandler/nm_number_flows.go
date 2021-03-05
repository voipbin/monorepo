package requesthandler

import (
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// FMFlowCreate sends a request to flow-manager
// to creating a flow.
// it returns created flow if it succeed.
func (r *requestHandler) NMNumberFlowDelete(flowID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/number_flows/%s", flowID)

	tmp, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodDelete, resourceNumberNumberFlows, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
