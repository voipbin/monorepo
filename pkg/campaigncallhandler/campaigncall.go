package campaigncallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// Create creates a new campaigncall
func (h *campaigncallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	outdialTargetID uuid.UUID,
	queueID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType campaigncall.ReferenceType,
	referenceID uuid.UUID,
	source *cmaddress.Address,
	destination *cmaddress.Address,
	destinationIndex int,
	tryCount int,
) (*campaigncall.Campaigncall, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"customer_id": customerID,
		})

	ts := dbhandler.GetCurTime()
	id := uuid.Must(uuid.NewV4())
	t := &campaigncall.Campaigncall{
		ID:               id,
		CustomerID:       customerID,
		CampaignID:       campaignID,
		OutplanID:        outplanID,
		OutdialID:        outdialID,
		OutdialTargetID:  outdialTargetID,
		QueueID:          queueID,
		ActiveflowID:     activeflowID,
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		Status:           campaigncall.StatusDialing,
		Source:           source,
		Destination:      destination,
		DestinationIndex: destinationIndex,
		TryCount:         tryCount,
		TMCreate:         ts,
		TMUpdate:         ts,
	}
	log.WithField("campaigncall", t).Debug("Creating a new campaigncall.")

	if err := h.db.CampaigncallCreate(ctx, t); err != nil {
		log.Errorf("Could not create the campaigncall. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created campaigncall. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaigncall.EventTypeCampaigncallCreated, res)

	return res, nil
}

// Get returns list of campaigncall
func (h *campaigncallHandler) Get(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Get",
			"id":   id,
		})
	log.Debug("Getting campaigncall.")

	res, err := h.db.CampaigncallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaigncall. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByCampaignID returns list of campaigncall
func (h *campaigncallHandler) GetsByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCampaignID",
			"campaign_id": campaignID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting campaigncalls.")

	res, err := h.db.CampaigncallGetsByCampaignID(ctx, campaignID, token, limit)
	if err != nil {
		log.Errorf("Could not get campaigncalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByCampaignIDAndStatus returns list of campaigncalls
func (h *campaigncallHandler) GetsByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCampaignID",
			"campaign_id": campaignID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting campaigncalls.")

	res, err := h.db.CampaigncallGetsByCampaignIDAndStatus(ctx, campaignID, status, token, limit)
	if err != nil {
		log.Errorf("Could not get GetsByCampaignIDAndStatus. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateStatus updates the status
func (h *campaigncallHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "GetsByCampaignID",
			"id":   id,
		})
	log.Debug("Getting campaigncalls.")

	if err := h.db.CampaigncallUpdateStatus(ctx, id, status); err != nil {
		log.Errorf("Could not get UpdateStatus. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaigncall. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaigncall.EventTypeCampaigncallUpdated, res)

	return res, nil
}
