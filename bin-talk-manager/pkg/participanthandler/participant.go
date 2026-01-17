package participanthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-talk-manager/models/participant"
)

// ParticipantAdd adds a participant to a talk
// Uses UPSERT behavior - if participant already exists, updates it
func (h *participantHandler) ParticipantAdd(ctx context.Context, customerID, chatID, ownerID uuid.UUID, ownerType string) (*participant.Participant, error) {
	log := log.WithFields(log.Fields{
		"func":        "ParticipantAdd",
		"customer_id": customerID,
		"chat_id":     chatID,
		"owner_id":    ownerID,
		"owner_type":  ownerType,
	})
	log.Debug("Adding participant")

	// Validate inputs
	if customerID == uuid.Nil {
		log.Error("Invalid customer_id: cannot be nil")
		return nil, fmt.Errorf("customer_id is required")
	}
	if chatID == uuid.Nil {
		log.Error("Invalid chat_id: cannot be nil")
		return nil, fmt.Errorf("chat_id is required")
	}
	if ownerID == uuid.Nil {
		log.Error("Invalid owner_id: cannot be nil")
		return nil, fmt.Errorf("owner_id is required")
	}
	if ownerType == "" {
		log.Error("Invalid owner_type: cannot be empty")
		return nil, fmt.Errorf("owner_type is required")
	}

	// Generate new participant ID
	participantID, err := uuid.NewV7()
	if err != nil {
		log.Errorf("Failed to generate participant ID. err: %v", err)
		return nil, fmt.Errorf("failed to generate participant ID: %w", err)
	}

	// Create participant object
	p := &participant.Participant{
		Identity: commonidentity.Identity{
			ID:         participantID,
			CustomerID: customerID,
		},
		ChatID: chatID,
		Owner: commonidentity.Owner{
			OwnerID:   ownerID,
			OwnerType: commonidentity.OwnerType(ownerType),
		},
	}

	// Create in database (UPSERT behavior)
	err = h.dbHandler.ParticipantCreate(ctx, p)
	if err != nil {
		log.Errorf("Failed to create participant. err: %v", err)
		return nil, fmt.Errorf("failed to create participant: %w", err)
	}

	// Augment log with result before final log
	log = log.WithField("participant_id", participantID)
	log.Info("Participant added successfully")

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, customerID, participant.EventParticipantAdded, p)

	return p, nil
}

// ParticipantList returns all participants for a talk
func (h *participantHandler) ParticipantList(ctx context.Context, customerID, chatID uuid.UUID) ([]*participant.Participant, error) {
	log := log.WithFields(log.Fields{
		"func":        "ParticipantList",
		"customer_id": customerID,
		"chat_id":     chatID,
	})
	log.Debug("Listing participants")

	// Validate inputs
	if customerID == uuid.Nil {
		log.Error("Invalid customer_id: cannot be nil")
		return nil, fmt.Errorf("customer_id is required")
	}
	if chatID == uuid.Nil {
		log.Error("Invalid chat_id: cannot be nil")
		return nil, fmt.Errorf("chat_id is required")
	}

	// Build filters
	filters := map[participant.Field]any{
		participant.FieldCustomerID: customerID,
		participant.FieldChatID:     chatID,
	}

	// Get participants from database
	participants, err := h.dbHandler.ParticipantList(ctx, filters)
	if err != nil {
		log.Errorf("Failed to list participants. err: %v", err)
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}

	// Augment log with result before final log
	log = log.WithField("count", len(participants))
	log.Info("Participants listed successfully")

	return participants, nil
}

// ParticipantRemove removes a participant from a talk (hard delete)
func (h *participantHandler) ParticipantRemove(ctx context.Context, customerID, participantID uuid.UUID) error {
	log := log.WithFields(log.Fields{
		"func":           "ParticipantRemove",
		"customer_id":    customerID,
		"participant_id": participantID,
	})
	log.Debug("Removing participant")

	// Validate inputs
	if customerID == uuid.Nil {
		log.Error("Invalid customer_id: cannot be nil")
		return fmt.Errorf("customer_id is required")
	}
	if participantID == uuid.Nil {
		log.Error("Invalid participant_id: cannot be nil")
		return fmt.Errorf("participant_id is required")
	}

	// Retrieve participant before deletion (for webhook payload)
	p, err := h.dbHandler.ParticipantGet(ctx, participantID)
	if err != nil {
		log.Errorf("Failed to get participant for deletion. err: %v", err)
		return fmt.Errorf("failed to get participant: %w", err)
	}

	// Delete from database (hard delete)
	err = h.dbHandler.ParticipantDelete(ctx, participantID)
	if err != nil {
		log.Errorf("Failed to delete participant. err: %v", err)
		return fmt.Errorf("failed to delete participant: %w", err)
	}

	log.Info("Participant removed successfully")

	// Publish webhook event
	h.notifyHandler.PublishWebhookEvent(ctx, customerID, participant.EventParticipantRemoved, p)

	return nil
}
