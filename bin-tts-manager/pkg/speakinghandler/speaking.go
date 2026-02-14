package speakinghandler

import (
	"context"
	"fmt"

	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// TODO: Add orphaned session cleanup on pod restart. When a pod restarts,
// any sessions with status=active/initiating and pod_id matching the old pod
// should be marked as stopped.

// Create creates a new speaking session.
func (h *speakingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language, provider, voiceID string,
	direction streaming.Direction,
) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	// Check for existing active/initiating session on same reference
	filters := map[speaking.Field]any{
		speaking.FieldCustomerID:    customerID,
		speaking.FieldReferenceType: string(referenceType),
		speaking.FieldReferenceID:   referenceID,
		speaking.FieldDeleted:       false,
	}
	existing, err := h.db.SpeakingGets(ctx, "", 100, filters)
	if err != nil {
		log.Errorf("Could not check existing sessions. err: %v", err)
		return nil, fmt.Errorf("could not check existing sessions: %v", err)
	}
	for _, s := range existing {
		if s.Status == speaking.StatusActive || s.Status == speaking.StatusInitiating {
			return nil, fmt.Errorf("an active speaking session already exists for this reference. speaking_id: %s", s.ID)
		}
	}

	if provider == "" {
		provider = string(streaming.VendorNameElevenlabs)
	}

	id := uuid.Must(uuid.NewV4())

	spk := &speaking.Speaking{
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Language:      language,
		Provider:      provider,
		VoiceID:       voiceID,
		Direction:     direction,
		Status:        speaking.StatusInitiating,
		PodID:         h.podID,
	}
	spk.ID = id
	spk.CustomerID = customerID

	if errCreate := h.db.SpeakingCreate(ctx, spk); errCreate != nil {
		log.Errorf("Could not create speaking record. err: %v", errCreate)
		return nil, fmt.Errorf("could not create speaking record: %v", errCreate)
	}
	log.WithField("speaking", spk).Debugf("Created speaking record. speaking_id: %s", id)

	// Post-create recheck: guard against concurrent session creation.
	// If two requests raced past the pre-check, the lower UUID wins.
	recheck, errRecheck := h.db.SpeakingGets(ctx, "", 100, filters)
	if errRecheck != nil {
		log.Errorf("Could not recheck sessions. err: %v", errRecheck)
	} else {
		for _, s := range recheck {
			if s.ID == id {
				continue
			}
			if (s.Status == speaking.StatusActive || s.Status == speaking.StatusInitiating) && s.ID.String() < id.String() {
				log.Infof("Concurrent session detected, backing off. winner: %s, loser: %s", s.ID, id)
				_ = h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
					speaking.FieldStatus: speaking.StatusStopped,
				})
				return nil, fmt.Errorf("concurrent session creation detected. speaking_id: %s already exists", s.ID)
			}
		}
	}

	// Start streaming session with the same ID as the speaking record
	_, errStart := h.streamingHandler.StartWithID(
		ctx,
		id,
		customerID,
		referenceType,
		referenceID,
		language,
		provider,
		voiceID,
		direction,
	)
	if errStart != nil {
		log.Errorf("Could not start streaming session. err: %v", errStart)
		_ = h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
			speaking.FieldStatus: speaking.StatusStopped,
		})
		return nil, fmt.Errorf("could not start streaming session: %v", errStart)
	}

	if errUpdate := h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
		speaking.FieldStatus: speaking.StatusActive,
	}); errUpdate != nil {
		log.Errorf("Could not update speaking status. err: %v", errUpdate)
	}

	res, errGet := h.db.SpeakingGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get speaking record. err: %v", errGet)
		return spk, nil
	}

	return res, nil
}

// Get returns a speaking session by ID.
func (h *speakingHandler) Get(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Get",
		"speaking_id": id,
	})

	res, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, err
	}
	log.WithField("speaking", res).Debugf("Retrieved speaking. speaking_id: %s", id)

	return res, nil
}

