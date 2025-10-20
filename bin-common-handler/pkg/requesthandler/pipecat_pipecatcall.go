package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	pcrequest "monorepo/bin-pipecat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

func (r *requestHandler) PipecatV1PipecatcallStart(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType pipecatcall.ReferenceType,
	referenceID uuid.UUID,
	llm pipecatcall.LLM,
	stt pipecatcall.STT,
	tts pipecatcall.TTS,
	voiceID string,
	messages []map[string]any,
) (*pipecatcall.Pipecatcall, error) {
	uri := "/v1/pipecatcalls"

	data := &pcrequest.V1DataPipecatcallsPost{
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

	var res pipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) PipecatV1PipecatcallGet(ctx context.Context, pipecallID uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	uri := fmt.Sprintf("/v1/pipecatcalls/%s", pipecallID)

	tmp, err := r.sendRequestPipecat(ctx, uri, sock.RequestMethodGet, "pipecat/pipecatcalls/<pipecatcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// PipecatV1PipecatcallTerminate sends a request to pipecat-manager
// to terminate an pipecatcall.
// it returns pipecatcall if it succeed.
func (r *requestHandler) PipecatV1PipecatcallTerminate(ctx context.Context, aicallID uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	uri := fmt.Sprintf("/v1/pipecatcalls/%s/stop", aicallID)

	tmp, err := r.sendRequestPipecat(ctx, uri, sock.RequestMethodPost, "pipecat/pipecatcalls/<pipecatcall-id>/terminate", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res pipecatcall.Pipecatcall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
