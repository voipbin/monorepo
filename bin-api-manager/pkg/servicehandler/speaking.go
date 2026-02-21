package servicehandler

import (
	"context"
	"fmt"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmspeaking "monorepo/bin-tts-manager/models/speaking"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// speakingGet gets a speaking record.
func (h *serviceHandler) speakingGet(ctx context.Context, speakingID uuid.UUID) (*tmspeaking.Speaking, error) {
	res, err := h.reqHandler.TTSV1SpeakingGet(ctx, speakingID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SpeakingCreate creates a new speaking session.
func (h *serviceHandler) SpeakingCreate(ctx context.Context, a *amagent.Agent, referenceType tmstreaming.ReferenceType, referenceID uuid.UUID, language string, provider string, voiceID string, direction tmstreaming.Direction) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingCreate(ctx, a.CustomerID, referenceType, referenceID, language, provider, voiceID, direction)
	if err != nil {
		log.Errorf("Could not create speaking. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", tmp).Debugf("Created speaking. speaking_id: %s", tmp.ID)

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingGet retrieves a speaking session.
func (h *serviceHandler) SpeakingGet(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	tmp, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingList retrieves a list of speaking sessions.
func (h *serviceHandler) SpeakingList(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingList",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	filters := map[tmspeaking.Field]any{
		tmspeaking.FieldCustomerID: a.CustomerID,
		tmspeaking.FieldDeleted:    false,
	}

	tmps, err := h.reqHandler.TTSV1SpeakingGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get speaking list. err: %v", err)
		return nil, err
	}

	res := make([]*tmspeaking.WebhookMessage, len(tmps))
	for i, s := range tmps {
		res[i] = s.ConvertWebhookMessage()
	}
	return res, nil
}

// SpeakingSay sends text to be spoken. Pod-targeted.
func (h *serviceHandler) SpeakingSay(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID, text string) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingSay",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", s).Debugf("Retrieved speaking info. speaking_id: %s", s.ID)

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingSay(ctx, s.PodID, speakingID, text)
	if err != nil {
		log.Errorf("Could not say text. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingFlush flushes pending text from the speaking queue. Pod-targeted.
func (h *serviceHandler) SpeakingFlush(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingFlush",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", s).Debugf("Retrieved speaking info. speaking_id: %s", s.ID)

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingFlush(ctx, s.PodID, speakingID)
	if err != nil {
		log.Errorf("Could not flush speaking. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingStop stops the speaking session. Pod-targeted.
func (h *serviceHandler) SpeakingStop(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingStop",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", s).Debugf("Retrieved speaking info. speaking_id: %s", s.ID)

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.TTSV1SpeakingStop(ctx, s.PodID, speakingID)
	if err != nil {
		log.Errorf("Could not stop speaking. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// SpeakingDelete soft-deletes a speaking session.
// Stops the streaming session first (pod-targeted) before deleting the DB record (shared queue).
func (h *serviceHandler) SpeakingDelete(ctx context.Context, a *amagent.Agent, speakingID uuid.UUID) (*tmspeaking.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "SpeakingDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"speaking_id": speakingID,
	})

	s, err := h.speakingGet(ctx, speakingID)
	if err != nil {
		log.Errorf("Could not get speaking info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, s.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// Stop the streaming session first (pod-targeted) to ensure proper cleanup
	if s.Status == tmspeaking.StatusActive || s.Status == tmspeaking.StatusInitiating {
		if _, errStop := h.reqHandler.TTSV1SpeakingStop(ctx, s.PodID, speakingID); errStop != nil {
			log.Errorf("Could not stop speaking before delete. err: %v", errStop)
		}
	}

	tmp, err := h.reqHandler.TTSV1SpeakingDelete(ctx, speakingID)
	if err != nil {
		log.Infof("Could not delete speaking. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}
