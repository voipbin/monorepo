package pipecatcallhandler

import (
	"context"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

func (h *pipecatcallHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType pipecatcall.ReferenceType,
	referenceID uuid.UUID,
	llmType pipecatcall.LLMType,
	llmMessages []map[string]any,
	sttType pipecatcall.STTType,
	ttsType pipecatcall.TTSType,
	ttsVoiceID string,
) (*pipecatcall.Pipecatcall, error) {

	tmp := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		HostID: h.hostID,

		LLMType:     llmType,
		LLMMessages: llmMessages,
		STTType:     sttType,
		TTSType:     ttsType,
		TTSVoiceID:  ttsVoiceID,
	}

	if errCreate := h.db.PipecatcallCreate(ctx, tmp); errCreate != nil {
		return nil, errors.Wrapf(errCreate, "could not create pipecatcall in db")
	}

	res, err := h.db.PipecatcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall after creation")
	}

	h.notifyHandler.PublishEvent(ctx, pipecatcall.EventTypeCreated, res)
	return res, nil
}

func (h *pipecatcallHandler) Get(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	res, err := h.db.PipecatcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall from db")
	}

	return res, nil
}

func (h *pipecatcallHandler) Delete(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	if errDelete := h.db.PipecatcallDelete(ctx, id); errDelete != nil {
		return nil, errors.Wrapf(errDelete, "could not delete pipecatcall in db")
	}

	res, err := h.db.PipecatcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pipecatcall after deletion")
	}

	h.notifyHandler.PublishEvent(ctx, pipecatcall.EventTypeDeleted, res)
	return res, nil
}
