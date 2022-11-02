package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvrequest "gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// conversation sends a request to conversation-manager
// to setup the conversation.
// it returns nil if it succeed.
func (r *requestHandler) ConversationV1Setup(ctx context.Context, customerID uuid.UUID, ReferenceType cvconversation.ReferenceType) error {
	uri := "/v1/setup"

	req := &cvrequest.V1DataSetupPost{
		CustomerID:    customerID,
		ReferenceType: ReferenceType,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestConversation(uri, rabbitmqhandler.RequestMethodPost, resourceConversationSetup, 30000, 0, ContentTypeJSON, m)
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