// Gets returns a list of speaking sessions.
func (h *speakingHandler) Gets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Gets",
	})

	res, err := h.db.SpeakingGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get speakings. err: %v", err)
		return nil, err
	}

	return res, nil
}

// maxTextLength is the maximum allowed text length for a single Say call.
const maxTextLength = 5000

// Say adds text to the speech queue.
func (h *speakingHandler) Say(ctx context.Context, id uuid.UUID, text string) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Say",
		"speaking_id": id,
	})

	if len(text) > maxTextLength {
		return nil, fmt.Errorf("text too long: %d characters (max %d)", len(text), maxTextLength)
	}

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if spk.Status != speaking.StatusActive {
		return nil, fmt.Errorf("session is no longer active. status: %s", spk.Status)
	}

	// SayInit ensures the vendor (ElevenLabs WebSocket) is initialized.
	// In the normal case the vendor is already initialized by runStart() when
	// the AudioSocket connection arrived, so this is a no-op. It guards against
	// the race where Say() is called before the AudioSocket connects.
	if _, errInit := h.streamingHandler.SayInit(ctx, id, uuid.Nil); errInit != nil {
		log.Errorf("Could not initialize streaming vendor. err: %v", errInit)
		return nil, fmt.Errorf("could not initialize streaming vendor: %v", errInit)
	}

	if errSay := h.streamingHandler.SayAdd(ctx, id, uuid.Nil, text); errSay != nil {
		log.Errorf("Could not add text. err: %v", errSay)
		return nil, fmt.Errorf("could not add text: %v", errSay)
	}

	return spk, nil
}

// Flush clears queued messages and stops current playback. Session stays alive.
func (h *speakingHandler) Flush(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Flush",
		"speaking_id": id,
	})

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if spk.Status != speaking.StatusActive {
		return nil, fmt.Errorf("session is no longer active. status: %s", spk.Status)
	}

	if errFlush := h.streamingHandler.SayFlush(ctx, id); errFlush != nil {
		log.Errorf("Could not flush. err: %v", errFlush)
		return nil, fmt.Errorf("could not flush: %v", errFlush)
	}

	return spk, nil
}

// Stop terminates the speaking session entirely.
func (h *speakingHandler) Stop(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Stop",
		"speaking_id": id,
	})

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if spk.Status == speaking.StatusStopped {
		return spk, nil
	}

	if _, errStop := h.streamingHandler.Stop(ctx, id); errStop != nil {
		log.Errorf("Could not stop streaming. err: %v", errStop)
	}

	if errUpdate := h.db.SpeakingUpdate(ctx, id, map[speaking.Field]any{
		speaking.FieldStatus: speaking.StatusStopped,
	}); errUpdate != nil {
		log.Errorf("Could not update speaking status. err: %v", errUpdate)
		return nil, fmt.Errorf("could not update speaking status: %v", errUpdate)
	}

	res, errGet := h.db.SpeakingGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get speaking after stop. err: %v", errGet)
		return spk, nil
	}

	return res, nil
}

// Delete soft-deletes a speaking record.
// Note: callers should call Stop (pod-targeted) before Delete to ensure
// the streaming session is properly terminated.
func (h *speakingHandler) Delete(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"speaking_id": id,
	})

	spk, err := h.db.SpeakingGet(ctx, id)
	if err != nil {
		log.Infof("Could not get speaking. err: %v", err)
		return nil, fmt.Errorf("speaking not found: %v", err)
	}
	log.WithField("speaking", spk).Debugf("Retrieved speaking. speaking_id: %s", id)

	if errDelete := h.db.SpeakingDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete speaking. err: %v", errDelete)
		return nil, fmt.Errorf("could not delete speaking: %v", errDelete)
	}

	res, errGet := h.db.SpeakingGet(ctx, id)
	if errGet != nil {
		log.Errorf("Could not get speaking after delete. err: %v", errGet)
		return spk, nil
	}

	return res, nil
}
