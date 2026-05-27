package aicallhandler

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Create is creating a new aicall.
// it increases corresponded counter
func (h *aicallHandler) Create(
	ctx context.Context,
	c *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	pipecatcallID uuid.UUID,
	currentMemberID uuid.UUID,
	parameter map[string]any,
	metadata map[string]any,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
		"ai":   c,
	})

	id := h.utilHandler.UUIDCreate()
	tmp := &aicall.AIcall{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: c.CustomerID,
		},

		AssistanceType: assistanceType,
		AssistanceID:   assistanceID,

		AIEngineModel: c.EngineModel,
		AITTSType:     c.TTSType,
		AITTSVoiceID:  c.TTSVoiceID,
		AISTTType:     c.STTType,
		AIVADConfig:        c.VADConfig,
		AISmartTurnEnabled: c.SmartTurnEnabled,

		Parameter: parameter,

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		ConfbridgeID:    confbridgeID,
		PipecatcallID:   pipecatcallID,
		CurrentMemberID: currentMemberID,

		STTLanguage: c.STTLanguage,

		Metadata: metadata,

		Status: aicall.StatusInitiating,
	}
	log = log.WithField("aicall_id", id.String())
	log.WithField("aicall", tmp).Debugf("Creating aicall. aicall_id: %s", tmp.ID)

	if errCreate := h.db.AIcallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new aicall. err: %v", errCreate)
		return nil, errCreate
	}
	promAIcallCreateTotal.WithLabelValues(string(tmp.ReferenceType)).Inc()

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created aicall info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusInitializing, res)

	// todo: start health check

	return res, nil
}

// CreateByMessaging creates a new aicall for non-realtime (messaging) paths.
// It only sets AIEngineModel from the AI config and does not set TTS/STT/VAD fields.
func (h *aicallHandler) CreateByMessaging(
	ctx context.Context,
	c *ai.AI,
	assistanceType aicall.AssistanceType,
	assistanceID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	pipecatcallID uuid.UUID,
	currentMemberID uuid.UUID,
	parameter map[string]any,
	metadata map[string]any,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "CreateByMessaging",
		"ai":   c,
	})

	id := h.utilHandler.UUIDCreate()
	tmp := &aicall.AIcall{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: c.CustomerID,
		},

		AssistanceType: assistanceType,
		AssistanceID:   assistanceID,

		AIEngineModel: c.EngineModel,

		Parameter: parameter,

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		ConfbridgeID:    uuid.Nil,
		PipecatcallID:   pipecatcallID,
		CurrentMemberID: currentMemberID,

		STTLanguage: c.STTLanguage,

		Metadata: metadata,

		Status: aicall.StatusInitiating,
	}
	log = log.WithField("aicall_id", id.String())
	log.WithField("aicall", tmp).Debugf("Creating aicall by messaging. aicall_id: %s", tmp.ID)

	if errCreate := h.db.AIcallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new aicall. err: %v", errCreate)
		return nil, errCreate
	}
	promAIcallCreateTotal.WithLabelValues(string(tmp.ReferenceType)).Inc()

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created aicall info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusInitializing, res)

	// todo: start health check

	return res, nil
}

// Get is handy function for getting a aicall.
func (h *aicallHandler) Get(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Get",
			"aicall_id": id,
		},
	)

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get aicall info. err: %v", err)
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameAIManager,
				"AICALL_NOT_FOUND",
				"The AI call was not found.",
			).Wrap(err)
		}
		return nil, err
	}

	return res, nil
}

// Delete deletes the aicall.
func (h *aicallHandler) Delete(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Delete",
			"aicall_id": id,
		},
	)

	if err := h.db.AIcallDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the aicall. err: %v", err)
		return nil, err
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated aicall. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID returns a aicall by the reference_id.
func (h *aicallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.db.AIcallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// List returns list of aicalls.
func (h *aicallHandler) List(ctx context.Context, size uint64, token string, filters map[aicall.Field]any) ([]*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "List",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.AIcallList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get aicalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *aicallHandler) UpdatePipecatcallID(ctx context.Context, id uuid.UUID, pipecatcallID uuid.UUID) (*aicall.AIcall, error) {
	fields := map[aicall.Field]any{
		aicall.FieldPipecatcallID: pipecatcallID,
	}
	if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the pipecatcall id for existing aicall. aicall_id: %s", id)
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	return res, nil
}

// UpdateActiveflowID updates the activeflow_id for the aicall. Used when a
// long-lived AIcall is reused across multiple per-message activeflows
// (conversation chat) so tools that read flow variables target the current flow.
func (h *aicallHandler) UpdateActiveflowID(ctx context.Context, id uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error) {
	fields := map[aicall.Field]any{
		aicall.FieldActiveflowID: activeflowID,
	}
	if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the activeflow id for aicall. aicall_id: %s", id)
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	return res, nil
}

// UpdatePipecatcallIDAndActiveflowID atomically updates both PipecatcallID and
// ActiveflowID in a single DB write. Used in the conversation reuse branch to
// avoid the race where a concurrent reader observes the new PipecatcallID
// alongside a stale ActiveflowID.
func (h *aicallHandler) UpdatePipecatcallIDAndActiveflowID(ctx context.Context, id uuid.UUID, pipecatcallID uuid.UUID, activeflowID uuid.UUID) (*aicall.AIcall, error) {
	fields := map[aicall.Field]any{
		aicall.FieldPipecatcallID: pipecatcallID,
		aicall.FieldActiveflowID:  activeflowID,
	}
	if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update pipecatcall_id+activeflow_id for aicall. aicall_id: %s", id)
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	return res, nil
}

// UpdateCurrentMemberID updates the current member id for the aicall
func (h *aicallHandler) UpdateCurrentMemberID(ctx context.Context, id uuid.UUID, currentMemberID uuid.UUID) (*aicall.AIcall, error) {
	fields := map[aicall.Field]any{
		aicall.FieldCurrentMemberID: currentMemberID,
	}
	if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the current member id for aicall. aicall_id: %s", id)
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	return res, nil
}

// UpdateStatus updates the aicall status and emits the corresponding webhook event.
func (h *aicallHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status aicall.Status) (*aicall.AIcall, error) {
	fields := map[aicall.Field]any{
		aicall.FieldStatus: status,
	}
	if status == aicall.StatusTerminated {
		now := h.utilHandler.TimeNow()
		fields[aicall.FieldTMEnd] = now
	}
	if errUpdate := h.db.AIcallUpdate(ctx, id, fields); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the aicall status. aicall_id: %s", id)
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	switch status {
	case aicall.StatusProgressing:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusProgressing, res)
	case aicall.StatusPausing:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusPausing, res)
	case aicall.StatusResuming:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusResuming, res)
	case aicall.StatusTerminating:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusTerminating, res)
	case aicall.StatusTerminated:
		promAIcallEndTotal.WithLabelValues(string(res.ReferenceType)).Inc()
		if res.TMCreate != nil {
			promAIcallDurationSeconds.WithLabelValues(string(res.ReferenceType)).Observe(time.Since(*res.TMCreate).Seconds())
		}
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusTerminated, res)
	}

	return res, nil
}
