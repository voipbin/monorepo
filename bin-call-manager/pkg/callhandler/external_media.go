package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/externalmedia"
)

// list of channel variables
const (
	ChannelValiableExternalMediaLocalPort    = "UNICASTRTP_LOCAL_PORT"
	ChannelValiableExternalMediaLocalAddress = "UNICASTRTP_LOCAL_ADDRESS"
)

// ExternalMediaStart starts the external media processing
func (h *callHandler) ExternalMediaStart(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ExternalMediaStart",
		"call_id":       id,
		"external_host": externalHost,
	})
	log.Debug("Starting the external media.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if c.ExternalMediaID != uuid.Nil {
		log.Errorf("The call has external media already. external_media_id: %s", c.ExternalMediaID)
		return nil, fmt.Errorf("the call has external media already")
	}

	tmp, err := h.externalMediaHandler.Start(ctx, externalMediaID, externalmedia.ReferenceTypeCall, c.ID, true, externalHost, encapsulation, transport, connectionType, format, direction)
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
func (h *callHandler) ExternalMediaStop(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "ExternalMediaStop",
		"call_id": id,
	})
	log.Debug("Stopping the external media.")

	// get call
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if c.ExternalMediaID == uuid.Nil {
		log.Errorf("The call has no external media id. call_id: %s", c.ID)
		return nil, fmt.Errorf("the call has no external media id")
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
