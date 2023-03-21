package campaignhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// EventHandleActiveflowDeleted handles activeflow's deleted event.
func (h *campaignHandler) EventHandleActiveflowDeleted(ctx context.Context, campaignID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventHandleActiveflowDeleted",
		"campaign_id": campaignID,
	})

	// check is the campaign stopable
	if !h.isStoppable(ctx, campaignID) {
		return nil
	}
	log.Debugf("The campaign is able to stop. Stop the campaign. campaign_id: %s", campaignID)

	c, err := h.campaignStopNow(ctx, campaignID)
	if err != nil {
		log.Errorf("Could not stop the campaign. err: %v", err)
		return err
	}
	log.WithField("campaign", c).Debugf("Stopped campaign. campaign_id: %s", c.ID)

	return nil
}

// EventhandleReferenceCallHungup handles reference call's hangup.
func (h *campaignHandler) EventHandleReferenceCallHungup(ctx context.Context, campaignID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventHandleReferenceCallHungup",
		"campaign_id": campaignID,
	})

	// check is the campaign stopable
	if !h.isStoppable(ctx, campaignID) {
		return nil
	}
	log.Debugf("The campaign is able to stop. Stop the campaign. campaign_id: %s", campaignID)

	c, err := h.campaignStopNow(ctx, campaignID)
	if err != nil {
		log.Errorf("Could not stop the campaign. err: %v", err)
		return err
	}
	log.WithField("campaign", c).Debugf("Stopped campaign. campaign_id: %s", c.ID)

	return nil
}
