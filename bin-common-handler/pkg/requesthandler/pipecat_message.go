package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	outline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pcrequest "monorepo/bin-pipecat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

func (r *requestHandler) PipecatV1MessageSend(
	ctx context.Context,
	hostID string,
	pipecatcallID uuid.UUID,
	messageID string,
	messageText string,
	runImmediately bool,
	audioResponse bool,
) (*pmmessage.Message, error) {
	uri := "/v1/messages"

	data := &pcrequest.V1DataMessagesPost{
		PipecatcallID:  pipecatcallID,
		MessageID:      messageID,
		MessageText:    messageText,
		RunImmediately: runImmediately,
		AudioResponse:  audioResponse,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	queueName := fmt.Sprintf("%s.%s", outline.QueueNamePipecatRequest, hostID)
	tmp, err := r.sendRequest(ctx, outline.QueueName(queueName), uri, sock.RequestMethodPost, "pipecat/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res pmmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
