package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	pcrequest "monorepo/bin-pipecat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

func (r *requestHandler) PipecatV1PipecatcallStart(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType pmpipecatcall.ReferenceType,
	referenceID uuid.UUID,
	llmType pmpipecatcall.LLMType,
	llmMessages []map[string]any,
	sttType pmpipecatcall.STTType,
	ttsType pmpipecatcall.TTSType,
	ttsVoiceID string,
) (*pmpipecatcall.Pipecatcall, error) {
	uri := "/v1/pipecatcalls"

	data := &pcrequest.V1DataPipecatcallsPost{
		ID:           id,
		CustomerID:   customerID,
		ActiveflowID: activeflowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		LLMType:     llmType,
		LLMMessages: llmMessages,
		STTType:     sttType,
		TTSType:     ttsType,
		TTSVoiceID:  ttsVoiceID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestPipecat(ctx, uri, sock.RequestMethodPost, "pipecat/pipecatcalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res pmpipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) PipecatV1PipecatcallGet(ctx context.Context, pipecallID uuid.UUID) (*pmpipecatcall.Pipecatcall, error) {
	uri := fmt.Sprintf("/v1/pipecatcalls/%s", pipecallID)

	tmp, err := r.sendRequestPipecat(ctx, uri, sock.RequestMethodGet, "pipecat/pipecatcalls/<pipecatcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pmpipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// PipecatV1PipecatcallTerminate sends a request to pipecat-manager
// to terminate an pipecatcall.
// it returns pipecatcall if it succeed.
func (r *requestHandler) PipecatV1PipecatcallTerminate(ctx context.Context, hostID string, aicallID uuid.UUID) (*pmpipecatcall.Pipecatcall, error) {
	uri := fmt.Sprintf("/v1/pipecatcalls/%s/stop", aicallID)

	queueName := fmt.Sprintf("%s.%s", outline.QueueNamePipecatRequest, hostID)
	tmp, err := r.sendRequest(ctx, outline.QueueName(queueName), uri, sock.RequestMethodPost, "pipecat/pipecatcalls/<pipecatcall-id>/terminate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pmpipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
