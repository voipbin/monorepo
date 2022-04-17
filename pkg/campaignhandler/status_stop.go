package campaignhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
)

// StatusStopping stopping the campaign
func (h *campaignHandler) StatusStopping(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "StatusStopping",
			"id":   id,
		})
	log.Debug("Running campaign.")

	// get dialing calls.

	return nil, nil
}
