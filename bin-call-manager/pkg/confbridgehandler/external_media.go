package confbridgehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
)

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

	if c.ExternalMediaID != uuid.Nil {
		log.Errorf("The confbridge has external media already. external_media_id: %s", c.ExternalMediaID)
		return nil, fmt.Errorf("the confbridge has external media already")
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

	res, err := h.UpdateExternalMediaID(ctx, id, tmp.ID)
	if err != nil {
		log.Errorf("Could not update the external media id. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ExternalMediaStop stops the external media processing
func (h *confbridgeHandler) ExternalMediaStop(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ExternalMediaStop",
		"confbridge_id": id,
	})
	log.Debug("Stopping the external media.")

	// get confbridge
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge info. err: %v", err)
		return nil, err
	}

	if c.ExternalMediaID == uuid.Nil {
		log.Errorf("The confbridge has no external media id. confbridge_id: %s", c.ID)
		return nil, fmt.Errorf("the confbridge has no external media id")
	}

	tmp, err := h.externalMediaHandler.Stop(ctx, c.ExternalMediaID)
	if err != nil {
		log.Errorf("Could not stop the external media handler. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)

	// update
	res, err := h.UpdateExternalMediaID(ctx, id, uuid.Nil)
	if err != nil {
		log.Errorf("Coudl not update the external media to empty. err: %v", err)
		return nil, err
	}

	return res, nil
}
