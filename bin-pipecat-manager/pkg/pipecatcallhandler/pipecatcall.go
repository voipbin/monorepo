package pipecatcallhandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"

	"github.com/gofrs/uuid"
)

func (h *pipecatcallHandler) Create(
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
	h.muPipecatcall.Lock()
	defer h.muPipecatcall.Unlock()

	id := h.utilHandler.UUIDCreate()
	res := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		LLM:      llm,
		STT:      stt,
		TTS:      tts,
		VoiceID:  voiceID,
		Messages: messages,

		RunnerWebsocketChan: make(chan *pipecatframe.Frame, 100),
	}

	_, ok := h.mapPipecatcall[id]
	if ok {
		return nil, fmt.Errorf("streaming already exists. streaming_id: %s", id)
	}

	h.mapPipecatcall[id] = res
	h.notifyHandler.PublishEvent(ctx, pipecatcall.EventTypeCreated, res)

	return res, nil
}

func (h *pipecatcallHandler) Get(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	h.muPipecatcall.Lock()
	defer h.muPipecatcall.Unlock()

	res, ok := h.mapPipecatcall[id]
	if !ok {
		return nil, fmt.Errorf("streaming not found. streaming_id: %s", id)
	}

	return res, nil
}

func (h *pipecatcallHandler) Delete(ctx context.Context, streamingID uuid.UUID) {
	h.muPipecatcall.Lock()
	defer h.muPipecatcall.Unlock()

	tmp, ok := h.mapPipecatcall[streamingID]
	if !ok {
		return
	}

	delete(h.mapPipecatcall, streamingID)
	h.notifyHandler.PublishEvent(ctx, pipecatcall.EventTypeDeleted, tmp)
}
