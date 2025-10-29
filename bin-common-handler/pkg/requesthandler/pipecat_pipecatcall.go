package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/outline"
	commonoutline "monorepo/bin-common-handler/models/outline"
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
	llm pmpipecatcall.LLM,
	stt pmpipecatcall.STT,
	tts pmpipecatcall.TTS,
	voiceID string,
	messages []map[string]any,
) (*pmpipecatcall.Pipecatcall, error) {
	uri := "/v1/pipecatcalls"

	data := &pcrequest.V1DataPipecatcallsPost{
		ID:           id,
		CustomerID:   customerID,
		ActiveflowID: activeflowID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		LLM:      llm,
		STT:      stt,
		TTS:      tts,
		VoiceID:  voiceID,
		Messages: messages,
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

func (r *requestHandler) PipecatV1PipecatcallGet(ctx context.Context, hostID string, pipecallID uuid.UUID) (*pmpipecatcall.Pipecatcall, error) {
	uri := fmt.Sprintf("/v1/pipecatcalls/%s", pipecallID)

	queueName := fmt.Sprintf("%s.%s", outline.QueueNamePipecatRequest, hostID)
	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodGet, "pipecat/pipecatcalls/<pipecatcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
	tmp, err := r.sendRequest(ctx, commonoutline.QueueName(queueName), uri, sock.RequestMethodPost, "pipecat/pipecatcalls/<pipecatcall-id>/terminate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pmpipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
