package outdialhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

// Create creates a new outdial
func (h *outdialHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	campaignID uuid.UUID,
	name string,
	detail string,
	data string,
) (*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"customer_id": customerID,
		})

	id := uuid.Must(uuid.NewV4())
	t := &outdial.Outdial{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		CampaignID: campaignID,

		Name:   name,
		Detail: detail,

		Data: data,

		TMCreate: dbhandler.GetCurTime(),
		TMUpdate: dbhandler.DefaultTimeStamp,
		TMDelete: dbhandler.DefaultTimeStamp,
	}
	log.WithField("outdial", t).Debug("Creating a new outdial.")

	if err := h.db.OutdialCreate(ctx, t); err != nil {
		log.Errorf("Could not create the outdial. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created outdial. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, outdial.EventTypeOutdialCreated, res)

	return res, nil
}

// Delete deletes outdial
func (h *outdialHandler) Delete(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Delete",
			"outdial_id": id,
		})

	if errDel := h.db.OutdialDelete(ctx, id); errDel != nil {
		log.Errorf("Could not delete the outdial. err: %v", errDel)
		return nil, errDel
	}

	res, err := h.db.OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted outdial. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, outdial.EventTypeOutdialDeleted, res)

	return res, nil
}

// Get returns outdial
func (h *outdialHandler) Get(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Get",
			"outdial_id": id,
		})
	res, err := h.db.OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Gets returns list of outdials with filters
func (h *outdialHandler) Gets(ctx context.Context, token string, limit uint64, filters map[outdial.Field]any) ([]*outdial.Outdial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"token":   token,
		"limit":   limit,
		"filters": filters,
	})
	log.Debug("Getting outdials with filters.")

	res, err := h.db.OutdialGets(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get outdials. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByCustomerID returns list of outdials
func (h *outdialHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCustomerID",
			"customer_id": customerID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting outdials.")

	filters := map[outdial.Field]any{
		outdial.FieldCustomerID: customerID,
		outdial.FieldDeleted:    false,
	}

	res, err := h.db.OutdialGets(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get outdials. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateBasicInfo updates outdial's basic info
func (h *outdialHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "UpdateBasicInfo",
			"id":     id,
			"name":   name,
			"detail": detail,
		})
	log.Debug("Updating outdial basic info.")

	fields := map[outdial.Field]any{
		outdial.FieldName:   name,
		outdial.FieldDetail: detail,
	}

	if err := h.db.OutdialUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update outdials. err: %v", err)
		return nil, err
	}

	// get updated outdial
	res, err := h.db.OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outdial info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, outdial.EventTypeOutdialUpdated, res)

	return res, nil
}

// UpdateCampaignID updates outdial's campaignID info
func (h *outdialHandler) UpdateCampaignID(ctx context.Context, id, campaignID uuid.UUID) (*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "UpdateCampaignID",
			"id":          id,
			"campaign_id": campaignID,
		})
	log.Debug("updating outdial campaign info.")

	fields := map[outdial.Field]any{
		outdial.FieldCampaignID: campaignID,
	}

	if err := h.db.OutdialUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update outdials. err: %v", err)
		return nil, err
	}

	// get updated outdial
	res, err := h.db.OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outdial info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, outdial.EventTypeOutdialUpdated, res)

	return res, nil
}

// UpdateData updates outdial's data info
func (h *outdialHandler) UpdateData(ctx context.Context, id uuid.UUID, data string) (*outdial.Outdial, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "UpdateData",
			"id":   id,
			"data": data,
		})
	log.Debug("updating outdial data info.")

	fields := map[outdial.Field]any{
		outdial.FieldData: data,
	}

	if err := h.db.OutdialUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update outdials. err: %v", err)
		return nil, err
	}

	// get updated outdial
	res, err := h.db.OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outdial info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, outdial.EventTypeOutdialUpdated, res)

	return res, nil
}
