package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
)

const defaultMaxExternalMediaPerConfbridge = 5

// ExternalMediaStart starts the external media processing
func (h *confbridgeHandler) ExternalMediaStart(
	ctx context.Context,
	id uuid.UUID,
	externalMediaID uuid.UUID,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	connectionType string,
	format string,
) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExternalMediaStart",
		"confbridge_id":     id,
		"external_media_id": externalMediaID,
	})
	log.Debug("Starting the external media.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, err
	}

	if len(c.ExternalMediaIDs) >= defaultMaxExternalMediaPerConfbridge {
		log.Errorf("The confbridge has reached the maximum number of external medias. count: %d", len(c.ExternalMediaIDs))
		return nil, fmt.Errorf("the confbridge has reached the maximum number of external medias")
	}

	tmp, err := h.externalMediaHandler.Start(
		ctx,
		externalMediaID,
		externalmedia.ReferenceTypeConfbridge,
		c.ID,
		externalHost,
		encapsulation,
		transport,
		connectionType,
		format,
		externalmedia.DirectionBoth,
		externalmedia.DirectionBoth,
	)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", tmp).Debugf("Started external media. external_media_id: %s", tmp.ID)

	res, err := h.AddExternalMediaID(ctx, id, tmp.ID)
	if err != nil {
		log.Errorf("Could not update the external media id. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ExternalMediaStop stops a specific external media on the confbridge
func (h *confbridgeHandler) ExternalMediaStop(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "ExternalMediaStop",
		"confbridge_id":     id,
		"external_media_id": externalMediaID,
	})
	log.Debug("Stopping the external media.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, err
	}

	found := false
	for _, emID := range c.ExternalMediaIDs {
		if emID == externalMediaID {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("The external media id is not in the confbridge's external media ids. external_media_id: %s", externalMediaID)
		return nil, fmt.Errorf("the external media id is not associated with this confbridge")
	}

	tmp, err := h.externalMediaHandler.Stop(ctx, externalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media handler. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)

	// externalMediaHandler.Stop already removes the ID from the parent's ExternalMediaIDs array,
	// so we just need to re-fetch the updated confbridge.
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated confbridge info. err: %v", err)
		return nil, err
	}

	return res, nil
}
