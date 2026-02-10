package extensiondirecthandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-registrar-manager/models/extensiondirect"
)

const (
	hashLength = 6 // 6 bytes = 12 hex chars
	maxRetries = 3
)

// generateHash generates a random 12-character hex string
func generateHash() (string, error) {
	b := make([]byte, hashLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("could not generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Create creates a new extension direct with a generated hash
func (h *extensionDirectHandler) Create(ctx context.Context, customerID, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Create",
		"customer_id":  customerID,
		"extension_id": extensionID,
	})

	// check if already exists
	existing, err := h.db.ExtensionDirectGetByExtensionID(ctx, extensionID)
	if err == nil && existing != nil {
		log.Debugf("Extension direct already exists. extension_id: %s", extensionID)
		return existing, nil
	}

	id := h.utilHandler.UUIDCreate()

	for i := 0; i < maxRetries; i++ {
		hash, err := generateHash()
		if err != nil {
			return nil, fmt.Errorf("could not generate hash: %w", err)
		}

		ed := &extensiondirect.ExtensionDirect{
			Identity: commonidentity.Identity{
				ID:         id,
				CustomerID: customerID,
			},
			ExtensionID: extensionID,
			Hash:        hash,
		}

		if err := h.db.ExtensionDirectCreate(ctx, ed); err != nil {
			log.Debugf("Could not create extension direct (attempt %d). err: %v", i+1, err)
			continue
		}

		res, err := h.db.ExtensionDirectGet(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not get created extension direct: %w", err)
		}

		return res, nil
	}

	return nil, fmt.Errorf("could not create extension direct after %d attempts", maxRetries)
}

// Delete deletes extension direct
func (h *extensionDirectHandler) Delete(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Delete",
		"id":   id,
	})

	if err := h.db.ExtensionDirectDelete(ctx, id); err != nil {
		log.Errorf("Could not delete extension direct. err: %v", err)
		return nil, err
	}

	res, err := h.db.ExtensionDirectGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted extension direct. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns extension direct by ID
func (h *extensionDirectHandler) Get(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGet(ctx, id)
}

// GetByExtensionID returns extension direct by extension ID
func (h *extensionDirectHandler) GetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGetByExtensionID(ctx, extensionID)
}

// GetByExtensionIDs returns extension directs by extension IDs
func (h *extensionDirectHandler) GetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGetByExtensionIDs(ctx, extensionIDs)
}

// GetByHash returns extension direct by hash
func (h *extensionDirectHandler) GetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error) {
	return h.db.ExtensionDirectGetByHash(ctx, hash)
}

// Regenerate generates a new hash for an existing extension direct
func (h *extensionDirectHandler) Regenerate(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Regenerate",
		"id":   id,
	})

	for i := 0; i < maxRetries; i++ {
		hash, err := generateHash()
		if err != nil {
			return nil, fmt.Errorf("could not generate hash: %w", err)
		}

		fields := map[extensiondirect.Field]any{
			extensiondirect.FieldHash: hash,
		}

		if err := h.db.ExtensionDirectUpdate(ctx, id, fields); err != nil {
			log.Debugf("Could not update extension direct hash (attempt %d). err: %v", i+1, err)
			continue
		}

		res, err := h.db.ExtensionDirectGet(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("could not get updated extension direct: %w", err)
		}

		return res, nil
	}

	return nil, fmt.Errorf("could not regenerate hash after %d attempts", maxRetries)
}
